package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/discovery"
	"github.com/mattgiles/skills/internal/source"
	"github.com/mattgiles/skills/internal/ui"
)

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
			view := ui.New(cmd)
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			alias := args[0]
			url := args[1]

			if err := validateSourceAlias(alias); err != nil {
				return err
			}

			target, err := resolveSourceManifestTarget(cmd.Context(), global)
			if err != nil {
				return err
			}

			sourceRef := strings.TrimSpace(ref)
			if sourceRef == "" {
				if existing, ok := target.Manifest.Sources[alias]; ok && strings.TrimSpace(existing.Ref) != "" {
					sourceRef = existing.Ref
				} else {
					sourceRef, err = source.InferDefaultRef(cmd.Context(), url)
					if err != nil {
						return fmt.Errorf("infer default ref for %s: %w", alias, err)
					}
				}
			}

			target.Manifest.Sources[alias] = newManifestSource(url, sourceRef)
			if err := saveManifestTarget(target); err != nil {
				return err
			}

			view.Successf("registered source %q", alias)
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
			view := ui.New(cmd)
			sources, err := resolveManifestSources(cmd.Context(), global, nil)
			if err != nil {
				return err
			}
			if len(sources) == 0 {
				view.Infof("no sources configured")
				return nil
			}

			rows := make([][]string, 0, len(sources))
			for _, src := range sources {
				status := source.Status(cmd.Context(), src)
				state := renderSourceState(status)
				remote := renderRemoteHead(status)
				local := renderLocalHead(status)
				if verboseEnabled(cmd) {
					rows = append(rows, []string{src.Alias, state, remote, local, src.RepoPath, src.URL})
				} else {
					rows = append(rows, []string{src.Alias, state, remote, local})
				}
			}

			columns := []string{"Alias", "Status", "Remote", "Local"}
			if verboseEnabled(cmd) {
				columns = append(columns, "Path", "URL")
			}
			return view.RenderTable(ui.Table{
				Title:   "Sources",
				Columns: columns,
				Rows:    rows,
			})
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
			view := ui.New(cmd)
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			selected, err := resolveManifestSources(cmd.Context(), global, args)
			if err != nil {
				return err
			}
			if len(selected) == 0 {
				view.Infof("no sources configured")
				return nil
			}

			type syncResult struct {
				action string
				source source.Source
				status source.SourceStatus
			}

			results := make([]syncResult, 0, len(selected))
			for _, src := range selected {
				var cloned bool
				err := view.RunTask(
					fmt.Sprintf("Syncing source %q", src.Alias),
					ui.TaskOptions{
						UseErrorWriter: true,
						SuccessText:    fmt.Sprintf("Synced source %q", src.Alias),
						FailureText:    fmt.Sprintf("Failed to sync source %q", src.Alias),
					},
					func() error {
						var syncErr error
						cloned, syncErr = source.Sync(cmd.Context(), src)
						return syncErr
					},
				)
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
					status: source.Status(cmd.Context(), src),
				})
			}

			rows := make([][]string, 0, len(results))
			columns := []string{"Action", "Alias", "Remote", "Local"}
			if verboseEnabled(cmd) {
				columns = append(columns, "Path", "URL")
			}
			for _, result := range results {
				row := []string{
					result.action,
					result.source.Alias,
					renderRemoteHead(result.status),
					renderLocalHead(result.status),
				}
				if verboseEnabled(cmd) {
					row = append(row, result.source.RepoPath, result.source.URL)
				}
				rows = append(rows, row)
			}

			return view.RenderTable(ui.Table{
				Title:   "Source Sync",
				Columns: columns,
				Rows:    rows,
			})
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
			view := ui.New(cmd)
			selected, err := skillListSources(cmd.Context(), global, sourceAlias)
			if err != nil {
				return err
			}
			if len(selected) == 0 {
				view.Infof("no sources configured")
				return nil
			}

			skills := []discovery.DiscoveredSkill{}
			for _, src := range selected {
				status := source.Status(cmd.Context(), src)
				if !status.Exists || !status.IsGitRepo {
					view.Warningf("skipping unsynced source %q", src.Alias)
					continue
				}

				discovered, err := discovery.Discover(src.Alias, src.RepoPath)
				if err != nil {
					return fmt.Errorf("discover skills in %s: %w", src.Alias, err)
				}
				skills = append(skills, discovered...)
			}

			if len(skills) == 0 {
				view.Infof("no skills found")
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

			rows := make([][]string, 0, len(skills))
			for _, skill := range skills {
				if verboseEnabled(cmd) {
					rows = append(rows, []string{skill.SourceAlias, skill.Name, skill.RelativePath, skill.Path})
				} else {
					rows = append(rows, []string{skill.SourceAlias, skill.Name, skill.RelativePath})
				}
			}

			columns := []string{"Source", "Name", "Path"}
			if verboseEnabled(cmd) {
				columns = append(columns, "Abs Path")
			}
			return view.RenderTable(ui.Table{
				Title:   "Skills",
				Columns: columns,
				Rows:    rows,
			})
		},
	}

	listCmd.Flags().BoolVar(&global, "global", false, "List skills from shared global sources instead of the current repo")
	listCmd.Flags().StringVar(&sourceAlias, "source", "", "Only list skills from the named source")
	cmd.AddCommand(listCmd)

	return cmd
}

func skillListSources(ctx context.Context, global bool, sourceAlias string) ([]source.Source, error) {
	aliases := []string{}
	if sourceAlias != "" {
		aliases = append(aliases, sourceAlias)
	}
	return resolveManifestSources(ctx, global, aliases)
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
