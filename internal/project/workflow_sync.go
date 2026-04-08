package project

import (
	"context"
	"strings"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/source"
)

type preparedSync struct {
	sourceReports      []SourceReport
	sourceStates       []SourceState
	desiredSkillLinks  []desiredLink
	skillReports       []LinkReport
	skillActions       []linkAction
	desiredClaudeLinks []desiredLink
	claudeReports      []LinkReport
	claudeActions      []linkAction
	staleSkillLinks    []ManagedLink
	prunedSkillLinks   []string
	staleClaudeLinks   []ManagedLink
	prunedClaudeLinks  []string
}

func Sync(ctx context.Context, projectDir string, options SyncOptions) (SyncResult, error) {
	ws, err := resolveProjectWorkspace(projectDir)
	if err != nil {
		return SyncResult{}, err
	}
	return syncWorkspace(ctx, ws, options)
}

func HomeSync(ctx context.Context, cfg config.Config, options SyncOptions) (SyncResult, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return SyncResult{}, err
	}
	return syncWorkspace(ctx, ws, options)
}

func syncWorkspace(ctx context.Context, ws workspace, options SyncOptions) (SyncResult, error) {
	manifest, state, resolvedSources, err := loadWorkspaceInputs(ws)
	if err != nil {
		return SyncResult{}, err
	}
	return runSyncWorkspace(ctx, ws, manifest, state, resolvedSources, options.DryRun)
}

func syncWorkspaceWithState(ctx context.Context, ws workspace, manifest Manifest, state State, resolvedSources map[string]*resolvedSource, dryRun bool) (SyncResult, error) {
	return runSyncWorkspace(ctx, ws, manifest, state, resolvedSources, dryRun)
}

func runSyncWorkspace(ctx context.Context, ws workspace, manifest Manifest, state State, resolvedSources map[string]*resolvedSource, dryRun bool) (SyncResult, error) {
	prepared, err := prepareSyncWorkspace(ctx, ws, manifest, state, resolvedSources, dryRun)
	if err != nil {
		return SyncResult{}, err
	}
	if dryRun {
		return buildSyncResult(prepared, true), nil
	}
	if err := applySyncWorkspace(ctx, ws, resolvedSources, prepared); err != nil {
		return SyncResult{}, err
	}
	return buildSyncResult(prepared, false), nil
}

func prepareSyncWorkspace(ctx context.Context, ws workspace, manifest Manifest, state State, resolvedSources map[string]*resolvedSource, dryRun bool) (preparedSync, error) {
	stateSources := sourceStateMap(state)
	skillStateLinks := managedLinkMap(state.SkillLinks)
	claudeStateLinks := managedLinkMap(state.ClaudeLinks)

	sourceReports, sourceStates, err := resolveSourcesForSync(ctx, resolvedSources, stateSources)
	if err != nil {
		return preparedSync{}, err
	}

	desiredSkillLinks, skillReports := buildSkillLinkReports(resolvedSources, manifest, ws.SkillsDir, skillStateLinks)
	if err := validateDesiredLinks(desiredSkillLinks); err != nil {
		return preparedSync{}, err
	}
	if err := fatalSkillLinkReports(skillReports); err != nil {
		return preparedSync{}, err
	}

	skillActions, err := planLinkActions(desiredSkillLinks, skillStateLinks)
	if err != nil {
		return preparedSync{}, err
	}
	updateLinkReportsForActions(skillReports, skillActions, dryRun)

	desiredClaudeLinks, claudeReports := buildClaudeLinkReports(desiredSkillLinks, ws.ClaudeSkillsDir, claudeStateLinks)
	if err := validateDesiredLinks(desiredClaudeLinks); err != nil {
		return preparedSync{}, err
	}
	if err := fatalAdapterLinkReports(claudeReports); err != nil {
		return preparedSync{}, err
	}

	claudeActions, err := planLinkActions(desiredClaudeLinks, claudeStateLinks)
	if err != nil {
		return preparedSync{}, err
	}
	updateLinkReportsForActions(claudeReports, claudeActions, dryRun)

	staleSkillLinks := staleLinks(state.SkillLinks, desiredLinkMap(desiredSkillLinks))
	prunedSkillLinks := managedLinkPaths(staleSkillLinks)
	staleClaudeLinks := staleLinks(state.ClaudeLinks, desiredLinkMap(desiredClaudeLinks))
	prunedClaudeLinks := managedLinkPaths(staleClaudeLinks)

	return preparedSync{
		sourceReports:      sourceReports,
		sourceStates:       sourceStates,
		desiredSkillLinks:  desiredSkillLinks,
		skillReports:       skillReports,
		skillActions:       skillActions,
		desiredClaudeLinks: desiredClaudeLinks,
		claudeReports:      claudeReports,
		claudeActions:      claudeActions,
		staleSkillLinks:    staleSkillLinks,
		prunedSkillLinks:   prunedSkillLinks,
		staleClaudeLinks:   staleClaudeLinks,
		prunedClaudeLinks:  prunedClaudeLinks,
	}, nil
}

func applySyncWorkspace(ctx context.Context, ws workspace, resolvedSources map[string]*resolvedSource, prepared preparedSync) error {
	for _, src := range sortedResolvedSources(resolvedSources) {
		if strings.TrimSpace(src.DesiredCommit) == "" {
			continue
		}
		if _, err := source.EnsureWorktree(ctx, source.Source{
			Alias:    src.Alias,
			URL:      src.URL,
			RepoPath: src.RepoPath,
		}, src.WorktreePath, src.DesiredCommit); err != nil {
			return err
		}
	}

	if err := applyLinkActions(prepared.skillActions); err != nil {
		return err
	}
	if err := applyLinkActions(prepared.claudeActions); err != nil {
		return err
	}
	if err := removeManagedLinks(prepared.staleClaudeLinks); err != nil {
		return err
	}
	if err := removeManagedLinks(prepared.staleSkillLinks); err != nil {
		return err
	}

	nextState := State{
		Sources:     prepared.sourceStates,
		SkillLinks:  toManagedLinks(prepared.desiredSkillLinks),
		ClaudeLinks: toManagedLinks(prepared.desiredClaudeLinks),
	}
	return SaveStateAt(ws.StatePath, nextState)
}

func buildSyncResult(prepared preparedSync, dryRun bool) SyncResult {
	sourceReports := append([]SourceReport(nil), prepared.sourceReports...)
	if !dryRun {
		for i := range sourceReports {
			sourceReports[i].Status = syncSourceStatus(sourceReports[i].Status)
		}
	}

	result := SyncResult{
		Sources:           sourceReports,
		SkillLinks:        append([]LinkReport(nil), prepared.skillReports...),
		ClaudeLinks:       append([]LinkReport(nil), prepared.claudeReports...),
		PrunedSkillLinks:  append([]string(nil), prepared.prunedSkillLinks...),
		PrunedClaudeLinks: append([]string(nil), prepared.prunedClaudeLinks...),
		DryRun:            dryRun,
	}
	sortSourceReports(result.Sources)
	sortLinkReports(result.SkillLinks)
	sortLinkReports(result.ClaudeLinks)
	return result
}
