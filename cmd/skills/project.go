package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/source"
)

func newProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage project-local standardized agent skills",
	}

	cmd.AddCommand(newProjectInitCommand())
	cmd.AddCommand(newProjectStatusCommand())
	cmd.AddCommand(newProjectSyncCommand())
	cmd.AddCommand(newProjectUpdateCommand())

	return cmd
}

func newProjectInitCommand() *cobra.Command {
	var cacheMode string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a project standardized workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectDir, err := os.Getwd()
			if err != nil {
				return err
			}

			return runProjectInit(cmd, projectDir, cacheMode)
		},
	}

	cmd.Flags().StringVar(&cacheMode, "cache", "", "Project cache backend: local or global")
	return cmd
}

func newProjectStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show source, canonical skill, and Claude adapter status for the current project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectDir, err := os.Getwd()
			if err != nil {
				return err
			}

			report, err := project.Status(context.Background(), projectDir)
			if err != nil {
				return err
			}

			renderProjectStatus(cmd, report, verboseEnabled(cmd))
			return nil
		},
	}
}

func newProjectSyncCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync declared project skills into canonical and Claude adapter directories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			projectDir, err := os.Getwd()
			if err != nil {
				return err
			}

			result, err := project.Sync(context.Background(), projectDir, project.SyncOptions{
				DryRun: dryRun,
			})
			if err != nil {
				return err
			}

			renderProjectSync(cmd, result, verboseEnabled(cmd))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview sync actions without changing state or links")
	return cmd
}

func newProjectUpdateCommand() *cobra.Command {
	var dryRun bool
	var syncAfter bool

	cmd := &cobra.Command{
		Use:   "update [source...]",
		Short: "Resolve newer commits for project sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			projectDir, err := os.Getwd()
			if err != nil {
				return err
			}

			result, err := project.Update(context.Background(), projectDir, project.UpdateOptions{
				SelectedSources: args,
				Sync:            syncAfter,
				DryRun:          dryRun,
			})
			if err != nil {
				return err
			}

			renderProjectUpdate(cmd, result, verboseEnabled(cmd))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview update actions without changing state or links")
	cmd.Flags().BoolVar(&syncAfter, "sync", false, "Run project sync after updating source state")
	return cmd
}

func renderProjectStatus(cmd *cobra.Command, report project.StatusReport, verbose bool) {
	renderWorkspaceStatus(cmd, report, verbose, "no project sources declared", "no project skills declared")
}

func renderWorkspaceStatus(cmd *cobra.Command, report project.StatusReport, verbose bool, noSources string, noSkills string) {
	if len(report.Sources) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), noSources)
	} else {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "SOURCES")
		if verbose {
			fmt.Fprintln(w, "SOURCE\tSTATUS\tREF\tCOMMIT\tSTORED\tREPO_PATH\tWORKTREE_PATH\tMESSAGE")
		} else {
			fmt.Fprintln(w, "SOURCE\tSTATUS\tREF\tCOMMIT\tMESSAGE")
		}
		for _, src := range report.Sources {
			if verbose {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					src.Alias, src.Status, src.Ref, renderVerboseValue(src.Commit), renderVerboseValue(src.PreviousCommit),
					renderVerboseValue(src.RepoPath), renderVerboseValue(src.WorktreePath), renderVerboseValue(src.Message))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", src.Alias, src.Status, src.Ref, src.Commit, src.Message)
			}
		}
		_ = w.Flush()
	}

	if len(report.SkillLinks) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), noSkills)
	} else {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "\nSKILLS")
		if verbose {
			fmt.Fprintln(w, "SOURCE\tSKILL\tSTATUS\tPATH\tTARGET\tMESSAGE")
		} else {
			fmt.Fprintln(w, "SOURCE\tSKILL\tSTATUS\tPATH\tMESSAGE")
		}
		for _, link := range report.SkillLinks {
			if verbose {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					link.Source, link.Skill, link.Status, renderVerboseValue(link.Path),
					renderVerboseValue(link.Target), renderVerboseValue(link.Message))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", link.Source, link.Skill, link.Status, link.Path, link.Message)
			}
		}
		_ = w.Flush()
	}

	if len(report.ClaudeLinks) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "\nCLAUDE")
		if verbose {
			fmt.Fprintln(w, "SOURCE\tSKILL\tSTATUS\tPATH\tTARGET\tMESSAGE")
		} else {
			fmt.Fprintln(w, "SOURCE\tSKILL\tSTATUS\tPATH\tMESSAGE")
		}
		for _, link := range report.ClaudeLinks {
			if verbose {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					link.Source, link.Skill, link.Status, renderVerboseValue(link.Path),
					renderVerboseValue(link.Target), renderVerboseValue(link.Message))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", link.Source, link.Skill, link.Status, link.Path, link.Message)
			}
		}
		_ = w.Flush()
	}

	if len(report.StaleSkillLinks) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "\nSTALE_SKILLS")
		fmt.Fprintln(w, "STALE_PATH\tSOURCE\tSKILL")
		for _, link := range report.StaleSkillLinks {
			fmt.Fprintf(w, "%s\t%s\t%s\n", link.Path, link.Source, link.Skill)
		}
		_ = w.Flush()
	}

	if len(report.StaleClaudeLinks) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "\nSTALE_CLAUDE")
		fmt.Fprintln(w, "STALE_PATH\tSOURCE\tSKILL")
		for _, link := range report.StaleClaudeLinks {
			fmt.Fprintf(w, "%s\t%s\t%s\n", link.Path, link.Source, link.Skill)
		}
		_ = w.Flush()
	}
}

