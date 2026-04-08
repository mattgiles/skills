package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/source"
)

const (
	SectionEnvironment = "ENVIRONMENT"
	SectionConfig      = "CONFIG"
	SectionWorkspace   = "WORKSPACE"
	SectionGit         = "GIT"
	SectionSources     = "SOURCES"
	SectionSkills      = "SKILLS"
	SectionClaude      = "CLAUDE"
	SectionHints       = "HINTS"
)

type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

type Severity string

const (
	SeverityError Severity = "ERROR"
	SeverityWarn  Severity = "WARN"
	SeverityInfo  Severity = "INFO"
)

type Finding struct {
	Section  string
	Severity Severity
	Code     string
	Subject  string
	Message  string
	Hint     string
	Path     string
	Target   string
	Ref      string
}

type Report struct {
	Scope    Scope
	Target   string
	Findings []Finding
}

type paths struct {
	repoRoot        string
	worktreeRoot    string
	sharedSkills    string
	sharedClaudeDir string
}

func Check(ctx context.Context, cwd string, scope Scope) (Report, error) {
	report := Report{
		Scope:  scope,
		Target: cwd,
	}

	gitAvailable := true
	if err := source.EnsureGitAvailable(); err != nil {
		gitAvailable = false
		report.addFinding(Finding{
			Section:  SectionEnvironment,
			Severity: SeverityError,
			Code:     "git-missing",
			Subject:  "git",
			Message:  err.Error(),
			Hint:     "install git and re-run skills doctor",
		})
	}

	switch scope {
	case ScopeProject:
		checkProjectWorkspace(ctx, &report, cwd, gitAvailable)
	case ScopeGlobal:
		configPath, err := config.DefaultConfigPath()
		if err != nil {
			report.addFinding(Finding{
				Section:  SectionConfig,
				Severity: SeverityError,
				Code:     "config-path-failed",
				Subject:  "config",
				Message:  err.Error(),
			})
			return report, nil
		}

		cfg, configUsable, configPaths := loadAndValidateConfig(&report, configPath)
		checkGlobalWorkspace(ctx, &report, cfg, configUsable, gitAvailable, configPaths)
	default:
		return Report{}, fmt.Errorf("unsupported doctor scope %q", scope)
	}

	return report, nil
}

func (r Report) ErrorCount() int {
	count := 0
	for _, finding := range r.Findings {
		if finding.Severity == SeverityError {
			count++
		}
	}
	return count
}

func (r Report) WarningCount() int {
	count := 0
	for _, finding := range r.Findings {
		if finding.Severity == SeverityWarn {
			count++
		}
	}
	return count
}

func (r Report) HasErrors() bool {
	return r.ErrorCount() > 0
}

func (r Report) Hints() []string {
	seen := map[string]struct{}{}
	hints := make([]string, 0)
	for _, finding := range r.Findings {
		if finding.Hint == "" || finding.Severity == SeverityInfo {
			continue
		}
		if _, ok := seen[finding.Hint]; ok {
			continue
		}
		seen[finding.Hint] = struct{}{}
		hints = append(hints, finding.Hint)
	}
	return hints
}

func (r *Report) addFinding(finding Finding) {
	r.Findings = append(r.Findings, finding)
}

