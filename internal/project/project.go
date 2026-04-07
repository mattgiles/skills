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
	Sources []ProjectSourceState `yaml:"sources,omitempty"`
	Links   []ManagedLink        `yaml:"links,omitempty"`
}

type ProjectSourceState struct {
	Source         string `yaml:"source"`
	Ref            string `yaml:"ref"`
	ResolvedCommit string `yaml:"resolved_commit"`
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
	Alias          string
	Ref            string
	Commit         string
	PreviousCommit string
	RepoPath       string
	WorktreePath   string
	Status         string
	Message        string
}

type LinkReport struct {
	Source  string
	Skill   string
	Agent   string
	Path    string
	Target  string
	Status  string
	Message string
}

type SyncResult struct {
	Sources []SourceReport
	Links   []LinkReport
	Pruned  []string
	DryRun  bool
}

type UpdateResult struct {
	Sources []SourceReport
	Sync    *SyncResult
	DryRun  bool
}

type SyncOptions struct {
	DryRun bool
}

type UpdateOptions struct {
	SelectedSources []string
	Sync            bool
	DryRun          bool
}

type desiredLink struct {
	ManagedLink
}

type linkAction struct {
	Link   desiredLink
	Status string
}

type resolvedSource struct {
	Alias        string
	URL          string
	Ref          string
	RepoPath     string
	WorktreeRoot string
	ProjectID    string

	StoredCommit  string
	CurrentCommit string
	DesiredCommit string
	WorktreePath  string
	InspectError  string
	SkillsByName  map[string][]discovery.DiscoveredSkill
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
	manifest, state, resolvedSources, agents, err := loadProjectInputs(projectDir, cfg)
	if err != nil {
		return StatusReport{}, err
	}

	stateSources := sourceStateMap(state)
	stateLinks := managedLinkMap(state)

	sourceReports, err := resolveSourcesForStatus(ctx, resolvedSources, stateSources)
	if err != nil {
		return StatusReport{}, err
	}

	desiredLinks, linkReports := buildLinkReports(resolvedSources, agents, manifest, stateLinks, false)
	desiredByPath := desiredLinkMap(desiredLinks)
	stale := staleLinks(state, desiredByPath)

	sortSourceReports(sourceReports)
	sortLinkReports(linkReports)
	sort.Slice(stale, func(i, j int) bool {
		return stale[i].Path < stale[j].Path
	})

	return StatusReport{
		Sources:    sourceReports,
		Links:      linkReports,
		StaleLinks: stale,
	}, nil
}

func Sync(ctx context.Context, projectDir string, cfg config.Config, options SyncOptions) (SyncResult, error) {
	manifest, state, resolvedSources, agents, err := loadProjectInputs(projectDir, cfg)
	if err != nil {
		return SyncResult{}, err
	}

	return syncWithState(ctx, projectDir, manifest, state, resolvedSources, agents, options)
}

func Update(ctx context.Context, projectDir string, cfg config.Config, options UpdateOptions) (UpdateResult, error) {
	manifest, state, resolvedSources, agents, err := loadProjectInputs(projectDir, cfg)
	if err != nil {
		return UpdateResult{}, err
	}

	selected, err := selectResolvedSources(resolvedSources, options.SelectedSources)
	if err != nil {
		return UpdateResult{}, err
	}

	stateSources := sourceStateMap(state)
	reports, nextSourceStates, err := resolveSourcesForUpdate(ctx, selected, stateSources)
	if err != nil {
		return UpdateResult{}, err
	}

	result := UpdateResult{
		Sources: reports,
		DryRun:  options.DryRun,
	}

	if options.Sync {
		nextState := mergeSourceStates(state, nextSourceStates, nil)
		syncResult, err := syncWithState(ctx, projectDir, manifest, nextState, resolvedSources, agents, SyncOptions{DryRun: options.DryRun})
		if err != nil {
			return UpdateResult{}, err
		}
		result.Sync = &syncResult
		return result, nil
	}

	if !options.DryRun {
		nextState := mergeSourceStates(state, nextSourceStates, nil)
		if err := SaveState(projectDir, nextState); err != nil {
			return UpdateResult{}, err
		}
	}

	sortSourceReports(result.Sources)
	return result, nil
}

