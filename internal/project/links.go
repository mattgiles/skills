package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mattgiles/skills/internal/discovery"
)

type desiredLink struct {
	ManagedLink
}

type linkAction struct {
	Link   desiredLink
	Status string
}

func buildSkillLinkReports(resolvedSources map[string]*resolvedSource, manifest Manifest, skillsDir string, stateLinks map[string]ManagedLink) ([]desiredLink, []LinkReport) {
	desired := make([]desiredLink, 0, len(manifest.Skills))
	reports := make([]LinkReport, 0, len(manifest.Skills))

	for _, skill := range manifest.Skills {
		src := resolvedSources[skill.Source]
		report := LinkReport{
			Source: skill.Source,
			Skill:  skill.Name,
			Path:   filepath.Join(skillsDir, skill.Name),
		}

		switch {
		case src == nil:
			report.Status = "unknown-source"
			report.Message = "source is not declared"
		case strings.TrimSpace(src.DesiredCommit) == "":
			report.Status = "source-not-ready"
		case strings.TrimSpace(src.InspectError) != "":
			report.Status = "inspect-failed"
			report.Message = src.InspectError
		case strings.TrimSpace(skill.Path) != "":
			selector := discovery.NormalizeSkillPath(skill.Path)
			match, ok := src.SkillsByPath[selector]
			if !ok {
				report.Status = "missing-skill"
				report.Message = missingPathMessage(src, skill.Path, selector)
				break
			}
			desired = appendResolvedLink(desired, &report, src, skill, match, stateLinks)
		default:
			matches := src.SkillsByName[skill.Name]
			switch {
			case len(matches) == 0:
				report.Status = "missing-skill"
				report.Message = missingNameMessage(src, skill.Name)
			case len(matches) > 1:
				report.Status = "ambiguous-skill"
				report.Message = ambiguousMessage(matches)
			default:
				desired = appendResolvedLink(desired, &report, src, skill, matches[0], stateLinks)
			}
		}

		reports = append(reports, report)
	}

	sortLinkReports(reports)
	return desired, reports
}

// appendResolvedLink fills report.Target/Status for a resolved skill and
// appends the corresponding desired link. report.Path and the link's Skill
// label always derive from skill.Name so state and adapters are unaffected
// by which discovered directory was selected.
func appendResolvedLink(desired []desiredLink, report *LinkReport, src *resolvedSource, skill ManifestSkill, match discovery.DiscoveredSkill, stateLinks map[string]ManagedLink) []desiredLink {
	target := skillTargetPath(src.WorktreePath, match.RelativePath)
	report.Target = target
	report.Status = currentLinkStatus(report.Path, target, stateLinks)
	return append(desired, desiredLink{
		ManagedLink: ManagedLink{
			Path:   report.Path,
			Target: target,
			Source: skill.Source,
			Skill:  skill.Name,
		},
	})
}

// missingPathMessage explains why a path: selector matched nothing —
// either the directory was discovered but filtered out by the source scope,
// or it does not exist at all.
func missingPathMessage(src *resolvedSource, rawPath string, selector string) string {
	for _, excluded := range src.ExcludedSkills {
		if discovery.NormalizeSkillPath(excluded.RelativePath) == selector {
			return fmt.Sprintf("skill path %q is excluded by the source's include/exclude scope", rawPath)
		}
	}
	return fmt.Sprintf("no skill directory at path %q", rawPath)
}

// missingNameMessage explains a name-based miss when same-named skills exist
// only outside the source scope; otherwise it returns "" (unchanged behavior).
func missingNameMessage(src *resolvedSource, name string) string {
	paths := make([]string, 0)
	for _, excluded := range src.ExcludedSkills {
		if excluded.Name == name {
			paths = append(paths, filepath.ToSlash(excluded.RelativePath))
		}
	}
	if len(paths) == 0 {
		return ""
	}
	sort.Strings(paths)
	return "skill directory exists only outside the source's include/exclude scope: " + strings.Join(paths, ", ")
}

// ambiguousMessage lists the candidate paths for a name shared by several
// in-scope skills so the user can pick one with path:.
func ambiguousMessage(matches []discovery.DiscoveredSkill) string {
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		paths = append(paths, filepath.ToSlash(match.RelativePath))
	}
	sort.Strings(paths)
	return "multiple skills share this directory name; set path: to one of: " + strings.Join(paths, ", ")
}

func buildClaudeLinkReports(skillLinks []desiredLink, claudeSkillsDir string, stateLinks map[string]ManagedLink) ([]desiredLink, []LinkReport) {
	desired := make([]desiredLink, 0, len(skillLinks))
	reports := make([]LinkReport, 0, len(skillLinks))

	for _, skillLink := range skillLinks {
		report := LinkReport{
			Source: skillLink.Source,
			Skill:  skillLink.Skill,
			Path:   filepath.Join(claudeSkillsDir, skillLink.Skill),
			Target: skillLink.Path,
		}
		report.Status = currentLinkStatus(report.Path, report.Target, stateLinks)
		desired = append(desired, desiredLink{
			ManagedLink: ManagedLink{
				Path:   report.Path,
				Target: report.Target,
				Source: skillLink.Source,
				Skill:  skillLink.Skill,
			},
		})
		reports = append(reports, report)
	}

	sortLinkReports(reports)
	return desired, reports
}