func loadAndValidateConfig(report *Report, configPath string) (config.Config, bool, paths) {
	cfg := config.DefaultConfig()
	configPaths := paths{}

	info, err := os.Stat(configPath)
	configExists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-stat-failed",
			Subject:  configPath,
			Message:  err.Error(),
			Path:     configPath,
		})
		return cfg, false, configPaths
	}

	loaded, err := config.Load(configPath)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-parse-failed",
			Subject:  configPath,
			Message:  err.Error(),
			Path:     configPath,
		})
		return cfg, false, configPaths
	}
	cfg = loaded

	if !configExists {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityInfo,
			Code:     "config-missing",
			Subject:  configPath,
			Message:  "config file not found; defaults are in effect",
			Path:     configPath,
		})
	} else if info != nil && info.IsDir() {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-invalid",
			Subject:  configPath,
			Message:  "config path is a directory, not a file",
			Path:     configPath,
		})
		return cfg, false, configPaths
	}

	repoRoot, err := config.RepoRootPath(cfg)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "repo-root-invalid",
			Subject:  "repo_root",
			Message:  err.Error(),
		})
	} else {
		configPaths.repoRoot = repoRoot
	}

	worktreeRoot, err := config.WorktreeRootPath(cfg)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "worktree-root-invalid",
			Subject:  "worktree_root",
			Message:  err.Error(),
		})
	} else {
		configPaths.worktreeRoot = worktreeRoot
	}

	sharedSkillsDir, err := config.SharedSkillsDirPath(cfg)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "shared-skills-dir-invalid",
			Subject:  "shared_skills_dir",
			Message:  err.Error(),
		})
	} else {
		configPaths.sharedSkills = sharedSkillsDir
	}

	sharedClaudeSkillsDir, err := config.SharedClaudeSkillsDirPath(cfg)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "shared-claude-skills-dir-invalid",
			Subject:  "shared_claude_skills_dir",
			Message:  err.Error(),
		})
	} else {
		configPaths.sharedClaudeDir = sharedClaudeSkillsDir
	}

	if configPaths.repoRoot != "" && configPaths.worktreeRoot != "" && configPaths.repoRoot == configPaths.worktreeRoot {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityWarn,
			Code:     "shared-storage-path",
			Subject:  configPaths.repoRoot,
			Message:  "repo_root and worktree_root resolve to the same path",
			Hint:     "set repo_root and worktree_root to separate directories",
			Path:     configPaths.repoRoot,
		})
	}

	if configPaths.sharedSkills != "" && configPaths.sharedClaudeDir != "" && configPaths.sharedSkills == configPaths.sharedClaudeDir {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityWarn,
			Code:     "shared-install-path",
			Subject:  configPaths.sharedSkills,
			Message:  "shared_skills_dir and shared_claude_skills_dir resolve to the same path",
			Hint:     "set shared_skills_dir and shared_claude_skills_dir to separate directories",
			Path:     configPaths.sharedSkills,
		})
	}

	return cfg, onlySectionErrors(report, SectionConfig) == 0, configPaths
}

func onlySectionErrors(report *Report, section string) int {
	count := 0
	for _, finding := range report.Findings {
		if finding.Section == section && finding.Severity == SeverityError {
			count++
		}
	}
	return count
}

func checkProjectWorkspace(ctx context.Context, report *Report, cwd string, gitAvailable bool) {
	report.Target = cwd
	manifestPath := project.ManifestPath(cwd)
	statePath := project.StatePath(cwd)
	localConfigPath := project.LocalConfigPath(cwd)

	cacheConfig, ok, skipReason := inspectProjectCacheConfig(report, cwd, localConfigPath)
	if !ok {
		addProjectOwnershipFindings(ctx, report, cwd)
		addSkippedSections(report, skipReason)
		return
	}

	addProjectOwnershipFindings(ctx, report, cwd)

	manifest, ok, skipReason := inspectWorkspaceInputs(report, manifestPath, statePath, "project", "skills init")
	if !ok {
		addSkippedSections(report, skipReason)
		return
	}

	if len(manifest.Sources) == 0 {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityWarn,
			Code:     "no-sources-declared",
			Subject:  manifestPath,
			Message:  "no project sources declared",
			Hint:     "declare at least one source in .agents/manifest.yaml",
			Path:     manifestPath,
		})
	}
	if len(manifest.Skills) == 0 {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityWarn,
			Code:     "no-skills-declared",
			Subject:  manifestPath,
			Message:  "no project skills declared",
			Hint:     "declare at least one skill in .agents/manifest.yaml",
			Path:     manifestPath,
		})
	}

	if !gitAvailable {
		addSkippedSections(report, "git is not available")
		return
	}

	status, err := project.Status(ctx, cwd)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "workspace-invalid",
			Subject:  manifestPath,
			Message:  err.Error(),
			Path:     manifestPath,
		})
		addSkippedSections(report, "workspace inputs could not be resolved")
		return
	}

	addStatusFindings(report, status, ScopeProject)

	_ = cacheConfig
}

