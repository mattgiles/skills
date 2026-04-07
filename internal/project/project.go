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
	ManifestFilename     = ".agents/manifest.yaml"
	StateFilename        = ".agents/state.yaml"
	SkillsDirname        = ".agents/skills"
	ClaudeSkillsDirname  = ".claude/skills"
	homeManifestFilename = "manifest.yaml"
	homeStateFilename    = "state.yaml"
	projectWorkspaceName = "project"
	sharedWorkspaceName  = "home"
)

type Manifest struct {
	Sources map[string]ManifestSource `yaml:"sources"`
	Skills  []ManifestSkill           `yaml:"skills"`
}

type ManifestSource struct {
	URL string `yaml:"url,omitempty"`
	Ref string `yaml:"ref"`
}

type ManifestSkill struct {
	Source string `yaml:"source"`
	Name   string `yaml:"name"`
}

type State struct {
	Sources     []SourceState `yaml:"sources,omitempty"`
	SkillLinks  []ManagedLink `yaml:"skill_links,omitempty"`
	ClaudeLinks []ManagedLink `yaml:"claude_links,omitempty"`
}

type SourceState struct {
	Source         string `yaml:"source"`
	Ref            string `yaml:"ref"`
	ResolvedCommit string `yaml:"resolved_commit"`
}

type ManagedLink struct {
	Path   string `yaml:"path"`
	Target string `yaml:"target"`
	Source string `yaml:"source"`
	Skill  string `yaml:"skill"`
}

type StatusReport struct {
	Sources          []SourceReport
	SkillLinks       []LinkReport
	ClaudeLinks      []LinkReport
	StaleSkillLinks  []ManagedLink
	StaleClaudeLinks []ManagedLink
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
	Path    string
	Target  string
	Status  string
	Message string
}

type SyncResult struct {
	Sources           []SourceReport
	SkillLinks        []LinkReport
	ClaudeLinks       []LinkReport
	PrunedSkillLinks  []string
	PrunedClaudeLinks []string
	DryRun            bool
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

type workspace struct {
	Name            string
	RootDir         string
	ManifestPath    string
	StatePath       string
	SkillsDir       string
	ClaudeSkillsDir string
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
	WorkspaceID  string

	StoredCommit  string
	CurrentCommit string
	DesiredCommit string
	WorktreePath  string
	InspectError  string
	SkillsByName  map[string][]discovery.DiscoveredSkill
}

func DefaultManifest() Manifest {
	return Manifest{
		Sources: map[string]ManifestSource{},
		Skills:  []ManifestSkill{},
	}
}

func ManifestPath(projectDir string) string {
	return filepath.Join(projectDir, ManifestFilename)
}

func StatePath(projectDir string) string {
	return filepath.Join(projectDir, StateFilename)
}

func SkillsDir(projectDir string) string {
	return filepath.Join(projectDir, SkillsDirname)
}

func ClaudeSkillsDir(projectDir string) string {
	return filepath.Join(projectDir, ClaudeSkillsDirname)
}

func HomeManifestPath(cfg config.Config) (string, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return "", err
	}
	return ws.ManifestPath, nil
}

func HomeStatePath(cfg config.Config) (string, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return "", err
	}
	return ws.StatePath, nil
}

