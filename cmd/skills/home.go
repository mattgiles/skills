package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/source"
)

func newHomeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "home",
		Short: "Manage shared home-level standardized agent skills",
	}

	cmd.AddCommand(newHomeInitCommand())
	cmd.AddCommand(newHomeStatusCommand())
	cmd.AddCommand(newHomeSyncCommand())
	cmd.AddCommand(newHomeUpdateCommand())

	return cmd
}

func newHomeInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the shared home manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			manifestPath, err := project.HomeManifestPath(cfg)
			if err != nil {
				return err
			}
			if _, err := os.Stat(manifestPath); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "manifest already exists: %s\n", manifestPath)
				return nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}

			createdPath, err := project.InitHome(cfg)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created manifest: %s\n", createdPath)
			return nil
		},
	}
}

func newHomeStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show source, canonical skill, and Claude adapter status for shared home skills",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			report, err := project.HomeStatus(context.Background(), cfg)
			if err != nil {
				return err
			}

			renderWorkspaceStatus(cmd, report, verboseEnabled(cmd), "no home sources declared", "no home skills declared")
			return nil
		},
	}
}

func newHomeSyncCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync shared home skills into canonical and Claude adapter directories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			result, err := project.HomeSync(context.Background(), cfg, project.SyncOptions{
				DryRun: dryRun,
			})
			if err != nil {
				return err
			}

			renderWorkspaceSync(cmd, result, verboseEnabled(cmd))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview sync actions without changing state or links")
	return cmd
}

func newHomeUpdateCommand() *cobra.Command {
	var dryRun bool
	var syncAfter bool

	cmd := &cobra.Command{
		Use:   "update [source...]",
		Short: "Resolve newer commits for shared home sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

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

			renderProjectUpdate(cmd, result, verboseEnabled(cmd))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview update actions without changing state or links")
	cmd.Flags().BoolVar(&syncAfter, "sync", false, "Run home sync after updating source state")
	return cmd
}