func inspectProjectCacheConfig(report *Report, cwd string, localConfigPath string) (project.ProjectCacheConfig, bool, string) {
	cacheConfig, err := project.LoadLocalConfig(cwd)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "local-config-parse-failed",
			Subject:  localConfigPath,
			Message:  err.Error(),
			Path:     localConfigPath,
		})
		return project.ProjectCacheConfig{}, false, "project cache settings could not be parsed"
	}

	if !cacheConfig.Exists {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityWarn,
			Code:     "local-config-missing",
			Subject:  localConfigPath,
			Message:  "project cache mode is not explicitly configured; implicit local mode is in effect",
			Hint:     "run skills init --cache=local or skills init --cache=global",
			Path:     localConfigPath,
		})
	}

	report.addFinding(Finding{
		Section:  SectionConfig,
		Severity: SeverityInfo,
		Code:     "project-cache-mode",
		Subject:  string(cacheConfig.Mode),
		Message:  "project install scope is repo-local and cache mode is " + string(cacheConfig.Mode),
		Path:     localConfigPath,
	})

	if cacheConfig.Mode == project.CacheModeLocal {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityInfo,
			Code:     "config-not-required",
			Subject:  "project scope",
			Message:  "local project cache mode does not require global config",
		})
		return cacheConfig, true, ""
	}

	configPath, err := config.DefaultConfigPath()
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-path-failed",
			Subject:  "config",
			Message:  err.Error(),
		})
		return cacheConfig, false, "global cache config path could not be resolved"
	}

	cfg, usable, configPaths := loadAndValidateProjectCacheConfig(report, configPath)
	if !usable {
		return cacheConfig, false, "global cache config could not be loaded"
	}

	report.addFinding(Finding{
		Section:  SectionConfig,
		Severity: SeverityInfo,
		Code:     "project-cache-roots",
		Subject:  string(cacheConfig.Mode),
		Message:  "project uses global cache roots from global config",
		Path:     configPaths.repoRoot,
		Target:   configPaths.worktreeRoot,
	})

	_ = cfg
	return cacheConfig, true, ""
}

func loadAndValidateProjectCacheConfig(report *Report, configPath string) (config.Config, bool, paths) {
	cfg := config.DefaultConfig()
	configPaths := paths{}

	info, err := os.Stat(configPath)
	configExists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-stat-failed",
			Subject:  configPath,
			Message:  err.Error(),
			Path:     configPath,
		})
		return cfg, false, configPaths
	}

	loaded, err := config.Load(configPath)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-parse-failed",
			Subject:  configPath,
			Message:  err.Error(),
			Path:     configPath,
		})
		return cfg, false, configPaths
	}
	cfg = loaded

	if !configExists {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityInfo,
			Code:     "config-missing",
			Subject:  configPath,
			Message:  "global config file not found; defaults are in effect for global cache mode",
			Path:     configPath,
		})
	} else if info != nil && info.IsDir() {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-invalid",
			Subject:  configPath,
			Message:  "config path is a directory, not a file",
			Path:     configPath,
		})
		return cfg, false, configPaths
	}

	repoRoot, err := config.RepoRootPath(cfg)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "repo-root-invalid",
			Subject:  "repo_root",
			Message:  err.Error(),
		})
	} else {
		configPaths.repoRoot = repoRoot
	}

	worktreeRoot, err := config.WorktreeRootPath(cfg)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "worktree-root-invalid",
			Subject:  "worktree_root",
			Message:  err.Error(),
		})
	} else {
		configPaths.worktreeRoot = worktreeRoot
	}

	if configPaths.repoRoot != "" && configPaths.worktreeRoot != "" && configPaths.repoRoot == configPaths.worktreeRoot {
		report.addFinding(Finding{
			Section:  SectionConfig,
			Severity: SeverityWarn,
			Code:     "shared-storage-path",
			Subject:  configPaths.repoRoot,
			Message:  "repo_root and worktree_root resolve to the same path",
			Hint:     "set repo_root and worktree_root to separate directories",
			Path:     configPaths.repoRoot,
		})
	}

	return cfg, onlySectionErrors(report, SectionConfig) == 0, configPaths
}

