package project

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattgiles/skills/internal/discovery"
	"github.com/mattgiles/skills/internal/source"
)

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

func loadWorkspaceInputs(ws workspace) (Manifest, State, map[string]*resolvedSource, error) {
	manifest, err := LoadManifestAt(ws.ManifestPath)
	if err != nil {
		return Manifest{}, State{}, nil, err
	}

	state, err := LoadStateAt(ws.StatePath)
	if err != nil {
		return Manifest{}, State{}, nil, err
	}

	resolvedSources, err := resolveInputs(ws, manifest)
	if err != nil {
		return Manifest{}, State{}, nil, err
	}

	return manifest, state, resolvedSources, nil
}

func resolveInputs(ws workspace, manifest Manifest) (map[string]*resolvedSource, error) {
	workspaceID, err := ProjectID(ws.RootDir)
	if err != nil {
		return nil, err
	}

	resolvedSources := map[string]*resolvedSource{}
	for alias, manifestSource := range manifest.Sources {
		url := manifestSource.URL
		if strings.TrimSpace(url) == "" {
			if ws.Name == projectWorkspaceName {
				return nil, fmt.Errorf("source %q has no URL in project manifest", alias)
			}
			return nil, fmt.Errorf("source %q has no URL in home manifest", alias)
		}

		resolvedSources[alias] = &resolvedSource{
			Alias:        alias,
			URL:          url,
			Ref:          manifestSource.Ref,
			RepoPath:     source.RepoPath(ws.RepoRoot, alias),
			WorktreeRoot: ws.WorktreeRoot,
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
