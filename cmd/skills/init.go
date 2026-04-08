package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/project"
)

func newInitCommand() *cobra.Command {
	var globalScope bool
	var cacheMode string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize repo-local state by default, or shared home state with --global",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if globalScope {
				return runHomeInit(cmd)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			projectRoot, err := resolveRepoRoot(cmd.Context(), cwd, false)
			if err != nil {
				return errors.New("outside a Git repo; use skills init --global")
			}
			return runProjectInit(cmd, projectRoot, cacheMode)
		},
	}

	cmd.Flags().BoolVar(&globalScope, "global", false, "Initialize shared home/global state")
	cmd.Flags().StringVar(&cacheMode, "cache", "", "Project cache backend: local or global")
	return cmd
}

func runProjectInit(cmd *cobra.Command, projectDir string, requestedCacheMode string) error {
	cacheMode, err := resolveProjectInitCacheMode(cmd, projectDir, requestedCacheMode)
	if err != nil {
		return err
	}
	result, err := project.InitProject(cmd.Context(), projectDir, project.InitProjectOptions{CacheMode: cacheMode})
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

func resolveProjectInitCacheMode(cmd *cobra.Command, projectDir string, requested string) (project.CacheMode, error) {
	if requested != "" {
		return normalizeCacheMode(requested)
	}

	current, err := project.LoadLocalConfig(projectDir)
	if err != nil {
		return "", err
	}

	if !isInteractive(cmd.InOrStdin(), cmd.OutOrStdout()) {
		if current.Exists {
			return current.Mode, nil
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
		if current != "" {
			return current, nil
		}
		return project.CacheModeLocal, nil
	}

	switch trimmed {
	case "1", "local":
		return project.CacheModeLocal, nil
	case "2", "global":
		return project.CacheModeGlobal, nil
	default:
		return "", markUsage(errors.New("invalid cache choice; use --cache=local or --cache=global"))
	}
}

func normalizeCacheMode(value string) (project.CacheMode, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "local":
		return project.CacheModeLocal, nil
	case "global":
		return project.CacheModeGlobal, nil
	default:
		return "", markUsage(fmt.Errorf("invalid cache mode %q: use local or global", value))
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
