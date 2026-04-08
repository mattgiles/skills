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

type workspaceTarget struct {
	Scope       commandScope
	TargetDir   string
	ProjectRoot string
	Config      config.Config
	Summary     workspaceSummary
}

func resolveWorkspaceTarget(ctx context.Context, global bool, requireManifest bool) (workspaceTarget, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return workspaceTarget{}, err
	}

	if global {
		cfg, err := loadConfig()
		if err != nil {
			return workspaceTarget{}, err
		}
		summary, err := globalWorkspaceSummary(ctx, cfg)
		if err != nil {
			return workspaceTarget{}, err
		}
		return workspaceTarget{
			Scope:     scopeGlobal,
			TargetDir: cwd,
			Config:    cfg,
			Summary:   summary,
		}, nil
	}

	projectRoot, err := resolveRepoRoot(ctx, cwd, requireManifest)
	if err != nil {
		return workspaceTarget{}, err
	}
	summary, err := repoWorkspaceSummary(ctx, projectRoot)
	if err != nil {
		return workspaceTarget{}, err
	}
	return workspaceTarget{
		Scope:       scopeRepo,
		TargetDir:   projectRoot,
		ProjectRoot: projectRoot,
		Summary:     summary,
	}, nil
}

func resolveRepoRoot(ctx context.Context, cwd string, requireManifest bool) (string, error) {
	info, err := gitrepo.Discover(ctx, cwd)
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

func repoWorkspaceSummary(_ context.Context, projectRoot string) (workspaceSummary, error) {
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

func globalWorkspaceSummary(_ context.Context, cfg config.Config) (workspaceSummary, error) {
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
