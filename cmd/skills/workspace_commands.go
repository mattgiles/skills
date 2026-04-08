package main

import (
	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/source"
	"github.com/mattgiles/skills/internal/ui"
)

func newStatusCommand() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show installed skill status for the current repo or shared home scope",
		RunE: func(cmd *cobra.Command, _ []string) error {
			target, err := resolveWorkspaceTarget(cmd.Context(), global, true)
			if err != nil {
				return err
			}
			return runStatusCommand(cmd, target)
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global installs")
	return cmd
}

func newCacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage cached source repos and worktrees",
	}

	cmd.AddCommand(newCacheCleanCommand())
	return cmd
}

func newCacheCleanCommand() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Delete cached repos and worktrees so the next sync re-downloads them",
		RunE: func(cmd *cobra.Command, _ []string) error {
			target, err := resolveWorkspaceTarget(cmd.Context(), global, false)
			if err != nil {
				return err
			}
			return runCacheCleanCommand(cmd, target)
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global cache roots")
	return cmd
}

func newSyncCommand() *cobra.Command {
	var global bool
	var dryRun bool
	var update bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Enforce the declared skills state for the current repo or shared home scope",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			target, err := resolveWorkspaceTarget(cmd.Context(), global, true)
			if err != nil {
				return err
			}
			return runSyncCommand(cmd, target, project.SyncOptions{
				DryRun: dryRun,
				Update: update,
			})
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global installs")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview sync actions without changing state or links")
	cmd.Flags().BoolVar(&update, "update", false, "Advance stored source commits before syncing links")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	var global bool
	var dryRun bool
	var syncAfter bool

	cmd := &cobra.Command{
		Use:   "update [source...]",
		Short: "Resolve newer commits for the current repo or shared home scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			target, err := resolveWorkspaceTarget(cmd.Context(), global, true)
			if err != nil {
				return err
			}
			return runUpdateCommand(cmd, target, project.UpdateOptions{
				SelectedSources: args,
				Sync:            syncAfter,
				DryRun:          dryRun,
			})
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global installs")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview update actions without changing state or links")
	cmd.Flags().BoolVar(&syncAfter, "sync", false, "Run sync after updating source state")
	return cmd
}

func renderWorkspaceSummary(cmd *cobra.Command, summary workspaceSummary, verbose bool) {
	view := ui.New(cmd)
	rows := [][2]string{
		{"Scope", string(summary.Scope)},
	}
	if summary.Root != "" {
		rows = append(rows, [2]string{"Root", summary.Root})
	}
	rows = append(rows, [2]string{"Installs", summary.InstallDir})
	if summary.CacheMode != "" {
		rows = append(rows, [2]string{"Cache", summary.CacheMode})
	}
	rows = append(rows, [2]string{"Worktrees", summary.WorktreeRoot})
	if verbose && summary.RepoRoot != "" {
		rows = append(rows, [2]string{"Repos", summary.RepoRoot})
	}
	_ = view.KeyValues("Workspace", rows)
	view.Blank()
}
