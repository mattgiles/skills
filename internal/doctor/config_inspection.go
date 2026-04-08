package doctor

import (
	"errors"
	"os"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/project"
)

type paths struct {
	repoRoot        string
	worktreeRoot    string
	sharedSkills    string
	sharedClaudeDir string
}

type configInspection struct {
	cfg      config.Config
	paths    paths
	usable   bool
	findings []Finding
}

type configInspectionOptions struct {
	missingMessage        string
	includeSharedInstalls bool
}

type projectCacheInspection struct {
	cacheConfig project.ProjectCacheConfig
	usable      bool
	skipReason  string
	findings    []Finding
}

func inspectConfig(configPath string, options configInspectionOptions) configInspection {
	result := configInspection{
		cfg:      config.DefaultConfig(),
		findings: []Finding{},
	}

	info, err := os.Stat(configPath)
	configExists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		result.findings = append(result.findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-stat-failed",
			Subject:  configPath,
			Message:  err.Error(),
			Path:     configPath,
		})
		result.usable = false
		return result
	}
	if info != nil && info.IsDir() {
		result.findings = append(result.findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-invalid",
			Subject:  configPath,
			Message:  "config path is a directory, not a file",
			Path:     configPath,
		})
		result.usable = false
		return result
	}

	loaded, err := config.Load(configPath)
	if err != nil {
		result.findings = append(result.findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-parse-failed",
			Subject:  configPath,
			Message:  err.Error(),
			Path:     configPath,
		})
		result.usable = false
		return result
	}
	result.cfg = loaded

	switch {
	case !configExists:
		result.findings = append(result.findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityInfo,
			Code:     "config-missing",
			Subject:  configPath,
			Message:  options.missingMessage,
			Path:     configPath,
		})
	}

	result.findings = append(result.findings, inspectStoragePaths(result.cfg, options.includeSharedInstalls)...)
	result.usable = countSectionErrors(result.findings, SectionConfig) == 0
	return result
}

