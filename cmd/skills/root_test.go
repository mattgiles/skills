package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/mattgiles/skills/internal/project"
)

func TestSourceAddPersistsConfig(t *testing.T) {
	env := newTestEnv(t)

	stdout, stderr, err := executeCommand(t, env, "source", "add", "dbt-agent-skills", "https://github.com/dbt-labs/dbt-agent-skills.git")
	if err != nil {
		t.Fatalf("executeCommand() error = %v, stderr = %s", err, stderr)
	}

	if !strings.Contains(stdout, `registered source "dbt-agent-skills"`) {
		t.Fatalf("stdout = %q", stdout)
	}

	configPath := filepath.Join(env.configHome, "skills", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "dbt-agent-skills") {
		t.Fatalf("config file missing source entry: %s", string(data))
	}
}

func TestSourceSyncClonesAndSkillListAggregates(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)

	remoteOne := initRemoteRepo(
		t,
		map[string]string{
			"analytics/SKILL.md": "# analytics",
			"dbt/core/SKILL.md":  "# dbt-core",
		},
	)
	remoteTwo := initRemoteRepo(
		t,
		map[string]string{
			"lint/SKILL.md": "# lint",
		},
	)

	_, stderr, err := executeCommand(t, env, "source", "add", "repo-one", remoteOne)
	if err != nil {
		t.Fatalf("add repo-one error = %v, stderr = %s", err, stderr)
	}
	_, stderr, err = executeCommand(t, env, "source", "add", "repo-two", remoteTwo)
	if err != nil {
		t.Fatalf("add repo-two error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, env, "source", "sync")
	if err != nil {
		t.Fatalf("sync error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "cloned\trepo-one") || !strings.Contains(stdout, "cloned\trepo-two") {
		t.Fatalf("sync stdout = %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, env, "skill", "list")
	if err != nil {
		t.Fatalf("skill list error = %v, stderr = %s", err, stderr)
	}

	assertOutputHasFields(t, stdout, "repo-one", "analytics", "analytics")
	assertOutputHasFields(t, stdout, "repo-one", "core", filepath.Join("dbt", "core"))
	assertOutputHasFields(t, stdout, "repo-two", "lint", "lint")
}

func TestSkillListSkipsUnsyncedSource(t *testing.T) {
	env := newTestEnv(t)

	_, stderr, err := executeCommand(t, env, "source", "add", "repo-one", "/tmp/does-not-matter.git")
	if err != nil {
		t.Fatalf("add source error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, env, "skill", "list")
	if err != nil {
		t.Fatalf("skill list error = %v, stderr = %s", err, stderr)
	}

	if !strings.Contains(stdout, "no skills found") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr, `warning: skipping unsynced source "repo-one"`) {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestVersionCommandShowsBuildInfo(t *testing.T) {
	env := newTestEnv(t)

	stdout, stderr, err := executeCommand(t, env, "version")
	if err != nil {
		t.Fatalf("version error = %v, stderr = %s", err, stderr)
	}

	for _, want := range []string{"version=dev", "commit=unknown", "date=unknown", "platform="} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestProjectInitCreatesStandardizedWorkspace(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "project", "init")
	if err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "created manifest:") {
		t.Fatalf("stdout = %q", stdout)
	}

	for _, path := range []string{
		filepath.Join(projectDir, ".agents", "manifest.yaml"),
		filepath.Join(projectDir, "AGENTS.md"),
		filepath.Join(projectDir, "CLAUDE.md"),
		filepath.Join(projectDir, ".agents", "skills"),
		filepath.Join(projectDir, ".claude", "skills"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected path %q: %v", path, err)
		}
	}
}

func TestProjectSyncCreatesCanonicalAndClaudeLinks(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	commit := gitOutput(t, remote, "rev-parse", "HEAD")

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync")
	if err != nil {
		t.Fatalf("project sync error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "repo-one", "resolved", "main", commit[:12])

	canonicalPath := filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics")
	claudePath := filepath.Join(resolvedProjectDir, ".claude", "skills", "analytics")
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "created", canonicalPath)
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "created", claudePath)

	projectID, err := project.ProjectID(resolvedProjectDir)
	if err != nil {
		t.Fatalf("ProjectID() error = %v", err)
	}

	target, err := os.Readlink(canonicalPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", canonicalPath, err)
	}

	wantTarget := filepath.Join(env.dataHome, "skills", "worktrees", projectID, "repo-one", commit, "analytics")
	if target != wantTarget {
		t.Fatalf("canonical target = %q, want %q", target, wantTarget)
	}

	claudeTarget, err := os.Readlink(claudePath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", claudePath, err)
	}
	if claudeTarget != canonicalPath {
		t.Fatalf("claude target = %q, want %q", claudeTarget, canonicalPath)
	}

	if _, err := os.Stat(filepath.Join(resolvedProjectDir, ".agents", "state.yaml")); err != nil {
		t.Fatalf("state file missing: %v", err)
	}

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "up-to-date", "main", commit[:12])
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", canonicalPath)
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", claudePath)
}

func TestProjectSyncPrunesManagedCanonicalAndClaudeLinks(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
		"lint/SKILL.md":      "# lint",
	})

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics", "lint"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	if !strings.Contains(statusOut, filepath.Join(resolvedProjectDir, ".agents", "skills", "lint")) {
		t.Fatalf("status output missing stale lint canonical path:\n%s", statusOut)
	}
	if !strings.Contains(statusOut, filepath.Join(resolvedProjectDir, ".claude", "skills", "lint")) {
		t.Fatalf("status output missing stale lint claude path:\n%s", statusOut)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync")
	if err != nil {
		t.Fatalf("project sync prune error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, filepath.Join(resolvedProjectDir, ".agents", "skills", "lint")) {
		t.Fatalf("sync output missing pruned canonical lint path:\n%s", stdout)
	}
	if !strings.Contains(stdout, filepath.Join(resolvedProjectDir, ".claude", "skills", "lint")) {
		t.Fatalf("sync output missing pruned claude lint path:\n%s", stdout)
	}

	for _, path := range []string{
		filepath.Join(resolvedProjectDir, ".agents", "skills", "lint"),
		filepath.Join(resolvedProjectDir, ".claude", "skills", "lint"),
	} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %q to be pruned, got err = %v", path, err)
		}
	}
}

func TestProjectUpdateAndDryRunFlow(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	commitOne := gitOutput(t, remote, "rev-parse", "HEAD")

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	mustWriteFile(t, filepath.Join(remote, "README.md"), "next\n")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "advance main")
	commitTwo := gitOutput(t, remote, "rev-parse", "HEAD")

	canonicalPath := filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics")
	claudePath := filepath.Join(resolvedProjectDir, ".claude", "skills", "analytics")

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "up-to-date", "main", commitOne[:12])
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", canonicalPath)
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", claudePath)

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "project", "update", "--dry-run")
	if err != nil {
		t.Fatalf("project update --dry-run error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Fatalf("stdout missing dry-run marker:\n%s", stdout)
	}
	assertOutputHasFields(t, stdout, "repo-one", "updated", "main", commitTwo[:12])

	statusOut, statusErr, err = executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status after dry-run error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "update-available", "main", commitTwo[:12])
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", canonicalPath)
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", claudePath)

	target, err := os.Readlink(canonicalPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", canonicalPath, err)
	}
	if !strings.Contains(target, commitOne) {
		t.Fatalf("dry-run should not change canonical link target, got %q", target)
	}

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "project", "update")
	if err != nil {
		t.Fatalf("project update error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "repo-one", "updated", "main", commitTwo[:12])

	statusOut, statusErr, err = executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "up-to-date", "main", commitTwo[:12])
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "stale", canonicalPath)
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", claudePath)

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "project", "sync", "--dry-run")
	if err != nil {
		t.Fatalf("project sync --dry-run error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Fatalf("stdout missing dry-run marker:\n%s", stdout)
	}
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "would-update", canonicalPath)
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "linked", claudePath)

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "project", "sync")
	if err != nil {
		t.Fatalf("project sync error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "repo-one", "up-to-date", "main", commitTwo[:12])
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "updated", canonicalPath)
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "linked", claudePath)

	target, err = os.Readlink(canonicalPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", canonicalPath, err)
	}
	if !strings.Contains(target, commitTwo) {
		t.Fatalf("sync should update canonical link target to %s, got %q", commitTwo, target)
	}
}

