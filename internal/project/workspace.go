package project

import (
	"fmt"
	"path/filepath"

	"github.com/mattgiles/skills/internal/config"
)

type workspace struct {
	Name            string
	RootDir         string
	ManifestPath    string
	StatePath       string
	SkillsDir       string
	ClaudeSkillsDir string
	CacheDir        string
	RepoRoot        string
	WorktreeRoot    string
	CacheMode       CacheMode
	LocalConfigPath string
}

func ManifestPath(projectDir string) string {
	return filepath.Join(projectDir, ManifestFilename)
}

func StatePath(projectDir string) string {
	return filepath.Join(projectDir, StateFilename)
}

func LocalConfigPath(projectDir string) string {
	return filepath.Join(projectDir, LocalConfigFilename)
}

func SkillsDir(projectDir string) string {
	return filepath.Join(projectDir, SkillsDirname)
}

func ClaudeSkillsDir(projectDir string) string {
	return filepath.Join(projectDir, ClaudeSkillsDirname)
}

func CacheDir(projectDir string) string {
	return filepath.Join(projectDir, CacheDirname)
}

func RepoRoot(projectDir string) string {
	return filepath.Join(projectDir, RepoCacheDirname)
}

func WorktreeRoot(projectDir string) string {
	return filepath.Join(projectDir, WorktreeCacheDirname)
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

func projectWorkspace(projectDir string, cacheMode CacheMode) (workspace, error) {
	if cacheMode != CacheModeLocal && cacheMode != CacheModeGlobal {
		return workspace{}, fmt.Errorf("invalid cache mode %q: use local or global", cacheMode)
	}

	repoRoot := RepoRoot(projectDir)
	worktreeRoot := WorktreeRoot(projectDir)
	if cacheMode == CacheModeGlobal {
		cfg, err := loadGlobalConfig()
		if err != nil {
			return workspace{}, err
		}
		repoRoot, err = config.RepoRootPath(cfg)
		if err != nil {
			return workspace{}, err
		}
		worktreeRoot, err = config.WorktreeRootPath(cfg)
		if err != nil {
			return workspace{}, err
		}
	}

	return workspace{
		Name:            projectWorkspaceName,
		RootDir:         projectDir,
		ManifestPath:    ManifestPath(projectDir),
		StatePath:       StatePath(projectDir),
		LocalConfigPath: LocalConfigPath(projectDir),
		SkillsDir:       SkillsDir(projectDir),
		ClaudeSkillsDir: ClaudeSkillsDir(projectDir),
		CacheDir:        CacheDir(projectDir),
		RepoRoot:        repoRoot,
		WorktreeRoot:    worktreeRoot,
		CacheMode:       cacheMode,
	}, nil
}

func resolveProjectWorkspace(projectDir string) (workspace, error) {
	cacheConfig, err := LoadLocalConfig(projectDir)
	if err != nil {
		return workspace{}, err
	}
	return projectWorkspace(projectDir, cacheConfig.Mode)
}

func homeWorkspace(cfg config.Config) (workspace, error) {
	repoRoot, err := config.RepoRootPath(cfg)
	if err != nil {
		return workspace{}, err
	}
	worktreeRoot, err := config.WorktreeRootPath(cfg)
	if err != nil {
		return workspace{}, err
	}
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
		RepoRoot:        repoRoot,
		WorktreeRoot:    worktreeRoot,
	}, nil
}

func loadGlobalConfig() (config.Config, error) {
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		return config.Config{}, err
	}
	return config.Load(configPath)
}
