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

func TestSourceSyncFetchesWithoutChangingHead(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)

	remote := initRemoteRepo(
		t,
		map[string]string{
			"analytics/SKILL.md": "# analytics",
		},
	)

	_, stderr, err := executeCommand(t, env, "source", "add", "repo-one", remote)
	if err != nil {
		t.Fatalf("add source error = %v, stderr = %s", err, stderr)
	}

	if _, stderr, err = executeCommand(t, env, "source", "sync", "repo-one"); err != nil {
		t.Fatalf("initial sync error = %v, stderr = %s", err, stderr)
	}

	localRepo := filepath.Join(env.dataHome, "skills", "repos", "repo-one")
	headBefore := gitOutput(t, localRepo, "rev-parse", "HEAD")

	mustWriteFile(t, filepath.Join(remote, "new-skill", "SKILL.md"), "# new")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "add new skill")

	if _, stderr, err = executeCommand(t, env, "source", "sync", "repo-one"); err != nil {
		t.Fatalf("second sync error = %v, stderr = %s", err, stderr)
	}

	headAfter := gitOutput(t, localRepo, "rev-parse", "HEAD")
	if headAfter != headBefore {
		t.Fatalf("HEAD changed after fetch: before %s after %s", headBefore, headAfter)
	}

	remoteTracking := gitOutput(t, localRepo, "rev-parse", "refs/remotes/origin/main")
	remoteHead := gitOutput(t, remote, "rev-parse", "HEAD")
	if remoteTracking != remoteHead {
		t.Fatalf("origin/main = %s, want %s", remoteTracking, remoteHead)
	}

	stdout, stderr, err := executeCommand(t, env, "source", "list")
	if err != nil {
		t.Fatalf("source list error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "repo-one", "synced", "main@"+remoteHead[:12], "main@"+headBefore[:12])
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

func TestProjectInitCreatesManifest(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "project", "init")
	if err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "created manifest:") {
		t.Fatalf("stdout = %q", stdout)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".skills.yaml")); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
}

