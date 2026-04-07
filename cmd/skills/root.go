package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/discovery"
	"github.com/mattgiles/skills/internal/source"
)

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "skills",
		Short:         "Manage local agent skill sources",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().Bool("verbose", false, "Show detailed diagnostic output")

	cmd.AddCommand(newConfigCommand())
	cmd.AddCommand(newSourceCommand())
	cmd.AddCommand(newSkillCommand())
	cmd.AddCommand(newProjectCommand())

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

func newConfigInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the default config file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := config.DefaultConfigPath()
			if err != nil {
				return err
			}

			if _, err := os.Stat(configPath); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "config already exists: %s\n", configPath)
				return nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}

			cfg := config.DefaultConfig()
			if err := config.Save(configPath, cfg); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "created config: %s\n", configPath)
			return nil
		},
	}
}

func newSourceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage skill sources",
	}

	cmd.AddCommand(newSourceAddCommand())
	cmd.AddCommand(newSourceListCommand())
	cmd.AddCommand(newSourceSyncCommand())

	return cmd
}

func newSourceAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <alias> <git-url>",
		Short: "Register a skill source",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			url := args[1]

			if err := config.ValidateAlias(alias); err != nil {
				return err
			}

			configPath, err := config.DefaultConfigPath()
			if err != nil {
				return err
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			cfg.Sources[alias] = config.SourceConfig{URL: url}

			if err := config.Save(configPath, cfg); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "registered source %q\n", alias)
			return nil
		},
	}
}

func newSourceListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, repoRoot, err := loadConfigAndRepoRoot()
			if err != nil {
				return err
			}

			sources := configuredSources(cfg, repoRoot)
			if len(sources) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no sources configured")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			if verboseEnabled(cmd) {
				fmt.Fprintln(w, "ALIAS\tSTATUS\tREMOTE\tLOCAL\tPATH\tURL")
			} else {
				fmt.Fprintln(w, "ALIAS\tSTATUS\tREMOTE\tLOCAL")
			}

			for _, src := range sources {
				status := source.Status(context.Background(), src)
				state := renderSourceState(status)
				remote := renderRemoteHead(status)
				local := renderLocalHead(status)
				if verboseEnabled(cmd) {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", src.Alias, state, remote, local, src.RepoPath, src.URL)
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", src.Alias, state, remote, local)
				}
			}

			return w.Flush()
		},
	}
}

func newSourceSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [alias...]",
		Short: "Clone or fetch configured sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			cfg, repoRoot, err := loadConfigAndRepoRoot()
			if err != nil {
				return err
			}

			selected, err := selectSources(cfg, repoRoot, args)
			if err != nil {
				return err
			}
			if len(selected) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no sources configured")
				return nil
			}

			if verboseEnabled(cmd) {
				type syncResult struct {
					action string
					source source.Source
					status source.SourceStatus
				}

				results := make([]syncResult, 0, len(selected))
				for _, src := range selected {
					cloned, err := source.Sync(context.Background(), src)
					if err != nil {
						return fmt.Errorf("sync %s: %w", src.Alias, err)
					}

					action := "fetched"
					if cloned {
						action = "cloned"
					}
					results = append(results, syncResult{
						action: action,
						source: src,
						status: source.Status(context.Background(), src),
					})
				}

				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "ACTION\tALIAS\tREMOTE\tLOCAL\tPATH\tURL")
				for _, result := range results {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
						result.action,
						result.source.Alias,
						renderRemoteHead(result.status),
						renderLocalHead(result.status),
						result.source.RepoPath,
						result.source.URL,
					)
				}
				return w.Flush()
			}

			for _, src := range selected {
				cloned, err := source.Sync(context.Background(), src)
				if err != nil {
					return fmt.Errorf("sync %s: %w", src.Alias, err)
				}

				action := "fetched"
				if cloned {
					action = "cloned"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", action, src.Alias)
			}

			return nil
		},
	}
}

