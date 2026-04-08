package project

import (
	"context"

	"github.com/mattgiles/skills/internal/config"
)

func Status(ctx context.Context, projectDir string) (StatusReport, error) {
	ws, err := resolveProjectWorkspace(projectDir)
	if err != nil {
		return StatusReport{}, err
	}
	return statusWorkspace(ctx, ws)
}

func HomeStatus(ctx context.Context, cfg config.Config) (StatusReport, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return StatusReport{}, err
	}
	return statusWorkspace(ctx, ws)
}

func statusWorkspace(ctx context.Context, ws workspace) (StatusReport, error) {
	manifest, state, resolvedSources, err := loadWorkspaceInputs(ws)
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
	staleSkillLinks := staleLinks(state.SkillLinks, desiredLinkMap(desiredSkillLinks))

	desiredClaudeLinks, claudeReports := buildClaudeLinkReports(desiredSkillLinks, ws.ClaudeSkillsDir, claudeStateLinks)
	staleClaudeLinks := staleLinks(state.ClaudeLinks, desiredLinkMap(desiredClaudeLinks))

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
