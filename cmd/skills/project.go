package main

import (
	"context"
	"errors"
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
		Short: "Manage project-local skill manifests",
	}

	cmd.AddCommand(newProjectInitCommand())
	cmd.AddCommand(newProjectStatusCommand())
	cmd.AddCommand(newProjectSyncCommand())

	return cmd
}

func newProjectInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a project manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectDir, err := os.Getwd()
			if err != nil {
				return err
			}

			manifestPath := project.ManifestPath(projectDir)
			if _, err := os.Stat(manifestPath); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "manifest already exists: %s\n", manifestPath)
				return nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}

			if err := project.SaveManifest(projectDir, project.DefaultManifest()); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created manifest: %s\n", manifestPath)
			return nil
		},
	}
}

func newProjectStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show manifest, source, and link status for the current project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			projectDir, err := os.Getwd()
			if err != nil {
				return err
			}

			report, err := project.Status(context.Background(), projectDir, cfg)
			if err != nil {
				return err
			}

			renderProjectStatus(cmd, report)
			return nil
		},
	}
}

func newProjectSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync declared project skills into target agent directories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			projectDir, err := os.Getwd()
			if err != nil {
				return err
			}

			result, err := project.Sync(context.Background(), projectDir, cfg)
			if err != nil {
				return err
			}

			renderProjectSync(cmd, result)
			return nil
		},
	}
}

func renderProjectStatus(cmd *cobra.Command, report project.StatusReport) {
	if len(report.Sources) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no project sources declared")
	} else {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SOURCE\tSTATUS\tREF\tCOMMIT\tMESSAGE")
		for _, src := range report.Sources {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", src.Alias, src.Status, src.Ref, src.Commit, src.Message)
		}
		_ = w.Flush()
	}

	if len(report.Links) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no project skills declared")
	} else {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "AGENT\tSOURCE\tSKILL\tSTATUS\tPATH")
		for _, link := range report.Links {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", link.Agent, link.Source, link.Skill, link.Status, link.Path)
		}
		_ = w.Flush()
	}

	if len(report.StaleLinks) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "STALE_PATH\tAGENT\tSOURCE\tSKILL")
		for _, link := range report.StaleLinks {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", link.Path, link.Agent, link.Source, link.Skill)
		}
		_ = w.Flush()
	}
}

func renderProjectSync(cmd *cobra.Command, result project.SyncResult) {
	if len(result.Sources) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SOURCE\tSTATUS\tREF\tCOMMIT")
		for _, src := range result.Sources {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", src.Alias, src.Status, src.Ref, src.Commit)
		}
		_ = w.Flush()
	}

	if len(result.Links) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "AGENT\tSOURCE\tSKILL\tSTATUS\tPATH")
		for _, link := range result.Links {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", link.Agent, link.Source, link.Skill, link.Status, link.Path)
		}
		_ = w.Flush()
	}

	if len(result.Pruned) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PRUNED_PATH")
		for _, path := range result.Pruned {
			fmt.Fprintf(w, "%s\n", path)
		}
		_ = w.Flush()
	}
}
