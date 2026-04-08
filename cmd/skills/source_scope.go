package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/source"
)

type sourceManifestTarget struct {
	Scope        commandScope
	ManifestPath string
	Manifest     project.Manifest
	RepoRoot     string
	ProjectRoot  string
	Config       config.Config
	Summary      workspaceSummary
}

func resolveSourceManifestTarget(ctx context.Context, global bool) (sourceManifestTarget, error) {
	target, err := resolveWorkspaceTarget(ctx, global, true)
	if err != nil {
		return sourceManifestTarget{}, err
	}

	if target.Scope == scopeGlobal {
		manifestPath, err := project.HomeManifestPath(target.Config)
		if err != nil {
			return sourceManifestTarget{}, err
		}
		manifest, err := project.LoadManifestAt(manifestPath)
		if err != nil {
			return sourceManifestTarget{}, err
		}
		return sourceManifestTarget{
			Scope:        scopeGlobal,
			ManifestPath: manifestPath,
			Manifest:     manifest,
			RepoRoot:     target.Summary.RepoRoot,
			Config:       target.Config,
			Summary:      target.Summary,
		}, nil
	}

	manifest, err := project.LoadManifest(target.ProjectRoot)
	if err != nil {
		return sourceManifestTarget{}, err
	}

	return sourceManifestTarget{
		Scope:        scopeRepo,
		ManifestPath: project.ManifestPath(target.ProjectRoot),
		Manifest:     manifest,
		RepoRoot:     target.Summary.RepoRoot,
		ProjectRoot:  target.ProjectRoot,
		Summary:      target.Summary,
	}, nil
}

func resolveManifestSources(ctx context.Context, global bool, aliases []string) ([]source.Source, error) {
	target, err := resolveSourceManifestTarget(ctx, global)
	if err != nil {
		return nil, err
	}
	return selectManifestSources(target.Manifest, target.RepoRoot, aliases, global)
}

func selectManifestSources(manifest project.Manifest, repoRoot string, aliases []string, global bool) ([]source.Source, error) {
	if len(aliases) == 0 {
		aliases = make([]string, 0, len(manifest.Sources))
		for alias := range manifest.Sources {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
	}

	selected := make([]source.Source, 0, len(aliases))
	seen := map[string]struct{}{}
	scopeLabel := "repo"
	if global {
		scopeLabel = "home"
	}

	for _, alias := range aliases {
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}

		entry, ok := manifest.Sources[alias]
		if !ok {
			return nil, fmt.Errorf("unknown source %q", alias)
		}
		if strings.TrimSpace(entry.URL) == "" {
			return nil, fmt.Errorf("source %q has no URL in %s manifest", alias, scopeLabel)
		}

		selected = append(selected, source.Source{
			Alias:    alias,
			Ref:      entry.Ref,
			URL:      entry.URL,
			RepoPath: source.RepoPath(repoRoot, alias),
		})
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Alias < selected[j].Alias
	})
	return selected, nil
}