func InitProject(projectDir string) error {
	ws := projectWorkspace(projectDir)
	if err := SaveManifestAt(ws.ManifestPath, DefaultManifest()); err != nil {
		return err
	}
	if err := os.MkdirAll(ws.SkillsDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(ws.ClaudeSkillsDir, 0o755); err != nil {
		return err
	}
	return nil
}

func InitHome(cfg config.Config) (string, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return "", err
	}
	if err := SaveManifestAt(ws.ManifestPath, DefaultManifest()); err != nil {
		return "", err
	}
	if err := os.MkdirAll(ws.SkillsDir, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(ws.ClaudeSkillsDir, 0o755); err != nil {
		return "", err
	}
	return ws.ManifestPath, nil
}

func LoadManifest(projectDir string) (Manifest, error) {
	return LoadManifestAt(ManifestPath(projectDir))
}

func LoadManifestAt(path string) (Manifest, error) {
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
	return SaveManifestAt(ManifestPath(projectDir), manifest)
}

func SaveManifestAt(path string, manifest Manifest) error {
	ensureManifestDefaults(&manifest)
	if err := ValidateManifest(manifest); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&manifest)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func LoadState(projectDir string) (State, error) {
	return LoadStateAt(StatePath(projectDir))
}

func LoadStateAt(path string) (State, error) {
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
	return SaveStateAt(StatePath(projectDir), state)
}

func SaveStateAt(path string, state State) error {
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

		key := skill.Source + "\x00" + skill.Name
		if _, ok := seenSkills[key]; ok {
			return fmt.Errorf("duplicate skill declaration for %s/%s", skill.Source, skill.Name)
		}
		seenSkills[key] = struct{}{}
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
	return statusWorkspace(ctx, projectWorkspace(projectDir), cfg)
}

func HomeStatus(ctx context.Context, cfg config.Config) (StatusReport, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return StatusReport{}, err
	}
	return statusWorkspace(ctx, ws, cfg)
}

func Sync(ctx context.Context, projectDir string, cfg config.Config, options SyncOptions) (SyncResult, error) {
	return syncWorkspace(ctx, projectWorkspace(projectDir), cfg, options)
}

func HomeSync(ctx context.Context, cfg config.Config, options SyncOptions) (SyncResult, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return SyncResult{}, err
	}
	return syncWorkspace(ctx, ws, cfg, options)
}

func Update(ctx context.Context, projectDir string, cfg config.Config, options UpdateOptions) (UpdateResult, error) {
	return updateWorkspace(ctx, projectWorkspace(projectDir), cfg, options)
}

func HomeUpdate(ctx context.Context, cfg config.Config, options UpdateOptions) (UpdateResult, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return UpdateResult{}, err
	}
	return updateWorkspace(ctx, ws, cfg, options)
}

func projectWorkspace(projectDir string) workspace {
	return workspace{
		Name:            projectWorkspaceName,
		RootDir:         projectDir,
		ManifestPath:    ManifestPath(projectDir),
		StatePath:       StatePath(projectDir),
		SkillsDir:       SkillsDir(projectDir),
		ClaudeSkillsDir: ClaudeSkillsDir(projectDir),
	}
}

func homeWorkspace(cfg config.Config) (workspace, error) {
	skillsDir, err := config.SharedSkillsDirPath(cfg)
	if err != nil {
		return workspace{}, err
	}
	claudeDir, err := config.SharedClaudeSkillsDirPath(cfg)
	if err != nil {
		return workspace{}, err
	}

	rootDir := filepath.Dir(skillsDir)
	return workspace{
		Name:            sharedWorkspaceName,
		RootDir:         rootDir,
		ManifestPath:    filepath.Join(rootDir, homeManifestFilename),
		StatePath:       filepath.Join(rootDir, homeStateFilename),
		SkillsDir:       skillsDir,
		ClaudeSkillsDir: claudeDir,
	}, nil
}