func addProjectOwnershipFindings(ctx context.Context, report *Report, projectDir string) {
	ownership, err := project.InspectProjectOwnershipContext(ctx, projectDir)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionGit,
			Severity: SeverityError,
			Code:     "git-inspection-failed",
			Subject:  projectDir,
			Message:  err.Error(),
			Path:     projectDir,
		})
		return
	}

	if !ownership.GitAvailable {
		report.addFinding(Finding{
			Section:  SectionGit,
			Severity: SeverityInfo,
			Code:     "git-unavailable",
			Subject:  ownership.GitignorePath,
			Message:  "git-aware ownership checks were skipped because git is not available",
			Path:     ownership.GitignorePath,
		})
		return
	}

	if !ownership.InGitRepo {
		report.addFinding(Finding{
			Section:  SectionGit,
			Severity: SeverityInfo,
			Code:     "git-repo-not-found",
			Subject:  ownership.GitignorePath,
			Message:  "no enclosing Git repo found; using the project-local .gitignore",
			Path:     ownership.GitignorePath,
		})
	} else {
		report.addFinding(Finding{
			Section:  SectionGit,
			Severity: SeverityInfo,
			Code:     "git-repo-root",
			Subject:  ownership.GitRoot,
			Message:  "using the enclosing Git repo root for ignore management",
			Path:     ownership.GitignorePath,
		})
	}

	if len(ownership.MissingRules) > 0 {
		report.addFinding(Finding{
			Section:  SectionGit,
			Severity: SeverityWarn,
			Code:     "ignore-rules-missing",
			Subject:  ownership.GitignorePath,
			Message:  "missing ignore rules for managed runtime artifacts",
			Hint:     "run skills init",
			Path:     ownership.GitignorePath,
		})
	}

	for _, trackedPath := range ownership.TrackedPaths {
		report.addFinding(Finding{
			Section:  SectionGit,
			Severity: SeverityError,
			Code:     "tracked-managed-path",
			Subject:  trackedPath,
			Message:  "managed runtime artifacts should not be tracked by Git",
			Hint:     "move or remove the tracked content from managed paths, then run skills init",
			Path:     trackedPath,
		})
	}
}

func checkGlobalWorkspace(ctx context.Context, report *Report, cfg config.Config, configUsable bool, gitAvailable bool, configPaths paths) {
	if !configUsable {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityInfo,
			Code:     "not-checked",
			Subject:  "home workspace",
			Message:  "home workspace was not checked because global config could not be loaded",
		})
		addSkippedSections(report, "global config could not be loaded")
		return
	}

	manifestPath, err := project.HomeManifestPath(cfg)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "home-manifest-path-failed",
			Subject:  "home workspace",
			Message:  err.Error(),
		})
		addSkippedSections(report, "home workspace paths could not be resolved")
		return
	}

	statePath, err := project.HomeStatePath(cfg)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "home-state-path-failed",
			Subject:  "home workspace",
			Message:  err.Error(),
		})
		addSkippedSections(report, "home workspace paths could not be resolved")
		return
	}

	report.Target = configPaths.sharedSkills
	manifest, ok, skipReason := inspectWorkspaceInputs(report, manifestPath, statePath, "home", "skills init --global")
	if !ok {
		if hasFinding(report.Findings, SectionWorkspace, "manifest-missing") {
			report.Findings[len(report.Findings)-1].Severity = SeverityWarn
		}
		addSkippedSections(report, skipReason)
		return
	}

	if len(manifest.Sources) == 0 {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityWarn,
			Code:     "no-sources-declared",
			Subject:  manifestPath,
			Message:  "no home sources declared",
			Hint:     "declare at least one source in the home manifest",
			Path:     manifestPath,
		})
	}
	if len(manifest.Skills) == 0 {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityWarn,
			Code:     "no-skills-declared",
			Subject:  manifestPath,
			Message:  "no home skills declared",
			Hint:     "declare at least one skill in the home manifest",
			Path:     manifestPath,
		})
	}

	if !gitAvailable {
		addSkippedSections(report, "git is not available")
		return
	}

	status, err := project.HomeStatus(ctx, cfg)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "workspace-invalid",
			Subject:  manifestPath,
			Message:  err.Error(),
			Path:     manifestPath,
		})
		addSkippedSections(report, "workspace inputs could not be resolved")
		return
	}

	addStatusFindings(report, status, ScopeGlobal)
}

