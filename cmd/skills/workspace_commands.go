package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/source"
)

func newStatusCommand() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show installed skill status for the current repo or shared home scope",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if global {
				cfg, err := loadConfig()
				if err != nil {
					return err
				}

				report, err := project.HomeStatus(context.Background(), cfg)
				if err != nil {
					return err
				}

				summary, err := globalWorkspaceSummary(cfg)
				if err != nil {
					return err
				}

				renderWorkspaceSummary(cmd, summary, verboseEnabled(cmd))
				renderWorkspaceStatus(cmd, report, verboseEnabled(cmd), "no home sources declared", "no home skills declared")
				return nil
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			projectRoot, err := resolveRepoRoot(cwd, true)
			if err != nil {
				return err
			}

			report, err := project.Status(context.Background(), projectRoot)
			if err != nil {
				return err
			}

			summary, err := repoWorkspaceSummary(projectRoot)
			if err != nil {
				return err
			}

			renderWorkspaceSummary(cmd, summary, verboseEnabled(cmd))
			renderWorkspaceStatus(cmd, report, verboseEnabled(cmd), "no repo sources declared", "no repo skills declared")
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global installs")
	return cmd
}

func newSyncCommand() *cobra.Command {
	var global bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Enforce the declared skills state for the current repo or shared home scope",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			if global {
				cfg, err := loadConfig()
				if err != nil {
					return err
				}

				result, err := project.HomeSync(context.Background(), cfg, project.SyncOptions{DryRun: dryRun})
				if err != nil {
					return err
				}

				summary, err := globalWorkspaceSummary(cfg)
				if err != nil {
					return err
				}

				renderWorkspaceSummary(cmd, summary, verboseEnabled(cmd))
				renderWorkspaceSync(cmd, result, verboseEnabled(cmd))
				return nil
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			projectRoot, err := resolveRepoRoot(cwd, true)
			if err != nil {
				return err
			}

			result, err := project.Sync(context.Background(), projectRoot, project.SyncOptions{DryRun: dryRun})
			if err != nil {
				return err
			}

			summary, err := repoWorkspaceSummary(projectRoot)
			if err != nil {
				return err
			}

			renderWorkspaceSummary(cmd, summary, verboseEnabled(cmd))
			renderWorkspaceSync(cmd, result, verboseEnabled(cmd))
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global installs")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview sync actions without changing state or links")
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

			if global {
				cfg, err := loadConfig()
				if err != nil {
					return err
				}

				result, err := project.HomeUpdate(context.Background(), cfg, project.UpdateOptions{
					SelectedSources: args,
					Sync:            syncAfter,
					DryRun:          dryRun,
				})
				if err != nil {
					return err
				}

				summary, err := globalWorkspaceSummary(cfg)
				if err != nil {
					return err
				}

				renderWorkspaceSummary(cmd, summary, verboseEnabled(cmd))
				renderProjectUpdate(cmd, result, verboseEnabled(cmd))
				return nil
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			projectRoot, err := resolveRepoRoot(cwd, true)
			if err != nil {
				return err
			}

			result, err := project.Update(context.Background(), projectRoot, project.UpdateOptions{
				SelectedSources: args,
				Sync:            syncAfter,
				DryRun:          dryRun,
			})
			if err != nil {
				return err
			}

			summary, err := repoWorkspaceSummary(projectRoot)
			if err != nil {
				return err
			}

			renderWorkspaceSummary(cmd, summary, verboseEnabled(cmd))
			renderProjectUpdate(cmd, result, verboseEnabled(cmd))
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global installs")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview update actions without changing state or links")
	cmd.Flags().BoolVar(&syncAfter, "sync", false, "Run sync after updating source state")
	return cmd
}

func renderWorkspaceSummary(cmd *cobra.Command, summary workspaceSummary, verbose bool) {
	fmt.Fprintf(cmd.OutOrStdout(), "scope: %s\n", summary.Scope)
	if summary.Root != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "root: %s\n", summary.Root)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "installs: %s\n", summary.InstallDir)
	if summary.CacheMode != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "cache: %s\n", summary.CacheMode)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "worktrees: %s\n", summary.WorktreeRoot)
	if verbose && summary.RepoRoot != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "repos: %s\n", summary.RepoRoot)
	}
	fmt.Fprintln(cmd.OutOrStdout())
}
