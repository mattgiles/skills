package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/discovery"
	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/source"
	"github.com/mattgiles/skills/internal/ui"
)

type addSkillChange struct {
	AddedSource bool
	AddedSkill  bool
	// UpdatedPath is set when an already-declared (source, name) entry had
	// its path: selector changed instead of a new entry being appended.
	UpdatedPath bool
	SourceURL   string
	SourceRef   string
	// SkillPath is the normalized path: selector written for the entry (""
	// when none).
	SkillPath string
}

type addSyncOutcome struct {
	summary workspaceSummary
	result  project.SyncResult
}

func newAddCommand() *cobra.Command {
	var global bool
	var url string
	var ref string
	var skillPath string

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
				discovery.NormalizeSkillPath(skillPath),
			)
			if err != nil {
				return err
			}
			if !change.AddedSkill && !change.UpdatedPath {
				view.Infof("skill %q from source %q is already declared", skillName, sourceAlias)
				return nil
			}
			// Validate the would-be manifest before touching the file so that
			// static problems (absolute/escaping path, duplicate path) fail
			// fast without a write + rollback cycle.
			if err := project.ValidateManifest(nextManifest); err != nil {
				return err
			}

			if change.AddedSource {
				if err := project.UpsertManifestSourceAt(target.ManifestPath, sourceAlias, project.ManifestSource{
					URL: change.SourceURL,
					Ref: change.SourceRef,
				}); err != nil {
					return err
				}
			}
			switch {
			case change.AddedSkill:
				if err := project.AppendManifestSkillAt(target.ManifestPath, project.ManifestSkill{
					Source: sourceAlias,
					Name:   skillName,
					Path:   change.SkillPath,
				}); err != nil {
					return err
				}
			case change.UpdatedPath:
				if err := project.SetManifestSkillPathAt(target.ManifestPath, sourceAlias, skillName, change.SkillPath); err != nil {
					return err
				}
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
			switch {
			case change.UpdatedPath:
				view.Successf("updated path for skill %q from source %q to %s", skillName, sourceAlias, change.SkillPath)
			case change.SkillPath != "":
				view.Successf("added skill %q from source %q (path %s)", skillName, sourceAlias, change.SkillPath)
			default:
				view.Successf("added skill %q from source %q", skillName, sourceAlias)
			}
			view.Blank()
			renderWorkspaceSummary(cmd, outcome.summary, verboseEnabled(cmd))
			renderWorkspaceSync(cmd, outcome.result, verboseEnabled(cmd))
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Operate on shared home/global installs")
	cmd.Flags().StringVar(&url, "url", "", "Source Git URL or local repo path for a new source")
	cmd.Flags().StringVar(&ref, "ref", "", "Source ref for a new source; defaults to the remote's default branch")
	cmd.Flags().StringVar(&skillPath, "path", "", "Repo-relative skill directory to select when multiple skills share a name (name stays the link directory)")
	return cmd
}

// applySkillAdd computes the manifest change for `skills add`. skillPath is
// the already-normalized path: selector ("" for none). For an existing
// (source, name) entry: no path or an equal path is a no-op; a differing path
// updates the selector in place. Otherwise a new entry is appended.
func applySkillAdd(ctx context.Context, manifest project.Manifest, sourceAlias string, skillName string, url string, ref string, skillPath string) (project.Manifest, addSkillChange, error) {
	nextManifest := cloneManifest(manifest)
	change := addSkillChange{SkillPath: skillPath}

	if existing, ok := nextManifest.Sources[sourceAlias]; ok {
		change.SourceURL = existing.URL
		change.SourceRef = existing.Ref
		if idx := manifestSkillIndex(nextManifest, sourceAlias, skillName); idx >= 0 {
			current := discovery.NormalizeSkillPath(nextManifest.Skills[idx].Path)
			if skillPath == "" || current == skillPath {
				change.SkillPath = current
				return nextManifest, change, nil
			}
			nextManifest.Skills[idx].Path = skillPath
			change.UpdatedPath = true
			return nextManifest, change, nil
		}

		nextManifest.Skills = append(nextManifest.Skills, project.ManifestSkill{
			Source: sourceAlias,
			Name:   skillName,
			Path:   skillPath,
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
		Path:   skillPath,
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

// manifestSkillIndex returns the index of the (source, name) entry in
// manifest.Skills, or -1 when absent.
func manifestSkillIndex(manifest project.Manifest, sourceAlias string, skillName string) int {
	for i, skill := range manifest.Skills {
		if skill.Source == sourceAlias && skill.Name == skillName {
			return i
		}
	}
	return -1
}

func restoreManifestBytes(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
