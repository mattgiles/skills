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
	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/source"
)

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "skills",
		Short:         "Manage standardized agent skills from Git sources",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().Bool("verbose", false, "Show detailed diagnostic output")

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
	var global bool
	var ref string

	cmd := &cobra.Command{
		Use:   "add <alias> <git-url>",
		Short: "Register a skill source",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			alias := args[0]
			url := args[1]

			if err := config.ValidateAlias(alias); err != nil {
				return err
			}

			target, err := resolveSourceManifestTarget(global)
			if err != nil {
				return err
			}
			manifest, err := project.LoadManifestAt(target.ManifestPath)
			if err != nil {
				return err
			}

			sourceRef := strings.TrimSpace(ref)
			if sourceRef == "" {
				if existing, ok := manifest.Sources[alias]; ok && strings.TrimSpace(existing.Ref) != "" {
					sourceRef = existing.Ref
				} else {
					sourceRef, err = source.InferDefaultRef(context.Background(), url)
					if err != nil {
						return fmt.Errorf("infer default ref for %s: %w", alias, err)
					}
				}
			}

			manifest.Sources[alias] = project.ManifestSource{
				URL: url,
				Ref: sourceRef,
			}

			if err := project.SaveManifestAt(target.ManifestPath, manifest); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "registered source %q\n", alias)
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global sources")
	cmd.Flags().StringVar(&ref, "ref", "", "Source ref to store in the manifest; defaults to the remote's default branch")
	return cmd
}

func newSourceListCommand() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List declared sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			sources, err := resolveManifestSources(global, nil)
			if err != nil {
				return err
			}
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

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global sources")
	return cmd
}

func newSourceSyncCommand() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "sync [alias...]",
		Short: "Clone or fetch declared sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			selected, err := resolveManifestSources(global, args)
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

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global sources")
	return cmd
}

func newSkillCommand() *cobra.Command {
	var sourceAlias string
	var global bool

	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Inspect discovered skills",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered skills",
		RunE: func(cmd *cobra.Command, _ []string) error {
			selected, err := skillListSources(global, sourceAlias)
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

	listCmd.Flags().BoolVar(&global, "global", false, "List skills from shared global sources instead of the current repo")
	listCmd.Flags().StringVar(&sourceAlias, "source", "", "Only list skills from the named source")
	cmd.AddCommand(listCmd)

	return cmd
}

func skillListSources(global bool, sourceAlias string) ([]source.Source, error) {
	aliases := []string{}
	if sourceAlias != "" {
		aliases = append(aliases, sourceAlias)
	}
	return resolveManifestSources(global, aliases)
}

func loadConfig() (config.Config, error) {
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		return config.Config{}, err
	}
	return config.Load(configPath)
}

type sourceManifestTarget struct {
	ManifestPath string
	Manifest     project.Manifest
	RepoRoot     string
}

func resolveSourceManifestTarget(global bool) (sourceManifestTarget, error) {
	if global {
		cfg, err := loadConfig()
		if err != nil {
			return sourceManifestTarget{}, err
		}
		manifestPath, err := project.HomeManifestPath(cfg)
		if err != nil {
			return sourceManifestTarget{}, err
		}
		manifest, err := project.LoadManifestAt(manifestPath)
		if err != nil {
			return sourceManifestTarget{}, err
		}
		repoRoot, err := config.RepoRootPath(cfg)
		if err != nil {
			return sourceManifestTarget{}, err
		}
		return sourceManifestTarget{
			ManifestPath: manifestPath,
			Manifest:     manifest,
			RepoRoot:     repoRoot,
		}, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return sourceManifestTarget{}, err
	}
	projectRoot, err := resolveRepoRoot(cwd, true)
	if err != nil {
		return sourceManifestTarget{}, err
	}
	manifest, err := project.LoadManifest(projectRoot)
	if err != nil {
		return sourceManifestTarget{}, err
	}
	cacheConfig, err := project.LoadLocalConfig(projectRoot)
	if err != nil {
		return sourceManifestTarget{}, err
	}

	repoRoot := project.RepoRoot(projectRoot)
	if cacheConfig.Mode == project.CacheModeGlobal {
		cfg, err := loadConfig()
		if err != nil {
			return sourceManifestTarget{}, err
		}
		repoRoot, err = config.RepoRootPath(cfg)
		if err != nil {
			return sourceManifestTarget{}, err
		}
	}

	return sourceManifestTarget{
		ManifestPath: project.ManifestPath(projectRoot),
		Manifest:     manifest,
		RepoRoot:     repoRoot,
	}, nil
}

func resolveManifestSources(global bool, aliases []string) ([]source.Source, error) {
	target, err := resolveSourceManifestTarget(global)
	if err != nil {
		return nil, err
	}
	return selectManifestSources(target.Manifest, target.RepoRoot, aliases, global)
}

func selectManifestSources(manifest project.Manifest, repoRoot string, aliases []string, global bool) ([]source.Source, error) {
	if len(aliases) == 0 {
		aliases = make([]string, 0, len(manifest.Sources))
		for alias := range manifest.Sources {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
	}

	selected := make([]source.Source, 0, len(aliases))
	seen := map[string]struct{}{}
	scopeLabel := "repo"
	if global {
		scopeLabel = "home"
	}

	for _, alias := range aliases {
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}

		entry, ok := manifest.Sources[alias]
		if !ok {
			return nil, fmt.Errorf("unknown source %q", alias)
		}
		if strings.TrimSpace(entry.URL) == "" {
			return nil, fmt.Errorf("source %q has no URL in %s manifest", alias, scopeLabel)
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
