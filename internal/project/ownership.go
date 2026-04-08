package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattgiles/skills/internal/gitrepo"
)

type gitignoreCoverage struct {
	MissingRules []string
}

func inspectProjectOwnership(ctx context.Context, projectDir string) (ProjectOwnershipReport, error) {
	ws := workspace{
		RootDir:         projectDir,
		StatePath:       StatePath(projectDir),
		LocalConfigPath: LocalConfigPath(projectDir),
		SkillsDir:       SkillsDir(projectDir),
		ClaudeSkillsDir: ClaudeSkillsDir(projectDir),
		CacheDir:        CacheDir(projectDir),
	}
	info, err := gitrepo.Discover(ctx, projectDir)
	if err != nil {
		return ProjectOwnershipReport{}, err
	}

	ignoreBase := projectDir
	report := ProjectOwnershipReport{
		GitAvailable: info.Available,
	}
	if info.Root != "" {
		report.InGitRepo = true
		report.GitRoot = info.Root
		ignoreBase = info.Root
	}

	ignorePath := filepath.Join(ignoreBase, ".gitignore")
	rules, pathspecs, err := managedRuntimeIgnoreRules(ignoreBase, ws)
	if err != nil {
		return ProjectOwnershipReport{}, err
	}
	report.GitignorePath = ignorePath
	report.RequiredRules = rules

	coverage, err := inspectGitignoreCoverage(ignorePath, rules)
	if err != nil {
		return ProjectOwnershipReport{}, err
	}
	report.MissingRules = coverage.MissingRules

	if report.InGitRepo {
		tracked, err := gitrepo.ListTracked(ctx, report.GitRoot, pathspecs)
		if err != nil {
			return ProjectOwnershipReport{}, err
		}
		report.TrackedPaths = tracked
	}

	return report, nil
}

func InspectProjectOwnershipContext(ctx context.Context, projectDir string) (ProjectOwnershipReport, error) {
	return inspectProjectOwnership(ctx, projectDir)
}

func InspectProjectOwnership(projectDir string) (ProjectOwnershipReport, error) {
	return InspectProjectOwnershipContext(context.Background(), projectDir)
}

func InspectProjectArtifacts(projectDir string) (ArtifactReport, error) {
	paths := []string{
		ManifestPath(projectDir),
		StatePath(projectDir),
		LocalConfigPath(projectDir),
		SkillsDir(projectDir),
		ClaudeSkillsDir(projectDir),
		CacheDir(projectDir),
	}

	found := make([]string, 0)
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			found = append(found, path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return ArtifactReport{}, err
		}
	}

	return ArtifactReport{
		HasArtifacts: len(found) > 0,
		Paths:        found,
	}, nil
}

func validateManagedPathTypes(ws workspace) error {
	checks := []struct {
		path    string
		wantDir bool
		label   string
	}{
		{path: ws.StatePath, wantDir: false, label: "state path"},
		{path: ws.LocalConfigPath, wantDir: false, label: "local config path"},
		{path: ws.SkillsDir, wantDir: true, label: "skills directory"},
		{path: ws.ClaudeSkillsDir, wantDir: true, label: "Claude skills directory"},
		{path: ws.CacheDir, wantDir: true, label: "cache directory"},
	}

	for _, check := range checks {
		info, err := os.Lstat(check.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if check.wantDir {
			if !info.IsDir() {
				return fmt.Errorf("%s exists and is not a directory: %s", check.label, check.path)
			}
			continue
		}
		if info.IsDir() {
			return fmt.Errorf("%s exists and is a directory: %s", check.label, check.path)
		}
	}

	return nil
}

func managedRuntimeIgnoreRules(ignoreBase string, ws workspace) ([]string, []string, error) {
	managedPaths := []string{ws.StatePath, ws.LocalConfigPath, ws.SkillsDir, ws.ClaudeSkillsDir, ws.CacheDir}
	rules := make([]string, 0, len(managedPaths))
	pathspecs := make([]string, 0, len(managedPaths))

	for _, managedPath := range managedPaths {
		rel, err := filepath.Rel(ignoreBase, managedPath)
		if err != nil {
			return nil, nil, err
		}
		rel = filepath.ToSlash(rel)
		pathspecs = append(pathspecs, rel)
		rule := "/" + rel
		if managedPath == ws.SkillsDir || managedPath == ws.ClaudeSkillsDir || managedPath == ws.CacheDir {
			rule += "/"
		}
		rules = append(rules, rule)
	}

	return rules, pathspecs, nil
}

func inspectGitignoreCoverage(path string, requiredRules []string) (gitignoreCoverage, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return gitignoreCoverage{MissingRules: append([]string(nil), requiredRules...)}, nil
	}
	if err != nil {
		return gitignoreCoverage{}, err
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	trimmed := map[string]struct{}{}
	for _, line := range lines {
		value := strings.TrimSpace(line)
		if value == "" {
			continue
		}
		trimmed[value] = struct{}{}
	}

	missing := make([]string, 0)
	for _, rule := range requiredRules {
		if _, ok := trimmed[rule]; ok {
			continue
		}
		missing = append(missing, rule)
	}

	return gitignoreCoverage{MissingRules: missing}, nil
}

func ensureProjectGitignore(report ProjectOwnershipReport) (bool, error) {
	info, err := os.Stat(report.GitignorePath)
	if err == nil && info.IsDir() {
		return false, fmt.Errorf("gitignore path is a directory: %s", report.GitignorePath)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	var existing string
	if data, err := os.ReadFile(report.GitignorePath); err == nil {
		existing = strings.ReplaceAll(string(data), "\r\n", "\n")
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	baseLines := stripManagedGitignoreBlock(strings.Split(existing, "\n"))
	existingRules := map[string]struct{}{}
	for _, line := range baseLines {
		value := strings.TrimSpace(line)
		if value == "" {
			continue
		}
		existingRules[value] = struct{}{}
	}

	blockRules := make([]string, 0, len(report.RequiredRules))
	for _, rule := range report.RequiredRules {
		if _, ok := existingRules[rule]; ok {
			continue
		}
		blockRules = append(blockRules, rule)
	}

	newContent := buildGitignoreContent(baseLines, blockRules)
	if normalizeGitignore(existing) == normalizeGitignore(newContent) {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(report.GitignorePath), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(report.GitignorePath, []byte(newContent), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func stripManagedGitignoreBlock(lines []string) []string {
	result := make([]string, 0, len(lines))
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case gitignoreBeginMarker:
			inBlock = true
			continue
		case gitignoreEndMarker:
			inBlock = false
			continue
		}
		if inBlock {
			continue
		}
		result = append(result, line)
	}
	return trimTrailingBlankLines(result)
}

func buildGitignoreContent(baseLines []string, blockRules []string) string {
	var out bytes.Buffer

	writeLines := func(lines []string) {
		for _, line := range trimTrailingBlankLines(lines) {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}

	baseLines = trimTrailingBlankLines(baseLines)
	if len(baseLines) > 0 {
		writeLines(baseLines)
	}

	if len(blockRules) > 0 {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(gitignoreBeginMarker)
		out.WriteByte('\n')
		for _, rule := range blockRules {
			out.WriteString(rule)
			out.WriteByte('\n')
		}
		out.WriteString(gitignoreEndMarker)
		out.WriteByte('\n')
	}

	return out.String()
}

func trimTrailingBlankLines(lines []string) []string {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return append([]string(nil), lines[:end]...)
}

func normalizeGitignore(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	lines := trimTrailingBlankLines(strings.Split(value, "\n"))
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