func statusWorkspace(ctx context.Context, ws workspace, cfg config.Config) (StatusReport, error) {
	manifest, state, resolvedSources, err := loadWorkspaceInputs(ws, cfg)
	if err != nil {
		return StatusReport{}, err
	}

	stateSources := sourceStateMap(state)
	skillStateLinks := managedLinkMap(state.SkillLinks)
	claudeStateLinks := managedLinkMap(state.ClaudeLinks)

	sourceReports, err := resolveSourcesForStatus(ctx, resolvedSources, stateSources)
	if err != nil {
		return StatusReport{}, err
	}

	desiredSkillLinks, skillReports := buildSkillLinkReports(resolvedSources, manifest, ws.SkillsDir, skillStateLinks)
	desiredSkillsByPath := desiredLinkMap(desiredSkillLinks)
	staleSkillLinks := staleLinks(state.SkillLinks, desiredSkillsByPath)

	desiredClaudeLinks, claudeReports := buildClaudeLinkReports(desiredSkillLinks, ws.ClaudeSkillsDir, claudeStateLinks)
	desiredClaudeByPath := desiredLinkMap(desiredClaudeLinks)
	staleClaudeLinks := staleLinks(state.ClaudeLinks, desiredClaudeByPath)

	sortSourceReports(sourceReports)
	sortLinkReports(skillReports)
	sortLinkReports(claudeReports)
	sortManagedLinks(staleSkillLinks)
	sortManagedLinks(staleClaudeLinks)

	return StatusReport{
		Sources:          sourceReports,
		SkillLinks:       skillReports,
		ClaudeLinks:      claudeReports,
		StaleSkillLinks:  staleSkillLinks,
		StaleClaudeLinks: staleClaudeLinks,
	}, nil
}

func syncWorkspace(ctx context.Context, ws workspace, cfg config.Config, options SyncOptions) (SyncResult, error) {
	manifest, state, resolvedSources, err := loadWorkspaceInputs(ws, cfg)
	if err != nil {
		return SyncResult{}, err
	}

	stateSources := sourceStateMap(state)
	skillStateLinks := managedLinkMap(state.SkillLinks)
	claudeStateLinks := managedLinkMap(state.ClaudeLinks)

	sourceReports, sourceStates, err := resolveSourcesForSync(ctx, resolvedSources, stateSources)
	if err != nil {
		return SyncResult{}, err
	}

	desiredSkillLinks, skillReports := buildSkillLinkReports(resolvedSources, manifest, ws.SkillsDir, skillStateLinks)
	if err := validateDesiredLinks(desiredSkillLinks); err != nil {
		return SyncResult{}, err
	}
	if err := fatalSkillLinkReports(skillReports); err != nil {
		return SyncResult{}, err
	}

	skillActions, err := planLinkActions(desiredSkillLinks, skillStateLinks)
	if err != nil {
		return SyncResult{}, err
	}
	updateLinkReportsForActions(skillReports, skillActions, options.DryRun)

	desiredClaudeLinks, claudeReports := buildClaudeLinkReports(desiredSkillLinks, ws.ClaudeSkillsDir, claudeStateLinks)
	if err := validateDesiredLinks(desiredClaudeLinks); err != nil {
		return SyncResult{}, err
	}
	if err := fatalAdapterLinkReports(claudeReports); err != nil {
		return SyncResult{}, err
	}

	claudeActions, err := planLinkActions(desiredClaudeLinks, claudeStateLinks)
	if err != nil {
		return SyncResult{}, err
	}
	updateLinkReportsForActions(claudeReports, claudeActions, options.DryRun)

	staleSkillLinks := staleLinks(state.SkillLinks, desiredLinkMap(desiredSkillLinks))
	prunedSkillLinks := managedLinkPaths(staleSkillLinks)

	staleClaudeLinks := staleLinks(state.ClaudeLinks, desiredLinkMap(desiredClaudeLinks))
	prunedClaudeLinks := managedLinkPaths(staleClaudeLinks)

	if options.DryRun {
		result := SyncResult{
			Sources:           sourceReports,
			SkillLinks:        skillReports,
			ClaudeLinks:       claudeReports,
			PrunedSkillLinks:  prunedSkillLinks,
			PrunedClaudeLinks: prunedClaudeLinks,
			DryRun:            true,
		}
		sortSourceReports(result.Sources)
		sortLinkReports(result.SkillLinks)
		sortLinkReports(result.ClaudeLinks)
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

	if err := applyLinkActions(skillActions); err != nil {
		return SyncResult{}, err
	}
	if err := applyLinkActions(claudeActions); err != nil {
		return SyncResult{}, err
	}
	if err := removeManagedLinks(staleClaudeLinks); err != nil {
		return SyncResult{}, err
	}
	if err := removeManagedLinks(staleSkillLinks); err != nil {
		return SyncResult{}, err
	}

	nextState := State{
		Sources:     sourceStates,
		SkillLinks:  toManagedLinks(desiredSkillLinks),
		ClaudeLinks: toManagedLinks(desiredClaudeLinks),
	}
	if err := SaveStateAt(ws.StatePath, nextState); err != nil {
		return SyncResult{}, err
	}

	for i := range sourceReports {
		sourceReports[i].Status = syncSourceStatus(sourceReports[i].Status)
	}

	result := SyncResult{
		Sources:           sourceReports,
		SkillLinks:        skillReports,
		ClaudeLinks:       claudeReports,
		PrunedSkillLinks:  prunedSkillLinks,
		PrunedClaudeLinks: prunedClaudeLinks,
	}
	sortSourceReports(result.Sources)
	sortLinkReports(result.SkillLinks)
	sortLinkReports(result.ClaudeLinks)
	return result, nil
}

func updateWorkspace(ctx context.Context, ws workspace, cfg config.Config, options UpdateOptions) (UpdateResult, error) {
	manifest, state, resolvedSources, err := loadWorkspaceInputs(ws, cfg)
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
		syncResult, err := syncWorkspaceWithState(ctx, ws, manifest, nextState, resolvedSources, options.DryRun)
		if err != nil {
			return UpdateResult{}, err
		}
		result.Sync = &syncResult
		return result, nil
	}

	if !options.DryRun {
		nextState := mergeSourceStates(state, nextSourceStates, nil)
		if err := SaveStateAt(ws.StatePath, nextState); err != nil {
			return UpdateResult{}, err
		}
	}

	sortSourceReports(result.Sources)
	return result, nil
}