func TestProjectSyncCreatesWorktreeAndSymlink(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	commit := gitOutput(t, remote, "rev-parse", "HEAD")

	writeProjectManifest(t, projectDir, manifestFor(remote, map[string][]string{
		"analytics": {"codex"},
	}))

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync")
	if err != nil {
		t.Fatalf("project sync error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "repo-one", "resolved", "main", commit[:12])
	assertOutputHasFields(t, stdout, "codex", "repo-one", "analytics", "created", filepath.Join(resolvedProjectDir, "agent-skills", "analytics"))

	projectID, err := project.ProjectID(resolvedProjectDir)
	if err != nil {
		t.Fatalf("ProjectID() error = %v", err)
	}

	linkPath := filepath.Join(resolvedProjectDir, "agent-skills", "analytics")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", linkPath, err)
	}

	wantTarget := filepath.Join(env.dataHome, "skills", "worktrees", projectID, "repo-one", commit, "analytics")
	if target != wantTarget {
		t.Fatalf("link target = %q, want %q", target, wantTarget)
	}

	if _, err := os.Stat(filepath.Join(env.dataHome, "skills", "repos", "repo-one")); err != nil {
		t.Fatalf("canonical clone missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(resolvedProjectDir, ".skills", "state.yaml")); err != nil {
		t.Fatalf("state file missing: %v", err)
	}

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "up-to-date", "main", commit[:12])
	assertOutputHasFields(t, statusOut, "codex", "repo-one", "analytics", "linked", linkPath)
}

func TestProjectSyncPrunesManagedStaleLinks(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
		"lint/SKILL.md":      "# lint",
	})

	writeProjectManifest(t, projectDir, manifestFor(remote, map[string][]string{
		"analytics": {"codex"},
		"lint":      {"codex"},
	}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	writeProjectManifest(t, projectDir, manifestFor(remote, map[string][]string{
		"analytics": {"codex"},
	}))

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	if !strings.Contains(statusOut, filepath.Join(resolvedProjectDir, "agent-skills", "lint")) {
		t.Fatalf("status output missing stale lint path:\n%s", statusOut)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync")
	if err != nil {
		t.Fatalf("project sync prune error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, filepath.Join(resolvedProjectDir, "agent-skills", "lint")) {
		t.Fatalf("sync output missing pruned lint path:\n%s", stdout)
	}

	if _, err := os.Lstat(filepath.Join(resolvedProjectDir, "agent-skills", "lint")); !os.IsNotExist(err) {
		t.Fatalf("lint symlink should be pruned, got err = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(resolvedProjectDir, "agent-skills", "analytics")); err != nil {
		t.Fatalf("analytics symlink missing after prune: %v", err)
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

	writeProjectManifest(t, projectDir, manifestFor(remote, map[string][]string{
		"analytics": {"codex"},
	}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	mustWriteFile(t, filepath.Join(remote, "README.md"), "next\n")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "advance main")
	commitTwo := gitOutput(t, remote, "rev-parse", "HEAD")

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "up-to-date", "main", commitOne[:12])
	assertOutputHasFields(t, statusOut, "codex", "repo-one", "analytics", "linked", filepath.Join(resolvedProjectDir, "agent-skills", "analytics"))

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "project", "update", "--dry-run")
	if err != nil {
		t.Fatalf("project update --dry-run error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Fatalf("stdout missing dry-run marker:\n%s", stdout)
	}
	assertOutputHasFields(t, stdout, "repo-one", "updated", "main", commitTwo[:12])

	linkPath := filepath.Join(resolvedProjectDir, "agent-skills", "analytics")

	statusOut, statusErr, err = executeCommandInDir(t, env, projectDir, "project", "status")
	if err != nil {
		t.Fatalf("project status after dry-run error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "update-available", "main", commitTwo[:12])
	assertOutputHasFields(t, statusOut, "codex", "repo-one", "analytics", "linked", linkPath)

	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", linkPath, err)
	}
	if !strings.Contains(target, commitOne) {
		t.Fatalf("dry-run should not change link target, got %q", target)
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
	assertOutputHasFields(t, statusOut, "codex", "repo-one", "analytics", "stale", linkPath)

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "project", "sync", "--dry-run")
	if err != nil {
		t.Fatalf("project sync --dry-run error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Fatalf("stdout missing dry-run marker:\n%s", stdout)
	}
	assertOutputHasFields(t, stdout, "codex", "repo-one", "analytics", "would-update", linkPath)

	target, err = os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", linkPath, err)
	}
	if !strings.Contains(target, commitOne) {
		t.Fatalf("sync dry-run should not change link target, got %q", target)
	}

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "project", "sync")
	if err != nil {
		t.Fatalf("project sync error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "repo-one", "up-to-date", "main", commitTwo[:12])
	assertOutputHasFields(t, stdout, "codex", "repo-one", "analytics", "updated", linkPath)

	target, err = os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", linkPath, err)
	}
	if !strings.Contains(target, commitTwo) {
		t.Fatalf("sync should update link target to %s, got %q", commitTwo, target)
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

	writeProjectManifest(t, projectDir, manifestFor(remote, map[string][]string{
		"analytics": {"codex"},
	}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "project", "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	statePath := filepath.Join(projectDir, ".skills", "state.yaml")
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
	assertOutputHasFields(t, stdout, "codex", "repo-one", "analytics", "inspect-failed")
	if !strings.Contains(stdout, "deadbeef") {
		t.Fatalf("status output missing underlying inspect error:\n%s", stdout)
	}
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
	t.Setenv("XDG_CONFIG_HOME", env.configHome)
	t.Setenv("XDG_DATA_HOME", env.dataHome)

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
	mustWriteFile(t, filepath.Join(projectDir, ".skills.yaml"), contents)
}

func manifestFor(remoteURL string, skills map[string][]string) string {
	lines := []string{
		"sources:",
		"  repo-one:",
		"    url: " + remoteURL,
		"    ref: main",
		"agents:",
		"  codex:",
		"    skills_dir: ./agent-skills",
		"skills:",
	}

	names := make([]string, 0, len(skills))
	for name := range skills {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		agents := skills[name]
		lines = append(lines,
			"  - source: repo-one",
			"    name: "+name,
			"    agents: ["+strings.Join(agents, ", ")+"]",
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
