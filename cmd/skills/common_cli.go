package main

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/ui"
)

func newConfigInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the default config file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			view := ui.New(cmd)
			configPath, err := config.DefaultConfigPath()
			if err != nil {
				return err
			}

			if _, err := os.Stat(configPath); err == nil {
				view.Infof("config already exists at %s", configPath)
				return nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}

			cfg := config.DefaultConfig()
			if err := config.Save(configPath, cfg); err != nil {
				return err
			}

			view.Successf("created config at %s", configPath)
			return nil
		},
	}
}

func loadConfig() (config.Config, error) {
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		return config.Config{}, err
	}
	return config.Load(configPath)
}

func verboseEnabled(cmd *cobra.Command) bool {
	value, err := cmd.Flags().GetBool("verbose")
	if err == nil {
		return value
	}

	value, err = cmd.InheritedFlags().GetBool("verbose")
	return err == nil && value
}

func validateSourceAlias(alias string) error {
	return config.ValidateAlias(alias)
}

func newManifestSource(url string, ref string) project.ManifestSource {
	return project.ManifestSource{
		URL: url,
		Ref: ref,
	}
}
