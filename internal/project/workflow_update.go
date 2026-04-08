package project

import (
	"context"

	"github.com/mattgiles/skills/internal/config"
)

func Update(ctx context.Context, projectDir string, options UpdateOptions) (UpdateResult, error) {
	ws, err := resolveProjectWorkspace(projectDir)
	if err != nil {
		return UpdateResult{}, err
	}
	return updateWorkspace(ctx, ws, options)
}

func HomeUpdate(ctx context.Context, cfg config.Config, options UpdateOptions) (UpdateResult, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return UpdateResult{}, err
	}
	return updateWorkspace(ctx, ws, options)
}

func updateWorkspace(ctx context.Context, ws workspace, options UpdateOptions) (UpdateResult, error) {
	manifest, state, resolvedSources, err := loadWorkspaceInputs(ws)
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