func TestProjectStatusReportsInspectFailure(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	commit := gitOutput(t, remote, "rev-parse", "HEAD")

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	statePath := filepath.Join(projectDir, ".agents", "state.yaml")
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", statePath, err)
	}
	replaced := strings.Replace(string(stateData), commit, "deadbeef", 1)
	if err := os.WriteFile(statePath, []byte(replaced), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", statePath, err)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, stderr)
	}

	assertOutputHasFields(t, stdout, "repo-one", "inspect-failed", "main", commit[:12])
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "inspect-failed")
	if !strings.Contains(stdout, "deadbeef") {
		t.Fatalf("status output missing underlying inspect error:\n%s", stdout)
	}
}

func TestHomeInitAndSyncUsesSeparateSharedPaths(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)

	stdout, stderr, err := executeCommand(t, env, "home", "init")
	if err != nil {
		t.Fatalf("home init error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "created manifest:") {
		t.Fatalf("stdout = %q", stdout)
	}

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	manifestPath := filepath.Join(env.home, ".agents", "manifest.yaml")
	mustWriteFile(t, manifestPath, manifestFor(remote, []string{"analytics"}))

	stdout, stderr, err = executeCommand(t, env, "home", "sync")
	if err != nil {
		t.Fatalf("home sync error = %v, stderr = %s", err, stderr)
	}

	assertOutputHasFields(t, stdout, "repo-one", "analytics", "created", filepath.Join(env.home, ".agents", "skills", "analytics"))
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "created", filepath.Join(env.home, ".claude", "skills", "analytics"))
}