func inspectStoragePaths(cfg config.Config, includeSharedInstalls bool) []Finding {
	findings := []Finding{}
	configPaths := paths{}

	repoRoot, err := config.RepoRootPath(cfg)
	if err != nil {
		findings = append(findings, Finding{
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
		findings = append(findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "worktree-root-invalid",
			Subject:  "worktree_root",
			Message:  err.Error(),
		})
	} else {
		configPaths.worktreeRoot = worktreeRoot
	}

	if includeSharedInstalls {
		sharedSkillsDir, err := config.SharedSkillsDirPath(cfg)
		if err != nil {
			findings = append(findings, Finding{
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
			findings = append(findings, Finding{
				Section:  SectionConfig,
				Severity: SeverityError,
				Code:     "shared-claude-skills-dir-invalid",
				Subject:  "shared_claude_skills_dir",
				Message:  err.Error(),
			})
		} else {
			configPaths.sharedClaudeDir = sharedClaudeSkillsDir
		}
	}

	findings = append(findings, sharedPathWarnings(configPaths, includeSharedInstalls)...)
	return findings
}

func sharedPathWarnings(configPaths paths, includeSharedInstalls bool) []Finding {
	findings := []Finding{}

	if configPaths.repoRoot != "" && configPaths.worktreeRoot != "" && configPaths.repoRoot == configPaths.worktreeRoot {
		findings = append(findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityWarn,
			Code:     "shared-storage-path",
			Subject:  configPaths.repoRoot,
			Message:  "repo_root and worktree_root resolve to the same path",
			Hint:     "set repo_root and worktree_root to separate directories",
			Path:     configPaths.repoRoot,
		})
	}

	if includeSharedInstalls &&
		configPaths.sharedSkills != "" &&
		configPaths.sharedClaudeDir != "" &&
		configPaths.sharedSkills == configPaths.sharedClaudeDir {
		findings = append(findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityWarn,
			Code:     "shared-install-path",
			Subject:  configPaths.sharedSkills,
			Message:  "shared_skills_dir and shared_claude_skills_dir resolve to the same path",
			Hint:     "set shared_skills_dir and shared_claude_skills_dir to separate directories",
			Path:     configPaths.sharedSkills,
		})
	}

	return findings
}

func resolveConfigPaths(cfg config.Config, includeSharedInstalls bool) paths {
	configPaths := paths{}

	if repoRoot, err := config.RepoRootPath(cfg); err == nil {
		configPaths.repoRoot = repoRoot
	}
	if worktreeRoot, err := config.WorktreeRootPath(cfg); err == nil {
		configPaths.worktreeRoot = worktreeRoot
	}
	if includeSharedInstalls {
		if sharedSkillsDir, err := config.SharedSkillsDirPath(cfg); err == nil {
			configPaths.sharedSkills = sharedSkillsDir
		}
		if sharedClaudeSkillsDir, err := config.SharedClaudeSkillsDirPath(cfg); err == nil {
			configPaths.sharedClaudeDir = sharedClaudeSkillsDir
		}
	}

	return configPaths
}

func inspectGlobalConfig(configPath string) configInspection {
	result := inspectConfig(configPath, configInspectionOptions{
		missingMessage:        "config file not found; defaults are in effect",
		includeSharedInstalls: true,
	})
	result.paths = resolveConfigPaths(result.cfg, true)
	return result
}

func inspectProjectCacheConfig(cwd string, localConfigPath string) projectCacheInspection {
	result := projectCacheInspection{
		findings: []Finding{},
	}

	cacheConfig, err := project.LoadLocalConfig(cwd)
	if err != nil {
		result.findings = append(result.findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "local-config-parse-failed",
			Subject:  localConfigPath,
			Message:  err.Error(),
			Path:     localConfigPath,
		})
		result.skipReason = "project cache settings could not be parsed"
		return result
	}
	result.cacheConfig = cacheConfig

	if !cacheConfig.Exists {
		result.findings = append(result.findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityWarn,
			Code:     "local-config-missing",
			Subject:  localConfigPath,
			Message:  "project cache mode is not explicitly configured; implicit local mode is in effect",
			Hint:     "run skills init --cache=local or skills init --cache=global",
			Path:     localConfigPath,
		})
	}

	result.findings = append(result.findings, Finding{
		Section:  SectionConfig,
		Severity: SeverityInfo,
		Code:     "project-cache-mode",
		Subject:  string(cacheConfig.Mode),
		Message:  "project install scope is repo-local and cache mode is " + string(cacheConfig.Mode),
		Path:     localConfigPath,
	})

	if cacheConfig.Mode == project.CacheModeLocal {
		result.findings = append(result.findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityInfo,
			Code:     "config-not-required",
			Subject:  "project scope",
			Message:  "local project cache mode does not require global config",
		})
		result.usable = true
		return result
	}

	configPath, err := defaultConfigPath()
	if err != nil {
		result.findings = append(result.findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-path-failed",
			Subject:  "config",
			Message:  err.Error(),
		})
		result.skipReason = "global cache config path could not be resolved"
		return result
	}

	configInspection := inspectConfig(configPath, configInspectionOptions{
		missingMessage:        "global config file not found; defaults are in effect for global cache mode",
		includeSharedInstalls: false,
	})
	result.findings = append(result.findings, configInspection.findings...)
	if !configInspection.usable {
		result.skipReason = "global cache config could not be loaded"
		return result
	}

	result.findings = append(result.findings, Finding{
		Section:  SectionConfig,
		Severity: SeverityInfo,
		Code:     "project-cache-roots",
		Subject:  string(cacheConfig.Mode),
		Message:  "project uses global cache roots from global config",
		Path:     configInspection.paths.repoRoot,
		Target:   configInspection.paths.worktreeRoot,
	})
	result.usable = true
	return result
}
