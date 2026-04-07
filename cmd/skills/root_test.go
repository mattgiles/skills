package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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

	err := cmd.Execute()
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

func assertOutputHasFields(t *testing.T, output string, want ...string) {
	t.Helper()

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) != len(want) {
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
