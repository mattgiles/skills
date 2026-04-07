package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/gitrepo"
	"github.com/mattgiles/skills/internal/project"
)

type commandScope string

const (
	scopeRepo   commandScope = "repo"
	scopeGlobal commandScope = "global"
)

type workspaceSummary struct {
	Scope        commandScope
	Root         string
	InstallDir   string
	CacheMode    string
	RepoRoot     string
	WorktreeRoot string
}

func resolveRepoRoot(cwd string, requireManifest bool) (string, error) {
	info, err := gitrepo.Discover(context.Background(), cwd)
	if err != nil {
		return "", err
	}
	if info.Root == "" {
		return "", errors.New("outside a Git repo; use --global")
	}

	projectRoot := info.Root
	if !requireManifest {
		return projectRoot, nil
	}

	manifestPath := project.ManifestPath(projectRoot)
	if _, err := os.Stat(manifestPath); err == nil {
		return projectRoot, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("no repo manifest found at %s; run skills init or use --global", manifestPath)
	} else {
		return "", err
	}
}

func repoWorkspaceSummary(projectRoot string) (workspaceSummary, error) {
	cacheConfig, err := project.LoadLocalConfig(projectRoot)
	if err != nil {
		return workspaceSummary{}, err
	}

	summary := workspaceSummary{
		Scope:      scopeRepo,
		Root:       projectRoot,
		InstallDir: project.SkillsDir(projectRoot),
		CacheMode:  string(cacheConfig.Mode),
		RepoRoot:   project.RepoRoot(projectRoot),
	}

	if cacheConfig.Mode == project.CacheModeGlobal {
		cfg, err := loadConfig()
		if err != nil {
			return workspaceSummary{}, err
		}
		repoRoot, err := config.RepoRootPath(cfg)
		if err != nil {
			return workspaceSummary{}, err
		}
		worktreeRoot, err := config.WorktreeRootPath(cfg)
		if err != nil {
			return workspaceSummary{}, err
		}
		summary.RepoRoot = repoRoot
		summary.WorktreeRoot = worktreeRoot
		return summary, nil
	}

	summary.WorktreeRoot = project.WorktreeRoot(projectRoot)
	return summary, nil
}

func globalWorkspaceSummary(cfg config.Config) (workspaceSummary, error) {
	installDir, err := config.SharedSkillsDirPath(cfg)
	if err != nil {
		return workspaceSummary{}, err
	}
	repoRoot, err := config.RepoRootPath(cfg)
	if err != nil {
		return workspaceSummary{}, err
	}
	worktreeRoot, err := config.WorktreeRootPath(cfg)
	if err != nil {
		return workspaceSummary{}, err
	}

	return workspaceSummary{
		Scope:        scopeGlobal,
		InstallDir:   installDir,
		RepoRoot:     repoRoot,
		WorktreeRoot: worktreeRoot,
	}, nil
}