func inspectWorkspaceInputs(report *Report, manifestPath string, statePath string, workspaceLabel string, initHint string) (project.Manifest, bool, string) {
	if _, err := os.Stat(manifestPath); errors.Is(err, os.ErrNotExist) {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "manifest-missing",
			Subject:  manifestPath,
			Message:  workspaceLabel + " manifest not found",
			Hint:     "run " + initHint,
			Path:     manifestPath,
		})
		return project.Manifest{}, false, workspaceLabel + " manifest is missing"
	} else if err != nil {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "manifest-stat-failed",
			Subject:  manifestPath,
			Message:  err.Error(),
			Path:     manifestPath,
		})
		return project.Manifest{}, false, workspaceLabel + " manifest could not be read"
	}

	manifest, err := project.LoadManifestAt(manifestPath)
	if err != nil {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "manifest-parse-failed",
			Subject:  manifestPath,
			Message:  err.Error(),
			Path:     manifestPath,
		})
		return project.Manifest{}, false, workspaceLabel + " manifest could not be parsed"
	}

	if _, err := os.Stat(statePath); errors.Is(err, os.ErrNotExist) {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityInfo,
			Code:     "state-missing",
			Subject:  statePath,
			Message:  "state file not found; sync has not been run yet",
			Path:     statePath,
		})
	} else if err != nil {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "state-stat-failed",
			Subject:  statePath,
			Message:  err.Error(),
			Path:     statePath,
		})
		return project.Manifest{}, false, workspaceLabel + " state could not be read"
	} else if _, err := project.LoadStateAt(statePath); err != nil {
		report.addFinding(Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "state-parse-failed",
			Subject:  statePath,
			Message:  err.Error(),
			Path:     statePath,
		})
		return project.Manifest{}, false, workspaceLabel + " state could not be parsed"
	}

	return manifest, true, ""
}

func hasFinding(findings []Finding, section string, code string) bool {
	for _, finding := range findings {
		if finding.Section == section && finding.Code == code {
			return true
		}
	}
	return false
}

func addSkippedSections(report *Report, reason string) {
	for _, section := range []string{SectionSources, SectionSkills, SectionClaude} {
		if hasFinding(report.Findings, section, "not-checked") {
			continue
		}
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityInfo,
			Code:     "not-checked",
			Subject:  section,
			Message:  "not checked because " + reason,
		})
	}
}