type testEnv struct {
	configHome string
	dataHome   string
	home       string
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()

	root := t.TempDir()
	return testEnv{
		configHome: filepath.Join(root, "config"),
		dataHome:   filepath.Join(root, "data"),
		home:       filepath.Join(root, "home"),
	}
}

func executeCommand(t *testing.T, env testEnv, args ...string) (string, string, error) {
	return executeCommandInDir(t, env, "", args...)
}

func executeCommandInDir(t *testing.T, env testEnv, dir string, args ...string) (string, string, error) {
	t.Helper()

	cmd := newRootCommand()
	cmd.SetArgs(args)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	t.Setenv("HOME", env.home)
	t.Setenv("SKILLS_CONFIG_HOME", env.configHome)
	t.Setenv("SKILLS_DATA_HOME", env.dataHome)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("Chdir(%q) error = %v", dir, err)
		}
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	err = cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initRemoteRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Codex Test")
	runGit(t, repo, "config", "user.email", "codex@example.com")

	for path, contents := range files {
		mustWriteFile(t, filepath.Join(repo, path), contents)
	}

	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return strings.TrimSpace(string(output))
}

func mustWriteFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeProjectManifest(t *testing.T, projectDir string, contents string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(projectDir, ".agents", "manifest.yaml"), contents)
}

func manifestFor(remoteURL string, skills []string) string {
	lines := []string{
		"sources:",
		"  repo-one:",
		"    url: " + remoteURL,
		"    ref: main",
		"skills:",
	}

	sort.Strings(skills)
	for _, name := range skills {
		lines = append(lines,
			"  - source: repo-one",
			"    name: "+name,
		)
	}

	return strings.Join(lines, "\n") + "\n"
}

func assertOutputHasFields(t *testing.T, output string, want ...string) {
	t.Helper()

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < len(want) {
			continue
		}

		matched := true
		for i := range want {
			if fields[i] != want[i] {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}

	t.Fatalf("output missing row %v:\n%s", want, output)
}

func resolvedPath(t *testing.T, path string) string {
	t.Helper()

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", path, err)
	}
	return resolved
}