func renderProjectSync(cmd *cobra.Command, result project.SyncResult, verbose bool) {
	renderWorkspaceSync(cmd, result, verbose)
}

func renderWorkspaceSync(cmd *cobra.Command, result project.SyncResult, verbose bool) {
	if result.DryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "dry-run")
	}

	if len(result.Sources) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "SOURCES")
		if verbose {
			fmt.Fprintln(w, "SOURCE\tSTATUS\tREF\tCOMMIT\tSTORED\tREPO_PATH\tWORKTREE_PATH\tMESSAGE")
		} else {
			fmt.Fprintln(w, "SOURCE\tSTATUS\tREF\tCOMMIT\tMESSAGE")
		}
		for _, src := range result.Sources {
			if verbose {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					src.Alias, src.Status, src.Ref, renderVerboseValue(src.Commit), renderVerboseValue(src.PreviousCommit),
					renderVerboseValue(src.RepoPath), renderVerboseValue(src.WorktreePath), renderVerboseValue(src.Message))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", src.Alias, src.Status, src.Ref, src.Commit, src.Message)
			}
		}
		_ = w.Flush()
	}

	if len(result.SkillLinks) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "\nSKILLS")
		if verbose {
			fmt.Fprintln(w, "SOURCE\tSKILL\tSTATUS\tPATH\tTARGET\tMESSAGE")
		} else {
			fmt.Fprintln(w, "SOURCE\tSKILL\tSTATUS\tPATH\tMESSAGE")
		}
		for _, link := range result.SkillLinks {
			if verbose {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					link.Source, link.Skill, link.Status, renderVerboseValue(link.Path),
					renderVerboseValue(link.Target), renderVerboseValue(link.Message))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", link.Source, link.Skill, link.Status, link.Path, link.Message)
			}
		}
		_ = w.Flush()
	}

	if len(result.ClaudeLinks) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "\nCLAUDE")
		if verbose {
			fmt.Fprintln(w, "SOURCE\tSKILL\tSTATUS\tPATH\tTARGET\tMESSAGE")
		} else {
			fmt.Fprintln(w, "SOURCE\tSKILL\tSTATUS\tPATH\tMESSAGE")
		}
		for _, link := range result.ClaudeLinks {
			if verbose {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					link.Source, link.Skill, link.Status, renderVerboseValue(link.Path),
					renderVerboseValue(link.Target), renderVerboseValue(link.Message))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", link.Source, link.Skill, link.Status, link.Path, link.Message)
			}
		}
		_ = w.Flush()
	}

	if len(result.PrunedSkillLinks) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "\nPRUNED_SKILLS")
		fmt.Fprintln(w, "PRUNED_PATH")
		for _, path := range result.PrunedSkillLinks {
			fmt.Fprintf(w, "%s\n", path)
		}
		_ = w.Flush()
	}

	if len(result.PrunedClaudeLinks) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "\nPRUNED_CLAUDE")
		fmt.Fprintln(w, "PRUNED_PATH")
		for _, path := range result.PrunedClaudeLinks {
			fmt.Fprintf(w, "%s\n", path)
		}
		_ = w.Flush()
	}
}

func renderProjectUpdate(cmd *cobra.Command, result project.UpdateResult, verbose bool) {
	if result.DryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "dry-run")
	}

	if len(result.Sources) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(cmd.OutOrStdout(), "SOURCES")
		if verbose {
			fmt.Fprintln(w, "SOURCE\tSTATUS\tREF\tCOMMIT\tSTORED\tREPO_PATH\tWORKTREE_PATH\tMESSAGE")
		} else {
			fmt.Fprintln(w, "SOURCE\tSTATUS\tREF\tCOMMIT\tMESSAGE")
		}
		for _, src := range result.Sources {
			if verbose {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					src.Alias, src.Status, src.Ref, renderVerboseValue(src.Commit), renderVerboseValue(src.PreviousCommit),
					renderVerboseValue(src.RepoPath), renderVerboseValue(src.WorktreePath), renderVerboseValue(src.Message))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", src.Alias, src.Status, src.Ref, src.Commit, src.Message)
			}
		}
		_ = w.Flush()
	}

	if result.Sync != nil {
		fmt.Fprintln(cmd.OutOrStdout())
		renderProjectSync(cmd, *result.Sync, verbose)
	}
}

func renderVerboseValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
