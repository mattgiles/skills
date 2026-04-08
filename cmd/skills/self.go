package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/selfupdate"
	"github.com/mattgiles/skills/internal/ui"
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
			view := ui.New(cmd)
			executablePath, err := os.Executable()
			if err != nil {
				return err
			}

			var result selfupdate.Result
			err = view.RunTask("Updating skills CLI", ui.TaskOptions{
				UseErrorWriter: true,
				SuccessText:    "Updated skills CLI",
				FailureText:    "Failed to update skills CLI",
			}, func() error {
				var updateErr error
				result, updateErr = runSelfUpdate(selfupdate.Options{
					CurrentVersion: version,
					TargetVersion:  targetVersion,
					TargetPath:     executablePath,
				})
				return updateErr
			})
			if err != nil {
				return err
			}

			if !result.Updated {
				view.Infof("skills is already at %s", result.Version)
				return nil
			}

			view.Successf(
				"updated skills from %s to %s",
				renderVersionValue(result.PreviousVersion),
				result.Version,
			)
			view.Infof("binary: %s", result.TargetPath)
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