func newSkillCommand() *cobra.Command {
	var sourceAlias string

	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Inspect discovered skills",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered skills",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, repoRoot, err := loadConfigAndRepoRoot()
			if err != nil {
				return err
			}

			aliases := []string{}
			if sourceAlias != "" {
				aliases = append(aliases, sourceAlias)
			}

			selected, err := selectSources(cfg, repoRoot, aliases)
			if err != nil {
				return err
			}
			if len(selected) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no sources configured")
				return nil
			}

			skills := make([]discovery.DiscoveredSkill, 0)
			for _, src := range selected {
				status := source.Status(context.Background(), src)
				if !status.Exists || !status.IsGitRepo {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping unsynced source %q\n", src.Alias)
					continue
				}

				discovered, err := discovery.Discover(src.Alias, src.RepoPath)
				if err != nil {
					return fmt.Errorf("discover skills in %s: %w", src.Alias, err)
				}
				skills = append(skills, discovered...)
			}

			if len(skills) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no skills found")
				return nil
			}

			sort.Slice(skills, func(i, j int) bool {
				if skills[i].SourceAlias != skills[j].SourceAlias {
					return skills[i].SourceAlias < skills[j].SourceAlias
				}
				if skills[i].Name != skills[j].Name {
					return skills[i].Name < skills[j].Name
				}
				return skills[i].RelativePath < skills[j].RelativePath
			})

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			if verboseEnabled(cmd) {
				fmt.Fprintln(w, "SOURCE\tNAME\tPATH\tABS_PATH")
			} else {
				fmt.Fprintln(w, "SOURCE\tNAME\tPATH")
			}
			for _, skill := range skills {
				if verboseEnabled(cmd) {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", skill.SourceAlias, skill.Name, skill.RelativePath, skill.Path)
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\n", skill.SourceAlias, skill.Name, skill.RelativePath)
				}
			}
			return w.Flush()
		},
	}

	listCmd.Flags().StringVar(&sourceAlias, "source", "", "Only list skills from the named source")
	cmd.AddCommand(listCmd)

	return cmd
}

func loadConfigAndRepoRoot() (config.Config, string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return config.Config{}, "", err
	}

	repoRoot, err := config.RepoRootPath(cfg)
	if err != nil {
		return config.Config{}, "", err
	}

	return cfg, repoRoot, nil
}

func loadConfig() (config.Config, error) {
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		return config.Config{}, err
	}
	return config.Load(configPath)
}

func configuredSources(cfg config.Config, repoRoot string) []source.Source {
	aliases := make([]string, 0, len(cfg.Sources))
	for alias := range cfg.Sources {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	sources := make([]source.Source, 0, len(aliases))
	for _, alias := range aliases {
		sources = append(sources, source.Source{
			Alias:    alias,
			URL:      cfg.Sources[alias].URL,
			RepoPath: source.RepoPath(repoRoot, alias),
		})
	}

	return sources
}

func selectSources(cfg config.Config, repoRoot string, aliases []string) ([]source.Source, error) {
	if len(aliases) == 0 {
		return configuredSources(cfg, repoRoot), nil
	}

	selected := make([]source.Source, 0, len(aliases))
	seen := map[string]struct{}{}

	for _, alias := range aliases {
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}

		entry, ok := cfg.Sources[alias]
		if !ok {
			return nil, fmt.Errorf("unknown source %q", alias)
		}

		selected = append(selected, source.Source{
			Alias:    alias,
			URL:      entry.URL,
			RepoPath: source.RepoPath(repoRoot, alias),
		})
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Alias < selected[j].Alias
	})
	return selected, nil
}

func renderSourceState(status source.SourceStatus) string {
	switch {
	case !status.Exists:
		return "missing"
	case !status.IsGitRepo:
		if status.LastError == "" {
			return "invalid"
		}
		return "invalid: " + status.LastError
	case status.DefaultRemoteCommit != "":
		return "synced"
	default:
		return "cloned"
	}
}

func renderRemoteHead(status source.SourceStatus) string {
	if status.DefaultRemoteCommit == "" {
		return "-"
	}

	commit := status.DefaultRemoteCommit
	if len(commit) > 12 {
		commit = commit[:12]
	}

	if strings.TrimSpace(status.DefaultRemoteRef) == "" {
		return commit
	}

	return status.DefaultRemoteRef + "@" + commit
}

func renderLocalHead(status source.SourceStatus) string {
	if status.HeadCommit == "" {
		return "-"
	}

	commit := status.HeadCommit
	if len(commit) > 12 {
		commit = commit[:12]
	}

	if strings.TrimSpace(status.HeadRef) == "" {
		return commit
	}

	return status.HeadRef + "@" + commit
}

func verboseEnabled(cmd *cobra.Command) bool {
	value, err := cmd.Flags().GetBool("verbose")
	if err == nil {
		return value
	}
	value, err = cmd.InheritedFlags().GetBool("verbose")
	return err == nil && value
}
