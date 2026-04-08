package main

import (
	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/ui"
)

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "skills",
		Short:         "Manage standardized agent skills from Git sources",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return markUsage(err)
	})

	cmd.PersistentFlags().Bool("verbose", false, "Show detailed diagnostic output")

	cmd.AddCommand(newAddCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newConfigCommand())
	cmd.AddCommand(newCacheCommand())
	cmd.AddCommand(newDoctorCommand())
	cmd.AddCommand(newSelfCommand())
	cmd.AddCommand(newSourceCommand())
	cmd.AddCommand(newSkillCommand())
	cmd.AddCommand(newVersionCommand())

	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		_ = ui.New(c).RenderHelp(c)
	})
	cmd.SetUsageFunc(func(c *cobra.Command) error {
		return ui.New(c).RenderHelp(c)
	})

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
