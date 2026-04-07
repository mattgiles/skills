package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/gitrepo"
	"github.com/mattgiles/skills/internal/project"
)

func newInitCommand() *cobra.Command {
	var projectScope bool
	var globalScope bool
	var cacheMode string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize repo-local or shared home skills state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if projectScope && globalScope {
				return errors.New("choose only one of --project or --global")
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			if projectScope {
				return runProjectInit(cmd, cwd, cacheMode)
			}
			if globalScope {
				return runHomeInit(cmd)
			}

			info, err := gitrepo.Discover(context.Background(), cwd)
			if err != nil {
				return err
			}
			if info.Root == "" {
				return errors.New("outside a Git repo; use skills init --project or skills init --global")
			}

			projectRoot := info.Root
			artifacts, err := project.InspectProjectArtifacts(projectRoot)
			if err != nil {
				return err
			}
			if artifacts.HasArtifacts {
				return runProjectInit(cmd, projectRoot, cacheMode)
			}

			if !isInteractive(cmd.InOrStdin(), cmd.OutOrStdout()) {
				return errors.New("inside a Git repo but no skills artifacts exist yet; use skills init --project or skills init --global")
			}

			scope, err := promptInitScope(cmd)
			if err != nil {
				return err
			}
			if scope == "global" {
				return runHomeInit(cmd)
			}
			return runProjectInit(cmd, projectRoot, cacheMode)
		},
	}

	cmd.Flags().BoolVar(&projectScope, "project", false, "Initialize repo-local project state")
	cmd.Flags().BoolVar(&globalScope, "global", false, "Initialize shared home/global state")
	cmd.Flags().StringVar(&cacheMode, "cache", "", "Project cache backend: local or global")
	return cmd
}

func runProjectInit(cmd *cobra.Command, projectDir string, requestedCacheMode string) error {
	cacheMode, err := resolveProjectInitCacheMode(cmd, projectDir, requestedCacheMode)
	if err != nil {
		return err
	}
	result, err := project.InitProject(projectDir, project.InitProjectOptions{CacheMode: cacheMode})
	if err != nil {
		return err
	}

	if result.ManifestCreated {
		fmt.Fprintf(cmd.OutOrStdout(), "created manifest: %s\n", result.ManifestPath)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "manifest already exists: %s\n", result.ManifestPath)
	}
	if result.LocalConfigSaved {
		fmt.Fprintf(cmd.OutOrStdout(), "saved local config: %s\n", result.LocalConfigPath)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "local config already set: %s\n", result.LocalConfigPath)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "cache mode: %s\n", result.CacheMode)
	if result.GitignoreUpdated {
		fmt.Fprintf(cmd.OutOrStdout(), "updated gitignore: %s\n", result.GitignorePath)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "gitignore already covers managed runtime artifacts: %s\n", result.GitignorePath)
	}
	return nil
}

func runHomeInit(cmd *cobra.Command) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	manifestPath, err := project.HomeManifestPath(cfg)
	if err != nil {
		return err
	}
	if _, err := os.Stat(manifestPath); err == nil {
		fmt.Fprintf(cmd.OutOrStdout(), "manifest already exists: %s\n", manifestPath)
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	createdPath, err := project.InitHome(cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "created manifest: %s\n", createdPath)
	return nil
}

func promptInitScope(cmd *cobra.Command) (string, error) {
	fmt.Fprintln(cmd.OutOrStdout(), "Choose skills initialization mode:")
	fmt.Fprintln(cmd.OutOrStdout(), "1. repo-local")
	fmt.Fprintln(cmd.OutOrStdout(), "2. global")
	fmt.Fprint(cmd.OutOrStdout(), "> ")

	reader := bufio.NewReader(cmd.InOrStdin())
	choice, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	switch strings.TrimSpace(strings.ToLower(choice)) {
	case "1", "repo-local", "repo", "project":
		return "project", nil
	case "2", "global", "home":
		return "global", nil
	default:
		return "", errors.New("invalid init choice; use skills init --project or skills init --global")
	}
}

func resolveProjectInitCacheMode(cmd *cobra.Command, projectDir string, requested string) (project.CacheMode, error) {
	if requested != "" {
		return normalizeCacheMode(requested)
	}

	current, err := project.LoadLocalConfig(projectDir)
	if err != nil {
		return "", err
	}

	if !isInteractive(cmd.InOrStdin(), cmd.OutOrStdout()) {
		if current.Exists || current.Implicit {
			if current.Exists {
				return current.Mode, nil
			}
			return "", errors.New("project cache mode is not configured yet; use --cache=local or --cache=global")
		}
		return "", errors.New("project cache mode is not configured yet; use --cache=local or --cache=global")
	}

	return promptProjectCacheMode(cmd, current.Mode)
}

func promptProjectCacheMode(cmd *cobra.Command, current project.CacheMode) (project.CacheMode, error) {
	defaultLabel := string(current)
	if defaultLabel == "" {
		defaultLabel = string(project.CacheModeLocal)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Choose project cache mode [%s]:\n", defaultLabel)
	fmt.Fprintln(cmd.OutOrStdout(), "1. local")
	fmt.Fprintln(cmd.OutOrStdout(), "2. global")
	fmt.Fprint(cmd.OutOrStdout(), "> ")

	reader := bufio.NewReader(cmd.InOrStdin())
	choice, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	trimmed := strings.TrimSpace(strings.ToLower(choice))
	if trimmed == "" {
		return current, nil
	}

	switch trimmed {
	case "1", "local":
		return project.CacheModeLocal, nil
	case "2", "global":
		return project.CacheModeGlobal, nil
	default:
		return "", errors.New("invalid cache choice; use --cache=local or --cache=global")
	}
}

func normalizeCacheMode(value string) (project.CacheMode, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "local":
		return project.CacheModeLocal, nil
	case "global":
		return project.CacheModeGlobal, nil
	default:
		return "", fmt.Errorf("invalid cache mode %q: use local or global", value)
	}
}

func isInteractive(in io.Reader, out io.Writer) bool {
	inFile, ok := in.(*os.File)
	if !ok {
		return false
	}
	outFile, ok := out.(*os.File)
	if !ok {
		return false
	}

	inInfo, err := inFile.Stat()
	if err != nil || inInfo.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	outInfo, err := outFile.Stat()
	if err != nil || outInfo.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return true
}