func loadProjectInputs(projectDir string, cfg config.Config) (Manifest, State, map[string]*resolvedSource, map[string]resolvedAgent, error) {
	manifest, err := LoadManifest(projectDir)
	if err != nil {
		return Manifest{}, State{}, nil, nil, err
	}

	state, err := LoadState(projectDir)
	if err != nil {
		return Manifest{}, State{}, nil, nil, err
	}

	worktreeRoot, err := config.WorktreeRootPath(cfg)
	if err != nil {
		return Manifest{}, State{}, nil, nil, err
	}

	resolvedSources, agents, err := resolveInputs(projectDir, cfg, manifest, worktreeRoot)
	if err != nil {
		return Manifest{}, State{}, nil, nil, err
	}

	return manifest, state, resolvedSources, agents, nil
}

func syncWithState(ctx context.Context, projectDir string, manifest Manifest, state State, resolvedSources map[string]*resolvedSource, agents map[string]resolvedAgent, options SyncOptions) (SyncResult, error) {
	stateSources := sourceStateMap(state)
	stateLinks := managedLinkMap(state)

	sourceReports, sourceStates, err := resolveSourcesForSync(ctx, resolvedSources, stateSources)
	if err != nil {
		return SyncResult{}, err
	}

	desiredLinks, linkReports := buildLinkReports(resolvedSources, agents, manifest, stateLinks, true)
	if err := validateDesiredLinks(desiredLinks); err != nil {
		return SyncResult{}, err
	}
	if err := fatalLinkReports(linkReports); err != nil {
		return SyncResult{}, err
	}

	actions, err := planLinkActions(desiredLinks, stateLinks)
	if err != nil {
		return SyncResult{}, err
	}

	actionByPath := map[string]string{}
	for _, action := range actions {
		actionByPath[action.Link.Path] = action.Status
	}
	for i := range linkReports {
		if planned, ok := actionByPath[linkReports[i].Path]; ok {
			if options.DryRun {
				linkReports[i].Status = dryRunLinkStatus(planned)
			} else {
				linkReports[i].Status = planned
			}
		}
	}

	desiredByPath := desiredLinkMap(desiredLinks)
	stale := staleLinks(state, desiredByPath)
	prunedPaths := make([]string, 0, len(stale))
	for _, link := range stale {
		prunedPaths = append(prunedPaths, link.Path)
	}
	sort.Strings(prunedPaths)

	if options.DryRun {
		result := SyncResult{
			Sources: sourceReports,
			Links:   linkReports,
			Pruned:  prunedPaths,
			DryRun:  true,
		}
		sortSourceReports(result.Sources)
		sortLinkReports(result.Links)
		return result, nil
	}

	for _, src := range sortedResolvedSources(resolvedSources) {
		if strings.TrimSpace(src.DesiredCommit) == "" {
			continue
		}
		if _, err := source.EnsureWorktree(ctx, source.Source{
			Alias:    src.Alias,
			URL:      src.URL,
			RepoPath: src.RepoPath,
		}, src.WorktreePath, src.DesiredCommit); err != nil {
			return SyncResult{}, err
		}
	}

	for _, action := range actions {
		switch action.Status {
		case "linked":
			continue
		case "created":
			if err := createSymlink(action.Link); err != nil {
				return SyncResult{}, err
			}
		case "updated":
			if err := replaceSymlink(action.Link); err != nil {
				return SyncResult{}, err
			}
		default:
			return SyncResult{}, fmt.Errorf("unexpected link action %q", action.Status)
		}
	}

	for _, link := range stale {
		if err := removeManagedLink(link); err != nil {
			return SyncResult{}, err
		}
	}

	nextState := State{
		Sources: sourceStates,
		Links:   make([]ManagedLink, 0, len(desiredLinks)),
	}
	for _, link := range desiredLinks {
		nextState.Links = append(nextState.Links, link.ManagedLink)
	}

	if err := SaveState(projectDir, nextState); err != nil {
		return SyncResult{}, err
	}

	for i := range sourceReports {
		sourceReports[i].Status = syncSourceStatus(sourceReports[i].Status)
	}

	result := SyncResult{
		Sources: sourceReports,
		Links:   linkReports,
		Pruned:  prunedPaths,
	}
	sortSourceReports(result.Sources)
	sortLinkReports(result.Links)
	return result, nil
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

func resolveSourcesForStatus(ctx context.Context, resolvedSources map[string]*resolvedSource, stateSources map[string]ProjectSourceState) ([]SourceReport, error) {
	reports := make([]SourceReport, 0, len(resolvedSources))

	for _, src := range sortedResolvedSources(resolvedSources) {
		prev, hasPrev := stateSources[src.Alias]
		src.StoredCommit = prev.ResolvedCommit

		status := inspectSource(ctx, src)
		report := SourceReport{
			Alias:          src.Alias,
			Ref:            src.Ref,
			PreviousCommit: shortCommit(prev.ResolvedCommit),
			RepoPath:       src.RepoPath,
		}

		switch {
		case !status.Exists:
			report.Status = "missing-source"
			report.Message = "canonical source repo is not cloned"
		case !status.IsGitRepo:
			report.Status = "invalid-source"
			report.Message = status.LastError
		default:
			commit, err := source.ResolveCommit(ctx, source.Source{
				Alias:    src.Alias,
				URL:      src.URL,
				RepoPath: src.RepoPath,
			}, src.Ref)
			if err != nil {
				report.Status = "invalid-ref"
				report.Message = err.Error()
			} else {
				src.CurrentCommit = commit
				report.Commit = shortCommit(commit)
				switch {
				case !hasPrev || strings.TrimSpace(prev.ResolvedCommit) == "":
					report.Status = "not-synced"
					report.Message = "run project sync or project update"
				case prev.Ref != src.Ref:
					report.Status = "update-available"
					report.Message = "state recorded for ref " + prev.Ref
				case prev.ResolvedCommit != commit:
					report.Status = "update-available"
					report.Message = "last resolved " + shortCommit(prev.ResolvedCommit)
				default:
					report.Status = "up-to-date"
				}
			}
		}

		setDesiredCommitForStatus(src, hasPrev, prev)
		if status.Exists && status.IsGitRepo && strings.TrimSpace(src.DesiredCommit) != "" {
			src.WorktreePath = source.WorktreePath(src.WorktreeRoot, src.ProjectID, src.Alias, src.DesiredCommit)
			report.WorktreePath = src.WorktreePath
			skillsByName, inspectErr := loadSkillsForCommit(ctx, src)
			if inspectErr != nil {
				src.InspectError = inspectErr.Error()
				report.Status = "inspect-failed"
				report.Message = inspectErr.Error()
			} else {
				src.SkillsByName = skillsByName
			}
		}

		reports = append(reports, report)
	}

	return reports, nil
}

func resolveSourcesForSync(ctx context.Context, resolvedSources map[string]*resolvedSource, stateSources map[string]ProjectSourceState) ([]SourceReport, []ProjectSourceState, error) {
	reports := make([]SourceReport, 0, len(resolvedSources))
	nextStates := make([]ProjectSourceState, 0, len(resolvedSources))

	for _, src := range sortedResolvedSources(resolvedSources) {
		prev, hasPrev := stateSources[src.Alias]
		src.StoredCommit = prev.ResolvedCommit

		status, err := ensureSourceReady(ctx, src, true, true)
		if err != nil {
			return nil, nil, err
		}

		report := SourceReport{
			Alias:          src.Alias,
			Ref:            src.Ref,
			PreviousCommit: shortCommit(prev.ResolvedCommit),
			RepoPath:       src.RepoPath,
		}

		switch {
		case !status.Exists:
			report.Status = "missing-source"
			report.Message = "canonical source repo is not cloned"
		case !status.IsGitRepo:
			report.Status = "invalid-source"
			report.Message = status.LastError
		default:
			useStored := hasPrev && prev.Ref == src.Ref && strings.TrimSpace(prev.ResolvedCommit) != ""
			if useStored {
				src.DesiredCommit = prev.ResolvedCommit
				report.Commit = shortCommit(prev.ResolvedCommit)
				report.Status = "up-to-date"
			} else {
				commit, err := source.ResolveCommit(ctx, source.Source{
					Alias:    src.Alias,
					URL:      src.URL,
					RepoPath: src.RepoPath,
				}, src.Ref)
				if err != nil {
					report.Status = "invalid-ref"
					report.Message = err.Error()
				} else {
					src.CurrentCommit = commit
					src.DesiredCommit = commit
					report.Commit = shortCommit(commit)
					report.Status = "not-synced"
					if hasPrev && prev.Ref == src.Ref && prev.ResolvedCommit != commit && prev.ResolvedCommit != "" {
						report.Status = "update-available"
						report.Message = "stored " + shortCommit(prev.ResolvedCommit)
					}
				}
			}
		}

		if status.Exists && status.IsGitRepo && strings.TrimSpace(src.DesiredCommit) != "" {
			src.WorktreePath = source.WorktreePath(src.WorktreeRoot, src.ProjectID, src.Alias, src.DesiredCommit)
			report.WorktreePath = src.WorktreePath
			skillsByName, inspectErr := loadSkillsForCommit(ctx, src)
			if inspectErr != nil {
				src.InspectError = inspectErr.Error()
				report.Status = "inspect-failed"
				report.Message = inspectErr.Error()
			} else {
				src.SkillsByName = skillsByName
			}
			nextStates = append(nextStates, ProjectSourceState{
				Source:         src.Alias,
				Ref:            src.Ref,
				ResolvedCommit: src.DesiredCommit,
			})
		}

		reports = append(reports, report)
	}

	if err := fatalSourceReports(reports); err != nil {
		return nil, nil, err
	}

	return reports, nextStates, nil
}

func resolveSourcesForUpdate(ctx context.Context, resolvedSources map[string]*resolvedSource, stateSources map[string]ProjectSourceState) ([]SourceReport, []ProjectSourceState, error) {
	reports := make([]SourceReport, 0, len(resolvedSources))
	nextStates := make([]ProjectSourceState, 0, len(resolvedSources))

	for _, src := range sortedResolvedSources(resolvedSources) {
		prev, hasPrev := stateSources[src.Alias]

		status, err := ensureSourceReady(ctx, src, true, true)
		if err != nil {
			return nil, nil, err
		}

		report := SourceReport{
			Alias:          src.Alias,
			Ref:            src.Ref,
			PreviousCommit: shortCommit(prev.ResolvedCommit),
			RepoPath:       src.RepoPath,
		}

		switch {
		case !status.Exists:
			report.Status = "missing-source"
			report.Message = "canonical source repo is not cloned"
		case !status.IsGitRepo:
			report.Status = "invalid-source"
			report.Message = status.LastError
		default:
			commit, err := source.ResolveCommit(ctx, source.Source{
				Alias:    src.Alias,
				URL:      src.URL,
				RepoPath: src.RepoPath,
			}, src.Ref)
			if err != nil {
				report.Status = "invalid-ref"
				report.Message = err.Error()
			} else {
				src.CurrentCommit = commit
				report.Commit = shortCommit(commit)
				report.WorktreePath = source.WorktreePath(src.WorktreeRoot, src.ProjectID, src.Alias, commit)
				switch {
				case !hasPrev || strings.TrimSpace(prev.ResolvedCommit) == "":
					report.Status = "resolved"
				case prev.Ref != src.Ref || prev.ResolvedCommit != commit:
					report.Status = "updated"
					if prev.ResolvedCommit != "" {
						report.Message = shortCommit(prev.ResolvedCommit) + " -> " + shortCommit(commit)
					}
				default:
					report.Status = "up-to-date"
				}

				nextStates = append(nextStates, ProjectSourceState{
					Source:         src.Alias,
					Ref:            src.Ref,
					ResolvedCommit: commit,
				})
			}
		}

		reports = append(reports, report)
	}

	if err := fatalSourceReports(reports); err != nil {
		return nil, nil, err
	}

	sortSourceReports(reports)
	return reports, nextStates, nil
}

func inspectSource(ctx context.Context, src *resolvedSource) source.SourceStatus {
	return source.Status(ctx, source.Source{
		Alias:    src.Alias,
		URL:      src.URL,
		RepoPath: src.RepoPath,
	})
}

func ensureSourceReady(ctx context.Context, src *resolvedSource, cloneMissing bool, fetchExisting bool) (source.SourceStatus, error) {
	srcDef := source.Source{
		Alias:    src.Alias,
		URL:      src.URL,
		RepoPath: src.RepoPath,
	}

	status := inspectSource(ctx, src)
	if !status.Exists {
		if !cloneMissing {
			return status, nil
		}
		if _, err := source.Sync(ctx, srcDef); err != nil {
			return source.SourceStatus{}, fmt.Errorf("sync source %s: %w", src.Alias, err)
		}
		return inspectSource(ctx, src), nil
	}

	if !status.IsGitRepo {
		return status, nil
	}

	if fetchExisting {
		if _, err := source.Sync(ctx, srcDef); err != nil {
			return source.SourceStatus{}, fmt.Errorf("sync source %s: %w", src.Alias, err)
		}
		return inspectSource(ctx, src), nil
	}

	return status, nil
}

func setDesiredCommitForStatus(src *resolvedSource, hasPrev bool, prev ProjectSourceState) {
	switch {
	case hasPrev && prev.Ref == src.Ref && strings.TrimSpace(prev.ResolvedCommit) != "":
		src.DesiredCommit = prev.ResolvedCommit
	case strings.TrimSpace(src.CurrentCommit) != "":
		src.DesiredCommit = src.CurrentCommit
	default:
		src.DesiredCommit = ""
	}
}

func loadSkillsForCommit(ctx context.Context, src *resolvedSource) (map[string][]discovery.DiscoveredSkill, error) {
	skillsByName := map[string][]discovery.DiscoveredSkill{}
	if strings.TrimSpace(src.DesiredCommit) == "" {
		return skillsByName, nil
	}

	paths, err := source.ListFilesAtCommit(ctx, source.Source{
		Alias:    src.Alias,
		URL:      src.URL,
		RepoPath: src.RepoPath,
	}, src.DesiredCommit)
	if err != nil {
		return nil, fmt.Errorf("inspect %s at %s: %w", src.Alias, shortCommit(src.DesiredCommit), err)
	}

	for _, skill := range discovery.DiscoverFromPaths(src.Alias, src.WorktreePath, paths) {
		skillsByName[skill.Name] = append(skillsByName[skill.Name], skill)
	}
	return skillsByName, nil
}

func buildLinkReports(resolvedSources map[string]*resolvedSource, agents map[string]resolvedAgent, manifest Manifest, stateLinks map[string]ManagedLink, forSync bool) ([]desiredLink, []LinkReport) {
	desired := make([]desiredLink, 0)
	reports := make([]LinkReport, 0)

	for _, skill := range manifest.Skills {
		src := resolvedSources[skill.Source]
		for _, agentName := range skill.Agents {
			agent := agents[agentName]
			report := LinkReport{
				Source: skill.Source,
				Skill:  skill.Name,
				Agent:  agentName,
				Path:   filepath.Join(agent.SkillsDir, skill.Name),
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
			default:
				matches := src.SkillsByName[skill.Name]
				switch {
				case len(matches) == 0:
					report.Status = "missing-skill"
				case len(matches) > 1:
					report.Status = "ambiguous-skill"
					report.Message = "multiple skills share this directory name"
				default:
					target := filepath.Join(src.WorktreePath, matches[0].RelativePath)
					report.Target = target
					report.Status = currentLinkStatus(report.Path, target, stateLinks)
					desired = append(desired, desiredLink{
						ManagedLink: ManagedLink{
							Path:   report.Path,
							Target: target,
							Source: skill.Source,
							Skill:  skill.Name,
							Agent:  agentName,
						},
					})
				}
			}

			reports = append(reports, report)
		}
	}

	if forSync {
		_ = fatalLinkReports(reports)
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

func fatalSourceReports(reports []SourceReport) error {
	problems := make([]string, 0)
	for _, report := range reports {
		switch report.Status {
		case "missing-source", "invalid-source", "invalid-ref", "inspect-failed":
			problem := fmt.Sprintf("%s: %s", report.Alias, report.Status)
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

func fatalLinkReports(reports []LinkReport) error {
	problems := make([]string, 0)
	for _, report := range reports {
		switch report.Status {
		case "unknown-source", "source-not-ready", "inspect-failed", "missing-skill", "ambiguous-skill", "conflict":
			problem := fmt.Sprintf("%s/%s/%s: %s", report.Agent, report.Source, report.Skill, report.Status)
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

func selectResolvedSources(all map[string]*resolvedSource, aliases []string) (map[string]*resolvedSource, error) {
	if len(aliases) == 0 {
		return all, nil
	}

	selected := map[string]*resolvedSource{}
	for _, alias := range aliases {
		src, ok := all[alias]
		if !ok {
			return nil, fmt.Errorf("unknown project source %q", alias)
		}
		selected[alias] = src
	}
	return selected, nil
}

func sourceStateMap(state State) map[string]ProjectSourceState {
	out := map[string]ProjectSourceState{}
	for _, src := range state.Sources {
		out[src.Source] = src
	}
	return out
}

func managedLinkMap(state State) map[string]ManagedLink {
	out := map[string]ManagedLink{}
	for _, link := range state.Links {
		out[link.Path] = link
	}
	return out
}

func desiredLinkMap(links []desiredLink) map[string]desiredLink {
	out := map[string]desiredLink{}
	for _, link := range links {
		out[link.Path] = link
	}
	return out
}

func mergeSourceStates(state State, updates []ProjectSourceState, removeAliases map[string]struct{}) State {
	current := sourceStateMap(state)
	for _, update := range updates {
		current[update.Source] = update
	}
	for alias := range removeAliases {
		delete(current, alias)
	}

	next := State{
		Sources: make([]ProjectSourceState, 0, len(current)),
		Links:   state.Links,
	}
	for _, src := range current {
		next.Sources = append(next.Sources, src)
	}
	sort.Slice(next.Sources, func(i, j int) bool {
		return next.Sources[i].Source < next.Sources[j].Source
	})
	return next
}

func ensureManifestDefaults(manifest *Manifest) {
	if manifest.Sources == nil {
		manifest.Sources = map[string]ProjectSource{}
	}
	if manifest.Agents == nil {
		manifest.Agents = map[string]ProjectAgentOverride{}
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

func syncSourceStatus(status string) string {
	switch status {
	case "not-synced":
		return "resolved"
	case "update-available":
		return "updated"
	default:
		return status
	}
}

func dryRunLinkStatus(status string) string {
	switch status {
	case "created":
		return "would-create"
	case "updated":
		return "would-update"
	default:
		return status
	}
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
