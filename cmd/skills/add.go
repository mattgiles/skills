package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/source"
	"github.com/mattgiles/skills/internal/ui"
)

type addSkillChange struct {
	AddedSource bool
	AddedSkill  bool
	SourceURL   string
	SourceRef   string
}

type addSyncOutcome struct {
	summary workspaceSummary
	result  project.SyncResult
}

func newAddCommand() *cobra.Command {
	var global bool
	var url string
	var ref string

	cmd := &cobra.Command{
		Use:   "add <source> <skill>",
		Short: "Add a skill to the active manifest and sync it",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			view := ui.New(cmd)
			if err := source.EnsureGitAvailable(); err != nil {
				return err
			}

			sourceAlias := strings.TrimSpace(args[0])
			skillName := strings.TrimSpace(args[1])

			if err := config.ValidateAlias(sourceAlias); err != nil {
				return err
			}
			if skillName == "" {
				return fmt.Errorf("skill name must not be empty")
			}

			target, err := resolveSourceManifestTarget(cmd.Context(), global)
			if err != nil {
				return err
			}

			originalBytes, err := os.ReadFile(target.ManifestPath)
			if err != nil {
				return err
			}

			nextManifest, change, err := applySkillAdd(
				cmd.Context(),
				target.Manifest,
				sourceAlias,
				skillName,
				strings.TrimSpace(url),
				strings.TrimSpace(ref),
			)
			if err != nil {
				return err
			}
			if !change.AddedSkill {
				view.Infof("skill %q from source %q is already declared", skillName, sourceAlias)
				return nil
			}

			if err := project.SaveManifestAt(target.ManifestPath, nextManifest); err != nil {
				return err
			}

			outcome, err := runAddSync(cmd, target, sourceAlias)
			if err != nil {
				if restoreErr := restoreManifestBytes(target.ManifestPath, originalBytes); restoreErr != nil {
					return fmt.Errorf("%w; rollback manifest %s: %v", err, target.ManifestPath, restoreErr)
				}
				return err
			}

			if change.AddedSource {
				view.Successf("added source %q (%s @ %s)", sourceAlias, change.SourceURL, change.SourceRef)
			}
			view.Successf("added skill %q from source %q", skillName, sourceAlias)
			view.Blank()
			renderWorkspaceSummary(cmd, outcome.summary, verboseEnabled(cmd))
			renderWorkspaceSync(cmd, outcome.result, verboseEnabled(cmd))
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global installs")
	cmd.Flags().StringVar(&url, "url", "", "Source Git URL or local repo path for a new source")
	cmd.Flags().StringVar(&ref, "ref", "", "Source ref for a new source; defaults to the remote's default branch")
	return cmd
}

func applySkillAdd(ctx context.Context, manifest project.Manifest, sourceAlias string, skillName string, url string, ref string) (project.Manifest, addSkillChange, error) {
	nextManifest := cloneManifest(manifest)
	change := addSkillChange{}

	if existing, ok := nextManifest.Sources[sourceAlias]; ok {
		change.SourceURL = existing.URL
		change.SourceRef = existing.Ref
		if manifestHasSkill(nextManifest, sourceAlias, skillName) {
			return nextManifest, change, nil
		}

		nextManifest.Skills = append(nextManifest.Skills, project.ManifestSkill{
			Source: sourceAlias,
			Name:   skillName,
		})
		change.AddedSkill = true
		return nextManifest, change, nil
	}

	if url == "" {
		return project.Manifest{}, addSkillChange{}, fmt.Errorf("source %q is not declared; --url is required to add a new source", sourceAlias)
	}

	sourceRef := ref
	if sourceRef == "" {
		inferredRef, err := source.InferDefaultRef(ctx, url)
		if err != nil {
			return project.Manifest{}, addSkillChange{}, fmt.Errorf("infer default ref for %s: %w", sourceAlias, err)
		}
		sourceRef = inferredRef
	}

	nextManifest.Sources[sourceAlias] = project.ManifestSource{
		URL: url,
		Ref: sourceRef,
	}
	nextManifest.Skills = append(nextManifest.Skills, project.ManifestSkill{
		Source: sourceAlias,
		Name:   skillName,
	})

	change.AddedSource = true
	change.AddedSkill = true
	change.SourceURL = url
	change.SourceRef = sourceRef
	return nextManifest, change, nil
}

func cloneManifest(manifest project.Manifest) project.Manifest {
	nextManifest := project.Manifest{
		Sources: make(map[string]project.ManifestSource, len(manifest.Sources)),
		Skills:  append([]project.ManifestSkill(nil), manifest.Skills...),
	}
	for alias, src := range manifest.Sources {
		nextManifest.Sources[alias] = src
	}
	return nextManifest
}

func manifestHasSkill(manifest project.Manifest, sourceAlias string, skillName string) bool {
	for _, skill := range manifest.Skills {
		if skill.Source == sourceAlias && skill.Name == skillName {
			return true
		}
	}
	return false
}

func restoreManifestBytes(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
