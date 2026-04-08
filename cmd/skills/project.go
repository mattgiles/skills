package main

import (
	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/ui"
)

func renderProjectStatus(cmd *cobra.Command, report project.StatusReport, verbose bool) {
	renderWorkspaceStatus(cmd, report, verbose, "no project sources declared", "no project skills declared")
}

func renderWorkspaceStatus(cmd *cobra.Command, report project.StatusReport, verbose bool, noSources string, noSkills string) {
	view := ui.New(cmd)

	if len(report.Sources) == 0 {
		view.Infof("%s", noSources)
	} else {
		_ = view.RenderTable(ui.Table{
			Title:   "Sources",
			Columns: sourceColumns(verbose),
			Rows:    sourceRows(report.Sources, verbose),
		})
	}

	view.Blank()

	if len(report.SkillLinks) == 0 {
		view.Infof("%s", noSkills)
	} else {
		_ = view.RenderTable(ui.Table{
			Title:   "Skills",
			Columns: linkColumns(verbose),
			Rows:    linkRows(report.SkillLinks, verbose),
		})
	}

	if len(report.ClaudeLinks) > 0 {
		view.Blank()
		_ = view.RenderTable(ui.Table{
			Title:   "Claude",
			Columns: linkColumns(verbose),
			Rows:    linkRows(report.ClaudeLinks, verbose),
		})
	}

	if len(report.StaleSkillLinks) > 0 {
		view.Blank()
		_ = view.RenderTable(ui.Table{
			Title:   "Stale Skills",
			Columns: []string{"Stale Path", "Source", "Skill"},
			Rows:    staleLinkRows(report.StaleSkillLinks),
		})
	}

	if len(report.StaleClaudeLinks) > 0 {
		view.Blank()
		_ = view.RenderTable(ui.Table{
			Title:   "Stale Claude",
			Columns: []string{"Stale Path", "Source", "Skill"},
			Rows:    staleLinkRows(report.StaleClaudeLinks),
		})
	}
}

func renderProjectSync(cmd *cobra.Command, result project.SyncResult, verbose bool) {
	renderWorkspaceSync(cmd, result, verbose)
}

func renderCacheClean(cmd *cobra.Command, result project.CacheCleanResult) {
	view := ui.New(cmd)
	_ = view.KeyValues("Cache Clean", [][2]string{
		{"Repos", result.RepoRoot},
		{"Worktrees", result.WorktreeRoot},
	})
}

func renderWorkspaceSync(cmd *cobra.Command, result project.SyncResult, verbose bool) {
	view := ui.New(cmd)

	if result.DryRun {
		view.Infof("dry-run")
		view.Blank()
	}

	if len(result.Sources) > 0 {
		_ = view.RenderTable(ui.Table{
			Title:   "Sources",
			Columns: sourceColumns(verbose),
			Rows:    sourceRows(result.Sources, verbose),
		})
	}

	if len(result.SkillLinks) > 0 {
		view.Blank()
		_ = view.RenderTable(ui.Table{
			Title:   "Skills",
			Columns: linkColumns(verbose),
			Rows:    linkRows(result.SkillLinks, verbose),
		})
	}

	if len(result.ClaudeLinks) > 0 {
		view.Blank()
		_ = view.RenderTable(ui.Table{
			Title:   "Claude",
			Columns: linkColumns(verbose),
			Rows:    linkRows(result.ClaudeLinks, verbose),
		})
	}

	if len(result.PrunedSkillLinks) > 0 {
		view.Blank()
		_ = view.RenderTable(ui.Table{
			Title:   "Pruned Skills",
			Columns: []string{"Pruned Path"},
			Rows:    pathRows(result.PrunedSkillLinks),
		})
	}

	if len(result.PrunedClaudeLinks) > 0 {
		view.Blank()
		_ = view.RenderTable(ui.Table{
			Title:   "Pruned Claude",
			Columns: []string{"Pruned Path"},
			Rows:    pathRows(result.PrunedClaudeLinks),
		})
	}
}

func renderProjectUpdate(cmd *cobra.Command, result project.UpdateResult, verbose bool) {
	view := ui.New(cmd)

	if result.DryRun {
		view.Infof("dry-run")
		view.Blank()
	}

	if len(result.Sources) > 0 {
		_ = view.RenderTable(ui.Table{
			Title:   "Sources",
			Columns: sourceColumns(verbose),
			Rows:    sourceRows(result.Sources, verbose),
		})
	}

	if result.Sync != nil {
		view.Blank()
		renderProjectSync(cmd, *result.Sync, verbose)
	}
}

func sourceColumns(verbose bool) []string {
	if verbose {
		return []string{"Source", "Status", "Ref", "Commit", "Stored", "Repo Path", "Worktree Path", "Message"}
	}
	return []string{"Source", "Status", "Ref", "Commit", "Message"}
}

func sourceRows(sources []project.SourceReport, verbose bool) [][]string {
	rows := make([][]string, 0, len(sources))
	for _, src := range sources {
		if verbose {
			rows = append(rows, []string{
				src.Alias,
				src.Status,
				src.Ref,
				renderVerboseValue(src.Commit),
				renderVerboseValue(src.PreviousCommit),
				renderVerboseValue(src.RepoPath),
				renderVerboseValue(src.WorktreePath),
				renderVerboseValue(src.Message),
			})
			continue
		}

		rows = append(rows, []string{
			src.Alias,
			src.Status,
			src.Ref,
			src.Commit,
			src.Message,
		})
	}
	return rows
}

func linkColumns(verbose bool) []string {
	if verbose {
		return []string{"Source", "Skill", "Status", "Path", "Target", "Message"}
	}
	return []string{"Source", "Skill", "Status", "Path", "Message"}
}

func linkRows(links []project.LinkReport, verbose bool) [][]string {
	rows := make([][]string, 0, len(links))
	for _, link := range links {
		if verbose {
			rows = append(rows, []string{
				link.Source,
				link.Skill,
				link.Status,
				renderVerboseValue(link.Path),
				renderVerboseValue(link.Target),
				renderVerboseValue(link.Message),
			})
			continue
		}

		rows = append(rows, []string{
			link.Source,
			link.Skill,
			link.Status,
			link.Path,
			link.Message,
		})
	}
	return rows
}

func staleLinkRows(links []project.ManagedLink) [][]string {
	rows := make([][]string, 0, len(links))
	for _, link := range links {
		rows = append(rows, []string{link.Path, link.Source, link.Skill})
	}
	return rows
}

func pathRows(paths []string) [][]string {
	rows := make([][]string, 0, len(paths))
	for _, path := range paths {
		rows = append(rows, []string{path})
	}
	return rows
}

func renderVerboseValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
