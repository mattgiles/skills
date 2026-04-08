package main

import "github.com/spf13/cobra"

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "skills",
		Short:         "Manage standardized agent skills from Git sources",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().Bool("verbose", false, "Show detailed diagnostic output")

	cmd.AddCommand(newAddCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newConfigCommand())
	cmd.AddCommand(newDoctorCommand())
	cmd.AddCommand(newSelfCommand())
	cmd.AddCommand(newSourceCommand())
	cmd.AddCommand(newSkillCommand())
	cmd.AddCommand(newVersionCommand())

	return cmd
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage global configuration",
	}
	cmd.AddCommand(newConfigInitCommand())
	return cmd
}
