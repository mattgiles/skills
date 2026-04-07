package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/discovery"
	"github.com/mattgiles/skills/internal/source"
)

const (
	ManifestFilename = ".skills.yaml"
	StateFilename    = ".skills/state.yaml"
)

type Manifest struct {
	Sources map[string]ProjectSource        `yaml:"sources"`
	Agents  map[string]ProjectAgentOverride `yaml:"agents,omitempty"`
	Skills  []ProjectSkill                  `yaml:"skills"`
}

type ProjectSource struct {
	URL string `yaml:"url,omitempty"`
	Ref string `yaml:"ref"`
}

type ProjectAgentOverride struct {
	SkillsDir string `yaml:"skills_dir"`
}

type ProjectSkill struct {
	Source string   `yaml:"source"`
	Name   string   `yaml:"name"`
	Agents []string `yaml:"agents"`
}

type State struct {
	Links []ManagedLink `yaml:"links,omitempty"`
}

type ManagedLink struct {
	Path   string `yaml:"path"`
	Target string `yaml:"target"`
	Source string `yaml:"source"`
	Skill  string `yaml:"skill"`
	Agent  string `yaml:"agent"`
}

type StatusReport struct {
	Sources    []SourceReport
	Links      []LinkReport
	StaleLinks []ManagedLink
}

type SourceReport struct {
	Alias   string
	Ref     string
	Status  string
	Commit  string
	Message string
}

type LinkReport struct {
	Source string
	Skill  string
	Agent  string
	Path   string
	Target string
	Status string
}

type SyncResult struct {
	Sources []SourceReport
	Links   []LinkReport
	Pruned  []string
}

type desiredLink struct {
	ManagedLink
}

type resolvedSource struct {
	Alias        string
	URL          string
	Ref          string
	RepoPath     string
	WorktreeRoot string
	ProjectID    string
	Commit       string
	WorktreePath string
	SkillsByName map[string][]discovery.DiscoveredSkill
}

type resolvedAgent struct {
	Name      string
	SkillsDir string
}

func DefaultManifest() Manifest {
	return Manifest{
		Sources: map[string]ProjectSource{},
		Skills:  []ProjectSkill{},
	}
}

func ManifestPath(projectDir string) string {
	return filepath.Join(projectDir, ManifestFilename)
}

func StatePath(projectDir string) string {
	return filepath.Join(projectDir, StateFilename)
}

func LoadManifest(projectDir string) (Manifest, error) {
	path := ManifestPath(projectDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, fmt.Errorf("manifest not found: %s", path)
		}
		return Manifest{}, err
	}

	manifest := DefaultManifest()
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	ensureManifestDefaults(&manifest)
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func SaveManifest(projectDir string, manifest Manifest) error {
	ensureManifestDefaults(&manifest)
	if err := ValidateManifest(manifest); err != nil {
		return err
	}

	path := ManifestPath(projectDir)
	data, err := yaml.Marshal(&manifest)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func LoadState(projectDir string) (State, error) {
	path := StatePath(projectDir)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}

	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse state %s: %w", path, err)
	}
	return state, nil
}

