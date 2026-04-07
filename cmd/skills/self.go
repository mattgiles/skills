package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/selfupdate"
)

var runSelfUpdate = selfupdate.Run

func newSelfCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "self",
		Short: "Manage the installed skills CLI",
	}
	cmd.AddCommand(newSelfUpdateCommand())
	return cmd
}

func newSelfUpdateCommand() *cobra.Command {
	var targetVersion string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the installed skills binary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			executablePath, err := os.Executable()
			if err != nil {
				return err
			}

			result, err := runSelfUpdate(selfupdate.Options{
				CurrentVersion: version,
				TargetVersion:  targetVersion,
				TargetPath:     executablePath,
			})
			if err != nil {
				return err
			}

			if !result.Updated {
				fmt.Fprintf(cmd.OutOrStdout(), "skills is already at %s\n", result.Version)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "updated skills from %s to %s\n", renderVersionValue(result.PreviousVersion), result.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "binary: %s\n", result.TargetPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&targetVersion, "version", "", "Install a specific release version")
	return cmd
}

func renderVersionValue(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