func addStatusFindings(report *Report, status project.StatusReport, scope Scope) {
	for _, src := range status.Sources {
		switch src.Status {
		case "up-to-date":
			continue
		case "not-synced":
			report.addFinding(Finding{
				Section:  SectionSources,
				Severity: SeverityWarn,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source has not been synced into this workspace"),
				Hint:     syncHint(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "update-available":
			report.addFinding(Finding{
				Section:  SectionSources,
				Severity: SeverityWarn,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source has a newer resolved commit available"),
				Hint:     updateHint(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "missing-source":
			report.addFinding(Finding{
				Section:  SectionSources,
				Severity: SeverityError,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "canonical source repo is not cloned"),
				Hint:     syncHint(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "invalid-source":
			report.addFinding(Finding{
				Section:  SectionSources,
				Severity: SeverityError,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source path exists but is not a valid git repository"),
				Hint:     "fix or remove the source path and re-run " + syncCommand(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "invalid-ref":
			report.addFinding(Finding{
				Section:  SectionSources,
				Severity: SeverityError,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source ref could not be resolved"),
				Hint:     "confirm the ref and re-run " + syncCommand(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "inspect-failed":
			report.addFinding(Finding{
				Section:  SectionSources,
				Severity: SeverityError,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source contents could not be inspected"),
				Hint:     "fix the source repo state and re-run skills doctor",
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		}
	}

	for _, link := range status.SkillLinks {
		addLinkFinding(report, SectionSkills, link, scope)
	}
	for _, link := range status.ClaudeLinks {
		addLinkFinding(report, SectionClaude, link, scope)
	}
	for _, link := range status.StaleSkillLinks {
		report.addFinding(Finding{
			Section:  SectionSkills,
			Severity: SeverityWarn,
			Code:     "stale-managed-link",
			Subject:  link.Source + "/" + link.Skill,
			Message:  "managed skill link is no longer declared",
			Hint:     syncHint(scope),
			Path:     link.Path,
			Target:   link.Target,
		})
	}
	for _, link := range status.StaleClaudeLinks {
		report.addFinding(Finding{
			Section:  SectionClaude,
			Severity: SeverityWarn,
			Code:     "stale-managed-link",
			Subject:  link.Source + "/" + link.Skill,
			Message:  "managed Claude adapter link is no longer declared",
			Hint:     syncHint(scope),
			Path:     link.Path,
			Target:   link.Target,
		})
	}
}

func addLinkFinding(report *Report, section string, link project.LinkReport, scope Scope) {
	subject := link.Source + "/" + link.Skill
	switch link.Status {
	case "linked":
		return
	case "missing":
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityWarn,
			Code:     link.Status,
			Subject:  subject,
			Message:  "declared link is missing",
			Hint:     syncHint(scope),
			Path:     link.Path,
			Target:   link.Target,
		})
	case "stale":
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityWarn,
			Code:     link.Status,
			Subject:  subject,
			Message:  "managed link points at an older target",
			Hint:     syncHint(scope),
			Path:     link.Path,
			Target:   link.Target,
		})
	case "conflict":
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "destination exists as a non-symlink or unmanaged symlink",
			Hint:     "remove or move the conflicting path and re-run " + syncCommand(scope),
			Path:     link.Path,
			Target:   link.Target,
		})
	case "invalid":
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "link could not be inspected",
			Hint:     "inspect the destination path and re-run " + syncCommand(scope),
			Path:     link.Path,
			Target:   link.Target,
		})
	case "unknown-source":
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "skill references a source that is not declared",
			Hint:     "update the manifest so the skill references a declared source",
			Path:     link.Path,
		})
	case "source-not-ready":
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "source is not ready for skill resolution",
			Hint:     "fix source errors and re-run " + syncCommand(scope),
			Path:     link.Path,
		})
	case "inspect-failed":
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  firstNonEmpty(link.Message, "source contents could not be inspected"),
			Hint:     "fix the source repo state and re-run " + syncCommand(scope),
			Path:     link.Path,
		})
	case "missing-skill":
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "declared skill name was not found in the source",
			Hint:     "run skills skill list --source " + link.Source + " and update the manifest",
			Path:     link.Path,
		})
	case "ambiguous-skill":
		report.addFinding(Finding{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  firstNonEmpty(link.Message, "multiple skills share this directory name"),
			Hint:     "rename one of the duplicate skill directories upstream or choose another source",
			Path:     link.Path,
		})
	}
}

func syncCommand(scope Scope) string {
	if scope == ScopeGlobal {
		return "skills sync --global"
	}
	return "skills sync"
}

func syncHint(scope Scope) string {
	return "run " + syncCommand(scope)
}

func updateHint(scope Scope) string {
	if scope == ScopeGlobal {
		return "run skills update --global --sync"
	}
	return "run skills update --sync"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