func validateDesiredLinks(links []desiredLink) error {
	seen := map[string]struct{}{}
	for _, link := range links {
		if _, ok := seen[link.Path]; ok {
			return fmt.Errorf("multiple skills want the same destination path: %s", link.Path)
		}
		seen[link.Path] = struct{}{}
	}
	return nil
}

func planLinkActions(links []desiredLink, stateLinks map[string]ManagedLink) ([]linkAction, error) {
	actions := make([]linkAction, 0, len(links))
	for _, link := range links {
		action, err := plannedLinkStatus(link, stateLinks)
		if err != nil {
			return nil, err
		}
		actions = append(actions, linkAction{Link: link, Status: action})
	}
	return actions, nil
}

func plannedLinkStatus(link desiredLink, stateLinks map[string]ManagedLink) (string, error) {
	info, err := os.Lstat(link.Path)
	if errors.Is(err, os.ErrNotExist) {
		return "created", nil
	}
	if err != nil {
		return "", err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return "", fmt.Errorf("destination exists and is not a symlink: %s", link.Path)
	}

	currentTarget, err := os.Readlink(link.Path)
	if err != nil {
		return "", err
	}
	if currentTarget == link.Target {
		return "linked", nil
	}

	if _, ok := stateLinks[link.Path]; !ok {
		return "", fmt.Errorf("destination is an unmanaged symlink: %s", link.Path)
	}

	return "updated", nil
}

func applyLinkActions(actions []linkAction) error {
	for _, action := range actions {
		switch action.Status {
		case "linked":
			continue
		case "created":
			if err := createSymlink(action.Link); err != nil {
				return err
			}
		case "updated":
			if err := replaceSymlink(action.Link); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected link action %q", action.Status)
		}
	}
	return nil
}

func updateLinkReportsForActions(reports []LinkReport, actions []linkAction, dryRun bool) {
	actionByPath := map[string]string{}
	for _, action := range actions {
		actionByPath[action.Link.Path] = action.Status
	}
	for i := range reports {
		if planned, ok := actionByPath[reports[i].Path]; ok {
			if dryRun {
				reports[i].Status = dryRunLinkStatus(planned)
			} else {
				reports[i].Status = planned
			}
		}
	}
}

func createSymlink(link desiredLink) error {
	if err := os.MkdirAll(filepath.Dir(link.Path), 0o755); err != nil {
		return err
	}
	return os.Symlink(link.Target, link.Path)
}

func replaceSymlink(link desiredLink) error {
	if err := os.Remove(link.Path); err != nil {
		return err
	}
	return os.Symlink(link.Target, link.Path)
}

func removeManagedLinks(links []ManagedLink) error {
	for _, link := range links {
		if err := removeManagedLink(link); err != nil {
			return err
		}
	}
	return nil
}

func removeManagedLink(link ManagedLink) error {
	info, err := os.Lstat(link.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("managed link path is no longer a symlink: %s", link.Path)
	}
	return os.Remove(link.Path)
}

func staleLinks(current []ManagedLink, desired map[string]desiredLink) []ManagedLink {
	stale := make([]ManagedLink, 0)
	for _, link := range current {
		if _, ok := desired[link.Path]; ok {
			continue
		}
		stale = append(stale, link)
	}
	return stale
}

func currentLinkStatus(path string, target string, stateLinks map[string]ManagedLink) string {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "missing"
	}
	if err != nil {
		return "invalid"
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "conflict"
	}

	currentTarget, err := os.Readlink(path)
	if err != nil {
		return "invalid"
	}
	if currentTarget == target {
		return "linked"
	}
	if _, ok := stateLinks[path]; ok {
		return "stale"
	}
	return "conflict"
}

func skillTargetPath(worktreePath string, relativePath string) string {
	if strings.TrimSpace(relativePath) == "" || filepath.Clean(relativePath) == "." {
		return worktreePath
	}
	return filepath.Join(worktreePath, relativePath)
}

func fatalSkillLinkReports(reports []LinkReport) error {
	problems := make([]string, 0)
	for _, report := range reports {
		switch report.Status {
		case "unknown-source", "source-not-ready", "inspect-failed", "missing-skill", "ambiguous-skill", "conflict":
			problem := fmt.Sprintf("%s/%s: %s", report.Source, report.Skill, report.Status)
			if report.Message != "" {
				problem += " (" + report.Message + ")"
			}
			problems = append(problems, problem)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.New(strings.Join(problems, "; "))
}

func fatalAdapterLinkReports(reports []LinkReport) error {
	problems := make([]string, 0)
	for _, report := range reports {
		switch report.Status {
		case "conflict":
			problem := fmt.Sprintf("%s/%s: %s", report.Source, report.Skill, report.Status)
			if report.Message != "" {
				problem += " (" + report.Message + ")"
			}
			problems = append(problems, problem)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.New(strings.Join(problems, "; "))
}