func syncWorkspaceWithState(ctx context.Context, ws workspace, manifest Manifest, state State, resolvedSources map[string]*resolvedSource, dryRun bool) (SyncResult, error) {
	skillStateLinks := managedLinkMap(state.SkillLinks)
	claudeStateLinks := managedLinkMap(state.ClaudeLinks)
	stateSources := sourceStateMap(state)

	sourceReports, sourceStates, err := resolveSourcesForSync(ctx, resolvedSources, stateSources)
	if err != nil {
		return SyncResult{}, err
	}

	desiredSkillLinks, skillReports := buildSkillLinkReports(resolvedSources, manifest, ws.SkillsDir, skillStateLinks)
	if err := validateDesiredLinks(desiredSkillLinks); err != nil {
		return SyncResult{}, err
	}
	if err := fatalSkillLinkReports(skillReports); err != nil {
		return SyncResult{}, err
	}

	skillActions, err := planLinkActions(desiredSkillLinks, skillStateLinks)
	if err != nil {
		return SyncResult{}, err
	}
	updateLinkReportsForActions(skillReports, skillActions, dryRun)

	desiredClaudeLinks, claudeReports := buildClaudeLinkReports(desiredSkillLinks, ws.ClaudeSkillsDir, claudeStateLinks)
	if err := validateDesiredLinks(desiredClaudeLinks); err != nil {
		return SyncResult{}, err
	}
	if err := fatalAdapterLinkReports(claudeReports); err != nil {
		return SyncResult{}, err
	}

	claudeActions, err := planLinkActions(desiredClaudeLinks, claudeStateLinks)
	if err != nil {
		return SyncResult{}, err
	}
	updateLinkReportsForActions(claudeReports, claudeActions, dryRun)

	staleSkillLinks := staleLinks(state.SkillLinks, desiredLinkMap(desiredSkillLinks))
	prunedSkillLinks := managedLinkPaths(staleSkillLinks)

	staleClaudeLinks := staleLinks(state.ClaudeLinks, desiredLinkMap(desiredClaudeLinks))
	prunedClaudeLinks := managedLinkPaths(staleClaudeLinks)

	if dryRun {
		result := SyncResult{
			Sources:           sourceReports,
			SkillLinks:        skillReports,
			ClaudeLinks:       claudeReports,
			PrunedSkillLinks:  prunedSkillLinks,
			PrunedClaudeLinks: prunedClaudeLinks,
			DryRun:            true,
		}
		sortSourceReports(result.Sources)
		sortLinkReports(result.SkillLinks)
		sortLinkReports(result.ClaudeLinks)
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

	if err := applyLinkActions(skillActions); err != nil {
		return SyncResult{}, err
	}
	if err := applyLinkActions(claudeActions); err != nil {
		return SyncResult{}, err
	}
	if err := removeManagedLinks(staleClaudeLinks); err != nil {
		return SyncResult{}, err
	}
	if err := removeManagedLinks(staleSkillLinks); err != nil {
		return SyncResult{}, err
	}

	nextState := State{
		Sources:     sourceStates,
		SkillLinks:  toManagedLinks(desiredSkillLinks),
		ClaudeLinks: toManagedLinks(desiredClaudeLinks),
	}
	if err := SaveStateAt(ws.StatePath, nextState); err != nil {
		return SyncResult{}, err
	}

	for i := range sourceReports {
		sourceReports[i].Status = syncSourceStatus(sourceReports[i].Status)
	}

	result := SyncResult{
		Sources:           sourceReports,
		SkillLinks:        skillReports,
		ClaudeLinks:       claudeReports,
		PrunedSkillLinks:  prunedSkillLinks,
		PrunedClaudeLinks: prunedClaudeLinks,
	}
	sortSourceReports(result.Sources)
	sortLinkReports(result.SkillLinks)
	sortLinkReports(result.ClaudeLinks)
	return result, nil
}

func loadWorkspaceInputs(ws workspace, cfg config.Config) (Manifest, State, map[string]*resolvedSource, error) {
	manifest, err := LoadManifestAt(ws.ManifestPath)
	if err != nil {
		return Manifest{}, State{}, nil, err
	}

	state, err := LoadStateAt(ws.StatePath)
	if err != nil {
		return Manifest{}, State{}, nil, err
	}

	worktreeRoot, err := config.WorktreeRootPath(cfg)
	if err != nil {
		return Manifest{}, State{}, nil, err
	}

	resolvedSources, err := resolveInputs(ws, cfg, manifest, worktreeRoot)
	if err != nil {
		return Manifest{}, State{}, nil, err
	}

	return manifest, state, resolvedSources, nil
}

func resolveInputs(ws workspace, cfg config.Config, manifest Manifest, worktreeRoot string) (map[string]*resolvedSource, error) {
	repoRoot, err := config.RepoRootPath(cfg)
	if err != nil {
		return nil, err
	}

	workspaceID, err := ProjectID(ws.RootDir)
	if err != nil {
		return nil, err
	}

	resolvedSources := map[string]*resolvedSource{}
	for alias, manifestSource := range manifest.Sources {
		globalSource, hasGlobal := cfg.Sources[alias]
		switch {
		case hasGlobal && strings.TrimSpace(manifestSource.URL) != "" && manifestSource.URL != globalSource.URL:
			return nil, fmt.Errorf("source %q has conflicting URLs between global config and manifest", alias)
		case strings.TrimSpace(manifestSource.URL) == "" && !hasGlobal:
			return nil, fmt.Errorf("source %q has no URL in manifest or global config", alias)
		}

		url := manifestSource.URL
		if strings.TrimSpace(url) == "" {
			url = globalSource.URL
		}

		resolvedSources[alias] = &resolvedSource{
			Alias:        alias,
			URL:          url,
			Ref:          manifestSource.Ref,
			RepoPath:     source.RepoPath(repoRoot, alias),
			WorktreeRoot: worktreeRoot,
			WorkspaceID:  workspaceID,
		}
	}

	return resolvedSources, nil
}

func resolveSourcesForStatus(ctx context.Context, resolvedSources map[string]*resolvedSource, stateSources map[string]SourceState) ([]SourceReport, error) {
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
					report.Message = "run sync or update"
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
			src.WorktreePath = source.WorktreePath(src.WorktreeRoot, src.WorkspaceID, src.Alias, src.DesiredCommit)
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

func resolveSourcesForSync(ctx context.Context, resolvedSources map[string]*resolvedSource, stateSources map[string]SourceState) ([]SourceReport, []SourceState, error) {
	reports := make([]SourceReport, 0, len(resolvedSources))
	nextStates := make([]SourceState, 0, len(resolvedSources))

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
			src.WorktreePath = source.WorktreePath(src.WorktreeRoot, src.WorkspaceID, src.Alias, src.DesiredCommit)
			report.WorktreePath = src.WorktreePath
			skillsByName, inspectErr := loadSkillsForCommit(ctx, src)
			if inspectErr != nil {
				src.InspectError = inspectErr.Error()
				report.Status = "inspect-failed"
				report.Message = inspectErr.Error()
			} else {
				src.SkillsByName = skillsByName
			}
			nextStates = append(nextStates, SourceState{
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

func resolveSourcesForUpdate(ctx context.Context, resolvedSources map[string]*resolvedSource, stateSources map[string]SourceState) ([]SourceReport, []SourceState, error) {
	reports := make([]SourceReport, 0, len(resolvedSources))
	nextStates := make([]SourceState, 0, len(resolvedSources))

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
				report.WorktreePath = source.WorktreePath(src.WorktreeRoot, src.WorkspaceID, src.Alias, commit)
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

				nextStates = append(nextStates, SourceState{
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

func setDesiredCommitForStatus(src *resolvedSource, hasPrev bool, prev SourceState) {
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
					},
				})
			}
		}

		reports = append(reports, report)
	}

	sortLinkReports(reports)
	return desired, reports
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

func selectResolvedSources(all map[string]*resolvedSource, aliases []string) (map[string]*resolvedSource, error) {
	if len(aliases) == 0 {
		return all, nil
	}

	selected := map[string]*resolvedSource{}
	for _, alias := range aliases {
		src, ok := all[alias]
		if !ok {
			return nil, fmt.Errorf("unknown source %q", alias)
		}
		selected[alias] = src
	}
	return selected, nil
}

func sourceStateMap(state State) map[string]SourceState {
	out := map[string]SourceState{}
	for _, src := range state.Sources {
		out[src.Source] = src
	}
	return out
}

func managedLinkMap(links []ManagedLink) map[string]ManagedLink {
	out := map[string]ManagedLink{}
	for _, link := range links {
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

func mergeSourceStates(state State, updates []SourceState, removeAliases map[string]struct{}) State {
	current := sourceStateMap(state)
	for _, update := range updates {
		current[update.Source] = update
	}
	for alias := range removeAliases {
		delete(current, alias)
	}

	next := State{
		Sources:     make([]SourceState, 0, len(current)),
		SkillLinks:  state.SkillLinks,
		ClaudeLinks: state.ClaudeLinks,
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
		manifest.Sources = map[string]ManifestSource{}
	}
	if manifest.Skills == nil {
		manifest.Skills = []ManifestSkill{}
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

func managedLinkPaths(links []ManagedLink) []string {
	paths := make([]string, 0, len(links))
	for _, link := range links {
		paths = append(paths, link.Path)
	}
	sort.Strings(paths)
	return paths
}

func toManagedLinks(links []desiredLink) []ManagedLink {
	out := make([]ManagedLink, 0, len(links))
	for _, link := range links {
		out = append(out, link.ManagedLink)
	}
	return out
}

func sortSourceReports(reports []SourceReport) {
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Alias < reports[j].Alias
	})
}

func sortLinkReports(reports []LinkReport) {
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Source != reports[j].Source {
			return reports[i].Source < reports[j].Source
		}
		return reports[i].Skill < reports[j].Skill
	})
}

func sortManagedLinks(links []ManagedLink) {
	sort.Slice(links, func(i, j int) bool {
		return links[i].Path < links[j].Path
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