func SaveState(projectDir string, state State) error {
	path := StatePath(projectDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ValidateManifest(manifest Manifest) error {
	ensureManifestDefaults(&manifest)

	for alias, src := range manifest.Sources {
		if err := config.ValidateAlias(alias); err != nil {
			return err
		}
		if strings.TrimSpace(src.Ref) == "" {
			return fmt.Errorf("source %q is missing ref", alias)
		}
	}

	for agent, override := range manifest.Agents {
		if err := config.ValidateAlias(agent); err != nil {
			return err
		}
		if strings.TrimSpace(override.SkillsDir) == "" {
			return fmt.Errorf("agent override %q is missing skills_dir", agent)
		}
	}

	seenSkills := map[string]struct{}{}
	for _, skill := range manifest.Skills {
		if strings.TrimSpace(skill.Source) == "" {
			return errors.New("skill is missing source")
		}
		if strings.TrimSpace(skill.Name) == "" {
			return fmt.Errorf("skill in source %q is missing name", skill.Source)
		}
		if _, ok := manifest.Sources[skill.Source]; !ok {
			return fmt.Errorf("skill %q references unknown source %q", skill.Name, skill.Source)
		}
		if len(skill.Agents) == 0 {
			return fmt.Errorf("skill %q in source %q has no agents", skill.Name, skill.Source)
		}

		key := skill.Source + "\x00" + skill.Name
		if _, ok := seenSkills[key]; ok {
			return fmt.Errorf("duplicate skill declaration for %s/%s", skill.Source, skill.Name)
		}
		seenSkills[key] = struct{}{}

		seenAgents := map[string]struct{}{}
		for _, agent := range skill.Agents {
			if err := config.ValidateAlias(agent); err != nil {
				return err
			}
			if _, ok := seenAgents[agent]; ok {
				return fmt.Errorf("skill %q in source %q repeats agent %q", skill.Name, skill.Source, agent)
			}
			seenAgents[agent] = struct{}{}
		}
	}

	return nil
}

func ProjectID(projectDir string) (string, error) {
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256([]byte(absProjectDir))
	hash := hex.EncodeToString(sum[:])[:12]
	base := sanitizeIDComponent(filepath.Base(absProjectDir))
	if base == "" {
		base = "project"
	}

	return base + "-" + hash, nil
}

func Status(ctx context.Context, projectDir string, cfg config.Config) (StatusReport, error) {
	manifest, err := LoadManifest(projectDir)
	if err != nil {
		return StatusReport{}, err
	}

	state, err := LoadState(projectDir)
	if err != nil {
		return StatusReport{}, err
	}

	worktreeRoot, err := config.WorktreeRootPath(cfg)
	if err != nil {
		return StatusReport{}, err
	}

	resolvedSources, agents, err := resolveInputs(projectDir, cfg, manifest, worktreeRoot)
	if err != nil {
		return StatusReport{}, err
	}

	report := StatusReport{
		Sources: make([]SourceReport, 0, len(resolvedSources)),
		Links:   make([]LinkReport, 0),
	}

	desiredLinks, sourceReports, linkReports, err := planLinks(ctx, resolvedSources, agents, manifest, false)
	if err != nil {
		return StatusReport{}, err
	}
	report.Sources = append(report.Sources, sourceReports...)
	report.Links = append(report.Links, linkReports...)

	desiredByPath := map[string]desiredLink{}
	for _, link := range desiredLinks {
		desiredByPath[link.Path] = link
	}

	for _, stale := range staleLinks(state, desiredByPath) {
		report.StaleLinks = append(report.StaleLinks, stale)
	}

	sortSourceReports(report.Sources)
	sortLinkReports(report.Links)
	sort.Slice(report.StaleLinks, func(i, j int) bool {
		return report.StaleLinks[i].Path < report.StaleLinks[j].Path
	})

	return report, nil
}

func Sync(ctx context.Context, projectDir string, cfg config.Config) (SyncResult, error) {
	manifest, err := LoadManifest(projectDir)
	if err != nil {
		return SyncResult{}, err
	}

	state, err := LoadState(projectDir)
	if err != nil {
		return SyncResult{}, err
	}

	worktreeRoot, err := config.WorktreeRootPath(cfg)
	if err != nil {
		return SyncResult{}, err
	}

	resolvedSources, agents, err := resolveInputs(projectDir, cfg, manifest, worktreeRoot)
	if err != nil {
		return SyncResult{}, err
	}

	desiredLinks, sourceReports, linkReports, err := planLinks(ctx, resolvedSources, agents, manifest, true)
	if err != nil {
		return SyncResult{}, err
	}

	desiredByPath := map[string]desiredLink{}
	for _, link := range desiredLinks {
		if _, ok := desiredByPath[link.Path]; ok {
			return SyncResult{}, fmt.Errorf("multiple skills want the same destination path: %s", link.Path)
		}
		desiredByPath[link.Path] = link
	}

	stateByPath := map[string]ManagedLink{}
	for _, link := range state.Links {
		stateByPath[link.Path] = link
	}

	prune := staleLinks(state, desiredByPath)
	if err := validateLinkPlan(desiredLinks, prune, stateByPath); err != nil {
		return SyncResult{}, err
	}

	for _, src := range resolvedSources {
		if _, err := source.EnsureWorktree(ctx, source.Source{
			Alias:    src.Alias,
			URL:      src.URL,
			RepoPath: src.RepoPath,
		}, src.WorktreePath, src.Commit); err != nil {
			return SyncResult{}, err
		}
	}

	for i, link := range desiredLinks {
		action, err := applyDesiredLink(link, stateByPath)
		if err != nil {
			return SyncResult{}, err
		}
		linkReports[i].Status = action
	}

	prunedPaths := make([]string, 0, len(prune))
	for _, stale := range prune {
		if err := removeManagedLink(stale); err != nil {
			return SyncResult{}, err
		}
		prunedPaths = append(prunedPaths, stale.Path)
	}

	newState := State{Links: make([]ManagedLink, 0, len(desiredLinks))}
	for _, link := range desiredLinks {
		newState.Links = append(newState.Links, link.ManagedLink)
	}
	if err := SaveState(projectDir, newState); err != nil {
		return SyncResult{}, err
	}

	sortSourceReports(sourceReports)
	sortLinkReports(linkReports)
	sort.Strings(prunedPaths)

	return SyncResult{
		Sources: sourceReports,
		Links:   linkReports,
		Pruned:  prunedPaths,
	}, nil
}

func resolveInputs(projectDir string, cfg config.Config, manifest Manifest, worktreeRoot string) (map[string]*resolvedSource, map[string]resolvedAgent, error) {
	repoRoot, err := config.RepoRootPath(cfg)
	if err != nil {
		return nil, nil, err
	}

	projectID, err := ProjectID(projectDir)
	if err != nil {
		return nil, nil, err
	}

	resolvedSources := map[string]*resolvedSource{}
	for alias, projectSource := range manifest.Sources {
		globalSource, hasGlobal := cfg.Sources[alias]
		switch {
		case hasGlobal && strings.TrimSpace(projectSource.URL) != "" && projectSource.URL != globalSource.URL:
			return nil, nil, fmt.Errorf("source %q has conflicting URLs between global config and project manifest", alias)
		case strings.TrimSpace(projectSource.URL) == "" && !hasGlobal:
			return nil, nil, fmt.Errorf("source %q has no URL in project manifest or global config", alias)
		}

		url := projectSource.URL
		if strings.TrimSpace(url) == "" {
			url = globalSource.URL
		}

		resolvedSources[alias] = &resolvedSource{
			Alias:        alias,
			URL:          url,
			Ref:          projectSource.Ref,
			RepoPath:     source.RepoPath(repoRoot, alias),
			WorktreeRoot: worktreeRoot,
			ProjectID:    projectID,
		}
	}

	agents := map[string]resolvedAgent{}
	for name, cfgAgent := range cfg.Agents {
		path, err := config.ResolvePath("", cfgAgent.SkillsDir)
		if err != nil {
			return nil, nil, err
		}
		agents[name] = resolvedAgent{Name: name, SkillsDir: path}
	}
	for name, override := range manifest.Agents {
		path, err := config.ResolvePath(projectDir, override.SkillsDir)
		if err != nil {
			return nil, nil, err
		}
		agents[name] = resolvedAgent{Name: name, SkillsDir: path}
	}

	for _, skill := range manifest.Skills {
		for _, agent := range skill.Agents {
			if _, ok := agents[agent]; !ok {
				return nil, nil, fmt.Errorf("agent %q is not configured globally or in the project manifest", agent)
			}
		}
	}

	return resolvedSources, agents, nil
}

func planLinks(ctx context.Context, resolvedSources map[string]*resolvedSource, agents map[string]resolvedAgent, manifest Manifest, mutate bool) ([]desiredLink, []SourceReport, []LinkReport, error) {
	sourceReports := make([]SourceReport, 0, len(resolvedSources))

	for _, src := range sortedResolvedSources(resolvedSources) {
		srcStatus := source.Status(ctx, source.Source{
			Alias:    src.Alias,
			URL:      src.URL,
			RepoPath: src.RepoPath,
		})

		report := SourceReport{
			Alias: src.Alias,
			Ref:   src.Ref,
		}

		if !srcStatus.Exists {
			if !mutate {
				report.Status = "missing-source"
				report.Message = "canonical source repo is not cloned"
				sourceReports = append(sourceReports, report)
				continue
			}

			if _, err := source.Sync(ctx, source.Source{
				Alias:    src.Alias,
				URL:      src.URL,
				RepoPath: src.RepoPath,
			}); err != nil {
				return nil, nil, nil, fmt.Errorf("sync source %s: %w", src.Alias, err)
			}
			srcStatus = source.Status(ctx, source.Source{
				Alias:    src.Alias,
				URL:      src.URL,
				RepoPath: src.RepoPath,
			})
		} else if mutate {
			if _, err := source.Sync(ctx, source.Source{
				Alias:    src.Alias,
				URL:      src.URL,
				RepoPath: src.RepoPath,
			}); err != nil {
				return nil, nil, nil, fmt.Errorf("sync source %s: %w", src.Alias, err)
			}
		}

		if !srcStatus.IsGitRepo {
			report.Status = "invalid-source"
			report.Message = srcStatus.LastError
			sourceReports = append(sourceReports, report)
			continue
		}

		commit, err := source.ResolveCommit(ctx, source.Source{
			Alias:    src.Alias,
			URL:      src.URL,
			RepoPath: src.RepoPath,
		}, src.Ref)
		if err != nil {
			report.Status = "invalid-ref"
			report.Message = err.Error()
			sourceReports = append(sourceReports, report)
			continue
		}

		src.Commit = commit
		src.WorktreePath = source.WorktreePath(src.WorktreeRoot, src.ProjectID, src.Alias, commit)

		paths, err := source.ListFilesAtCommit(ctx, source.Source{
			Alias:    src.Alias,
			URL:      src.URL,
			RepoPath: src.RepoPath,
		}, commit)
		if err != nil {
			return nil, nil, nil, err
		}

		skills := discovery.DiscoverFromPaths(src.Alias, src.WorktreePath, paths)
		src.SkillsByName = map[string][]discovery.DiscoveredSkill{}
		for _, skill := range skills {
			src.SkillsByName[skill.Name] = append(src.SkillsByName[skill.Name], skill)
		}

		report.Status = "ready"
		report.Commit = shortCommit(commit)
		sourceReports = append(sourceReports, report)
	}

	links := make([]desiredLink, 0)
	linkReports := make([]LinkReport, 0)

	for _, skill := range manifest.Skills {
		src := resolvedSources[skill.Source]
		for _, agentName := range skill.Agents {
			agent := agents[agentName]
			linkReport := LinkReport{
				Source: skill.Source,
				Skill:  skill.Name,
				Agent:  agentName,
				Path:   filepath.Join(agent.SkillsDir, skill.Name),
			}

			switch {
			case src == nil:
				linkReport.Status = "unknown-source"
			case src.Commit == "":
				linkReport.Status = "source-not-ready"
			default:
				matches := src.SkillsByName[skill.Name]
				if len(matches) == 0 {
					linkReport.Status = "missing-skill"
				} else if len(matches) > 1 {
					linkReport.Status = "ambiguous-skill"
				} else {
					target := filepath.Join(src.WorktreePath, matches[0].RelativePath)
					linkReport.Target = target
					if mutate {
						linkReport.Status = "pending"
					} else {
						linkReport.Status = currentLinkStatus(linkReport.Path, target)
					}
					links = append(links, desiredLink{
						ManagedLink: ManagedLink{
							Path:   linkReport.Path,
							Target: target,
							Source: skill.Source,
							Skill:  skill.Name,
							Agent:  agentName,
						},
					})
				}
			}

			linkReports = append(linkReports, linkReport)
		}
	}

	return links, sourceReports, linkReports, nil
}

func validateLinkPlan(desired []desiredLink, stale []ManagedLink, stateByPath map[string]ManagedLink) error {
	for _, link := range desired {
		info, err := os.Lstat(link.Path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("destination exists and is not a symlink: %s", link.Path)
		}

		currentTarget, err := os.Readlink(link.Path)
		if err != nil {
			return err
		}
		if currentTarget == link.Target {
			continue
		}

		if _, ok := stateByPath[link.Path]; !ok {
			return fmt.Errorf("destination is an unmanaged symlink: %s", link.Path)
		}
	}

	for _, link := range stale {
		info, err := os.Lstat(link.Path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("managed link path is no longer a symlink: %s", link.Path)
		}
	}

	return nil
}

func applyDesiredLink(link desiredLink, stateByPath map[string]ManagedLink) (string, error) {
	if err := os.MkdirAll(filepath.Dir(link.Path), 0o755); err != nil {
		return "", err
	}

	info, err := os.Lstat(link.Path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Symlink(link.Target, link.Path); err != nil {
			return "", err
		}
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
		return "ok", nil
	}

	if _, ok := stateByPath[link.Path]; !ok {
		return "", fmt.Errorf("destination is an unmanaged symlink: %s", link.Path)
	}

	if err := os.Remove(link.Path); err != nil {
		return "", err
	}
	if err := os.Symlink(link.Target, link.Path); err != nil {
		return "", err
	}
	return "updated", nil
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

func staleLinks(state State, desired map[string]desiredLink) []ManagedLink {
	stale := make([]ManagedLink, 0)
	for _, link := range state.Links {
		if _, ok := desired[link.Path]; ok {
			continue
		}
		stale = append(stale, link)
	}
	return stale
}

func currentLinkStatus(path string, target string) string {
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
		return "ok"
	}
	return "conflict"
}

func ensureManifestDefaults(manifest *Manifest) {
	if manifest.Sources == nil {
		manifest.Sources = map[string]ProjectSource{}
	}
	if manifest.Skills == nil {
		manifest.Skills = []ProjectSkill{}
	}
}

func sanitizeIDComponent(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func shortCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

func sortSourceReports(reports []SourceReport) {
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Alias < reports[j].Alias
	})
}

func sortLinkReports(reports []LinkReport) {
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Agent != reports[j].Agent {
			return reports[i].Agent < reports[j].Agent
		}
		if reports[i].Source != reports[j].Source {
			return reports[i].Source < reports[j].Source
		}
		return reports[i].Skill < reports[j].Skill
	})
}

func sortedResolvedSources(sources map[string]*resolvedSource) []*resolvedSource {
	aliases := make([]string, 0, len(sources))
	for alias := range sources {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	out := make([]*resolvedSource, 0, len(aliases))
	for _, alias := range aliases {
		out = append(out, sources[alias])
	}
	return out
}
