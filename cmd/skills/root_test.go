package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/pflag"

	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/selfupdate"
)

func TestSourceAddGlobalPersistsHomeManifest(t *testing.T) {
	env := newTestEnv(t)

	if _, stderr, err := executeCommand(t, env, "init", "--global"); err != nil {
		t.Fatalf("init --global error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, env, "source", "add", "--global", "--ref", "main", "dbt-agent-skills", "https://github.com/dbt-labs/dbt-agent-skills.git")
	if err != nil {
		t.Fatalf("executeCommand() error = %v, stderr = %s", err, stderr)
	}

	if !strings.Contains(stdout, `registered source "dbt-agent-skills"`) {
		t.Fatalf("stdout = %q", stdout)
	}

	manifestPath := filepath.Join(env.home, ".agents", "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "dbt-agent-skills") || !strings.Contains(string(data), "url: https://github.com/dbt-labs/dbt-agent-skills.git") {
		t.Fatalf("home manifest missing source entry: %s", string(data))
	}
}

func TestSourceSyncClonesAndSkillListAggregatesInGlobalScope(t *testing.T) {
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

	if _, stderr, err := executeCommand(t, env, "init", "--global"); err != nil {
		t.Fatalf("init --global error = %v, stderr = %s", err, stderr)
	}

	_, stderr, err := executeCommand(t, env, "source", "add", "--global", "--ref", "main", "repo-one", remoteOne)
	if err != nil {
		t.Fatalf("add repo-one error = %v, stderr = %s", err, stderr)
	}
	_, stderr, err = executeCommand(t, env, "source", "add", "--global", "--ref", "main", "repo-two", remoteTwo)
	if err != nil {
		t.Fatalf("add repo-two error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, env, "source", "sync", "--global")
	if err != nil {
		t.Fatalf("sync error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "cloned  repo-one") || !strings.Contains(stdout, "cloned  repo-two") {
		t.Fatalf("sync stdout = %q", stdout)
	}

	stdout, stderr, err = executeCommand(t, env, "skill", "list", "--global")
	if err != nil {
		t.Fatalf("skill list error = %v, stderr = %s", err, stderr)
	}

	assertOutputHasFields(t, stdout, "repo-one", "analytics", "analytics")
	assertOutputHasFields(t, stdout, "repo-one", "core", filepath.Join("dbt", "core"))
	assertOutputHasFields(t, stdout, "repo-two", "lint", "lint")

	mustWriteFile(t, filepath.Join(remoteOne, "ops", "SKILL.md"), "# ops")
	runGit(t, remoteOne, "add", ".")
	runGit(t, remoteOne, "commit", "-m", "add ops")

	if _, stderr, err := executeCommand(t, env, "source", "sync", "--global", "repo-one"); err != nil {
		t.Fatalf("sync repo-one error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err = executeCommand(t, env, "skill", "list", "--global", "--source", "repo-one")
	if err != nil {
		t.Fatalf("skill list after fetch error = %v, stderr = %s", err, stderr)
	}

	assertOutputHasFields(t, stdout, "repo-one", "analytics", "analytics")
	assertOutputHasFields(t, stdout, "repo-one", "core", filepath.Join("dbt", "core"))
	assertOutputHasFields(t, stdout, "repo-one", "ops", "ops")
}

func TestSkillListSkipsUnsyncedSource(t *testing.T) {
	env := newTestEnv(t)

	if _, stderr, err := executeCommand(t, env, "init", "--global"); err != nil {
		t.Fatalf("init --global error = %v, stderr = %s", err, stderr)
	}

	_, stderr, err := executeCommand(t, env, "source", "add", "--global", "--ref", "main", "repo-one", "/tmp/does-not-matter.git")
	if err != nil {
		t.Fatalf("add source error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, env, "skill", "list", "--global")
	if err != nil {
		t.Fatalf("skill list error = %v, stderr = %s", err, stderr)
	}

	if !strings.Contains(stdout, "no skills found") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr, `WARNING: skipping unsynced source "repo-one"`) {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestRepoSourceCommandsUseProjectManifest(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "source", "add", "--ref", "main", "repo-one", remote); err != nil {
		t.Fatalf("source add error = %v, stderr = %s", err, stderr)
	}

	manifestData, err := os.ReadFile(filepath.Join(projectDir, ".agents", "manifest.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	if !strings.Contains(string(manifestData), "repo-one:") || !strings.Contains(string(manifestData), "url: "+remote) {
		t.Fatalf("manifest missing source entry:\n%s", string(manifestData))
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "source", "sync")
	if err != nil {
		t.Fatalf("source sync error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "cloned  repo-one") {
		t.Fatalf("sync stdout = %q", stdout)
	}
}

func TestAddCommandAddsSkillToExistingRepoSourceAndSyncs(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"skills: []",
		"",
	}, "\n"))

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "analytics")
	if err != nil {
		t.Fatalf("add error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `added skill "analytics" from source "repo-one"`) {
		t.Fatalf("stdout = %q", stdout)
	}

	manifestData, err := os.ReadFile(filepath.Join(projectDir, ".agents", "manifest.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	if !strings.Contains(string(manifestData), "name: analytics") {
		t.Fatalf("manifest missing skill entry:\n%s", string(manifestData))
	}

	canonicalPath := filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics")
	if _, err := os.Lstat(canonicalPath); err != nil {
		t.Fatalf("expected canonical skill link %q: %v", canonicalPath, err)
	}
}

func TestAddCommandAdvancesOnlyTargetSourceWhenNewSkillExistsUpstream(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)
	initGitRepo(t, projectDir)

	remoteOne := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	remoteTwo := initRemoteRepo(t, map[string]string{
		"lint/SKILL.md": "# lint",
	})

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remoteOne,
		"    ref: main",
		"  repo-two:",
		"    url: " + remoteTwo,
		"    ref: main",
		"skills:",
		"  - source: repo-one",
		"    name: analytics",
		"  - source: repo-two",
		"    name: lint",
		"",
	}, "\n"))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
		t.Fatalf("initial sync error = %v, stderr = %s", err, stderr)
	}

	commitTwoBefore := gitOutput(t, remoteTwo, "rev-parse", "HEAD")
	mustWriteFile(t, filepath.Join(remoteOne, "partner-project-inspector", "SKILL.md"), "# partner-project-inspector")
	runGit(t, remoteOne, "add", ".")
	runGit(t, remoteOne, "commit", "-m", "add partner-project-inspector")
	commitOneAfter := gitOutput(t, remoteOne, "rev-parse", "HEAD")

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "partner-project-inspector")
	if err != nil {
		t.Fatalf("add error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `added skill "partner-project-inspector" from source "repo-one"`) {
		t.Fatalf("stdout = %q", stdout)
	}
	assertOutputHasFields(t, stdout, "repo-one", "up-to-date", "main", commitOneAfter[:12])
	assertOutputHasFields(t, stdout, "repo-two", "up-to-date", "main", commitTwoBefore[:12])

	manifestData, err := os.ReadFile(filepath.Join(projectDir, ".agents", "manifest.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	if !strings.Contains(string(manifestData), "name: partner-project-inspector") {
		t.Fatalf("manifest missing added skill entry:\n%s", string(manifestData))
	}

	canonicalPath := filepath.Join(resolvedProjectDir, ".agents", "skills", "partner-project-inspector")
	if _, err := os.Lstat(canonicalPath); err != nil {
		t.Fatalf("expected canonical skill link %q: %v", canonicalPath, err)
	}

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "status")
	if err != nil {
		t.Fatalf("status error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "up-to-date", "main", commitOneAfter[:12])
	assertOutputHasFields(t, statusOut, "repo-two", "up-to-date", "main", commitTwoBefore[:12])
	assertOutputHasFields(t, statusOut, "repo-one", "partner-project-inspector", "linked", canonicalPath)
}

func TestAddCommandAddsSkillToExistingGlobalSourceAndSyncs(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, stderr, err := executeCommand(t, env, "init", "--global"); err != nil {
		t.Fatalf("init --global error = %v, stderr = %s", err, stderr)
	}

	manifestPath := filepath.Join(env.home, ".agents", "manifest.yaml")
	mustWriteFile(t, manifestPath, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"skills: []",
		"",
	}, "\n"))

	stdout, stderr, err := executeCommand(t, env, "add", "--global", "repo-one", "analytics")
	if err != nil {
		t.Fatalf("add --global error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `added skill "analytics" from source "repo-one"`) {
		t.Fatalf("stdout = %q", stdout)
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	if !strings.Contains(string(manifestData), "name: analytics") {
		t.Fatalf("manifest missing skill entry:\n%s", string(manifestData))
	}

	canonicalPath := filepath.Join(env.home, ".agents", "skills", "analytics")
	if _, err := os.Lstat(canonicalPath); err != nil {
		t.Fatalf("expected canonical skill link %q: %v", canonicalPath, err)
	}
}

func TestAddCommandAddsNewRepoSourceWithExplicitRefAndSyncs(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "--url", remote, "--ref", "main", "repo-one", "analytics")
	if err != nil {
		t.Fatalf("add error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `added source "repo-one" (`) {
		t.Fatalf("stdout = %q", stdout)
	}

	manifestData, err := os.ReadFile(filepath.Join(projectDir, ".agents", "manifest.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	for _, want := range []string{
		"repo-one:",
		"url: " + remote,
		"ref: main",
		"name: analytics",
	} {
		if !strings.Contains(string(manifestData), want) {
			t.Fatalf("manifest missing %q:\n%s", want, string(manifestData))
		}
	}

	canonicalPath := filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics")
	if _, err := os.Lstat(canonicalPath); err != nil {
		t.Fatalf("expected canonical skill link %q: %v", canonicalPath, err)
	}
}

func TestAddCommandInfersRefForNewGlobalSource(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, stderr, err := executeCommand(t, env, "init", "--global"); err != nil {
		t.Fatalf("init --global error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommand(t, env, "add", "--global", "--url", remote, "repo-one", "analytics")
	if err != nil {
		t.Fatalf("add --global error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `added source "repo-one" (`) {
		t.Fatalf("stdout = %q", stdout)
	}

	manifestPath := filepath.Join(env.home, ".agents", "manifest.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	for _, want := range []string{
		"url: " + remote,
		"ref: main",
		"name: analytics",
	} {
		if !strings.Contains(string(manifestData), want) {
			t.Fatalf("manifest missing %q:\n%s", want, string(manifestData))
		}
	}

	configPath := filepath.Join(env.configHome, "skills", "config.yaml")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("did not expect global config write, got err = %v", err)
	}
}

func TestAddCommandNoOpsForDuplicateSkill(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "analytics")
	if err != nil {
		t.Fatalf("add error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `skill "analytics" from source "repo-one" is already declared`) {
		t.Fatalf("stdout = %q", stdout)
	}

	manifestData, err := os.ReadFile(filepath.Join(projectDir, ".agents", "manifest.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	if strings.Count(string(manifestData), "name: analytics") != 1 {
		t.Fatalf("manifest duplicated skill entry:\n%s", string(manifestData))
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "state.yaml")); !os.IsNotExist(err) {
		t.Fatalf("did not expect sync state write, got err = %v", err)
	}
}

func TestAddCommandRequiresURLForNewSourceBeforeWritingManifest(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}

	manifestPath := filepath.Join(projectDir, ".agents", "manifest.yaml")
	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "analytics")
	if err == nil {
		t.Fatalf("expected add error, stdout = %s, stderr = %s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), `source "repo-one" is not declared; --url is required`) {
		t.Fatalf("unexpected error: %v", err)
	}

	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("manifest changed unexpectedly:\nbefore:\n%s\nafter:\n%s", string(before), string(after))
	}
}

func TestAddCommandFailsOutsideRepo(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "analytics")
	if err == nil {
		t.Fatalf("expected add error, stdout = %s, stderr = %s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "outside a Git repo; use --global") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddCommandFailsWithoutRepoManifest(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "analytics")
	if err == nil {
		t.Fatalf("expected add error, stdout = %s, stderr = %s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "no repo manifest found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddCommandRollsBackManifestOnSyncFailure(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"lint/SKILL.md": "# lint",
	})

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}

	manifestPath := filepath.Join(projectDir, ".agents", "manifest.yaml")
	original := strings.Join([]string{
		"sources:",
		"  repo-one: {url: " + remote + ", ref: main}",
		"skills: []",
		"",
	}, "\n")
	mustWriteFile(t, manifestPath, original)

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "analytics")
	if err == nil {
		t.Fatalf("expected add error, stdout = %s, stderr = %s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "repo-one/analytics: missing-skill") {
		t.Fatalf("unexpected error: %v", err)
	}

	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	if string(after) != original {
		t.Fatalf("manifest rollback mismatch:\nwant:\n%s\ngot:\n%s", original, string(after))
	}
}

func TestSkillListUsesRepoManifestSourcesByDefault(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"skills/golang-cli/SKILL.md":  "# golang-cli",
		"skills/golang-http/SKILL.md": "# golang-http",
	})

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=global"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"golang-cli"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
		t.Fatalf("sync error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "skill", "list")
	if err != nil {
		t.Fatalf("skill list error = %v, stderr = %s", err, stderr)
	}

	assertOutputHasFields(t, stdout, "repo-one", "golang-cli", filepath.Join("skills", "golang-cli"))
	assertOutputHasFields(t, stdout, "repo-one", "golang-http", filepath.Join("skills", "golang-http"))
}

func TestVersionCommandShowsBuildInfo(t *testing.T) {
	env := newTestEnv(t)

	stdout, stderr, err := executeCommand(t, env, "version")
	if err != nil {
		t.Fatalf("version error = %v, stderr = %s", err, stderr)
	}

	for _, want := range []string{"Version", "Commit", "Date", "Platform"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSelfUpdateCommandReportsUpdate(t *testing.T) {
	env := newTestEnv(t)

	previous := runSelfUpdate
	runSelfUpdate = func(options selfupdate.Options) (selfupdate.Result, error) {
		return selfupdate.Result{
			PreviousVersion: options.CurrentVersion,
			Version:         "v1.2.3",
			TargetPath:      "/tmp/skills",
			Updated:         true,
		}, nil
	}
	t.Cleanup(func() {
		runSelfUpdate = previous
	})

	stdout, stderr, err := executeCommand(t, env, "self", "update", "--version", "v1.2.3")
	if err != nil {
		t.Fatalf("self update error = %v, stderr = %s", err, stderr)
	}
	for _, want := range []string{
		"updated skills from dev to v1.2.3",
		"binary: /tmp/skills",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorProjectHealthy(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	initGitRepo(t, projectDir)
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
		t.Fatalf("project sync error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "doctor", "--verbose")
	if err != nil {
		t.Fatalf("doctor error = %v, stderr = %s", err, stderr)
	}

	for _, want := range []string{
		"ENVIRONMENT",
		"CONFIG",
		"WORKSPACE",
		"GIT",
		"SOURCES",
		"SKILLS",
		"CLAUDE",
		"HINTS",
		"project-cache-mode",
		"config-not-required",
		"doctor: 0 errors, 0 warnings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorProjectMissingManifest(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "doctor", "--verbose")
	if err == nil {
		t.Fatalf("expected doctor error, stdout = %s, stderr = %s", stdout, stderr)
	}

	for _, want := range []string{
		"manifest-missing",
		"git-repo-root",
		"project manifest not found",
		"run skills init",
		"not checked because project manifest is missing",
		"doctor: 1 errors, 2 warnings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorGlobalMissingHomeManifestIsWarning(t *testing.T) {
	env := newTestEnv(t)

	stdout, stderr, err := executeCommand(t, env, "doctor", "--global")
	if err != nil {
		t.Fatalf("doctor --global error = %v, stderr = %s", err, stderr)
	}

	for _, want := range []string{
		"manifest-missing",
		"home manifest not found",
		"run skills init --global",
		"doctor: 0 errors, 1 warnings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorConfigParseFailure(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()

	initGitRepo(t, projectDir)
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}

	configPath := filepath.Join(env.configHome, "skills", "config.yaml")
	mustWriteFile(t, configPath, "sources: [\n")

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "doctor", "--verbose")
	if err != nil {
		t.Fatalf("project doctor should ignore global config parse failures, got err = %v, stderr = %s", err, stderr)
	}

	for _, want := range []string{
		"project install scope is repo-local and cache mode is local",
		"doctor: 0 errors, 2 warnings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorProjectMalformedLocalConfigStillRendersReport(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()

	initGitRepo(t, projectDir)
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	mustWriteFile(t, filepath.Join(projectDir, ".agents", "local.yaml"), "cache: [\n")

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "doctor", "--verbose")
	if err == nil {
		t.Fatalf("expected doctor error, stdout = %s, stderr = %s", stdout, stderr)
	}

	for _, want := range []string{
		"local-config-parse-failed",
		"doctor: 1 errors, 0 warnings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorGlobalMalformedConfigStillRendersReport(t *testing.T) {
	env := newTestEnv(t)

	configPath := filepath.Join(env.configHome, "skills", "config.yaml")
	mustWriteFile(t, configPath, "sources: [\n")

	stdout, stderr, err := executeCommand(t, env, "doctor", "--global")
	if err == nil {
		t.Fatalf("expected doctor error, stdout = %s, stderr = %s", stdout, stderr)
	}

	for _, want := range []string{
		"config-parse-failed",
		"doctor: 1 errors, 0 warnings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorWarnsAboutStaleManagedLinks(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
		"lint/SKILL.md":      "# lint",
	})
	initGitRepo(t, projectDir)
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics", "lint"}))
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "doctor")
	if err != nil {
		t.Fatalf("doctor error = %v, stderr = %s", err, stderr)
	}

	for _, want := range []string{
		"stale-managed-link",
		"managed skill link is no longer declared",
		"managed Claude adapter link is no longer declared",
		"run skills sync",
		"doctor: 0 errors, 2 warnings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestProjectInitCreatesStandardizedWorkspace(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local")
	if err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "created manifest:") {
		t.Fatalf("stdout = %q", stdout)
	}

	for _, path := range []string{
		filepath.Join(projectDir, ".agents", "manifest.yaml"),
		filepath.Join(projectDir, ".agents", "local.yaml"),
		filepath.Join(projectDir, ".agents", "skills"),
		filepath.Join(projectDir, ".agents", "cache"),
		filepath.Join(projectDir, ".agents", "cache", "repos"),
		filepath.Join(projectDir, ".agents", "cache", "worktrees"),
		filepath.Join(projectDir, ".claude", "skills"),
		filepath.Join(projectDir, ".gitignore"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected path %q: %v", path, err)
		}
	}

	gitignoreData, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(.gitignore) error = %v", err)
	}
	for _, want := range []string{
		"# BEGIN skills managed runtime artifacts",
		"/.agents/state.yaml",
		"/.agents/local.yaml",
		"/.agents/skills/",
		"/.agents/cache/",
		"/.claude/skills/",
		"# END skills managed runtime artifacts",
	} {
		if !strings.Contains(string(gitignoreData), want) {
			t.Fatalf(".gitignore missing %q:\n%s", want, string(gitignoreData))
		}
	}

	for _, path := range []string{
		filepath.Join(projectDir, "AGENTS.md"),
		filepath.Join(projectDir, "CLAUDE.md"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("did not expect %q to be created, got err=%v", path, err)
		}
	}
}

func TestProjectInitUsesRepoRootGitignoreForNestedProject(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	repoRoot := t.TempDir()
	initGitRepo(t, repoRoot)

	projectDir := filepath.Join(repoRoot, "apps", "nested")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", projectDir, err)
	}
	mustWriteFile(t, filepath.Join(repoRoot, ".gitignore"), "# existing\n")

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local")
	if err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	resolvedRepoRoot := resolvedPath(t, repoRoot)
	if !strings.Contains(stdout, "updated gitignore: "+filepath.Join(resolvedRepoRoot, ".gitignore")) {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "created manifest: "+filepath.Join(resolvedRepoRoot, ".agents", "manifest.yaml")) {
		t.Fatalf("stdout = %q", stdout)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(repo .gitignore) error = %v", err)
	}
	for _, want := range []string{
		"# existing",
		"/.agents/state.yaml",
		"/.agents/local.yaml",
		"/.agents/skills/",
		"/.agents/cache/",
		"/.claude/skills/",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("repo .gitignore missing %q:\n%s", want, string(data))
		}
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "manifest.yaml")); !os.IsNotExist(err) {
		t.Fatalf("did not expect nested project workspace, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".agents", "manifest.yaml")); err != nil {
		t.Fatalf("expected repo-root manifest: %v", err)
	}
}

func TestProjectInitIsIdempotentForGitignoreRules(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()

	initGitRepo(t, projectDir)
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("first project init error = %v, stderr = %s", err, stderr)
	}
	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local")
	if err != nil {
		t.Fatalf("second project init error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "manifest already exists:") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "cache mode: local") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "gitignore already covers managed runtime artifacts:") {
		t.Fatalf("stdout = %q", stdout)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(.gitignore) error = %v", err)
	}
	if strings.Count(string(data), "# BEGIN skills managed runtime artifacts") != 1 {
		t.Fatalf(".gitignore duplicated managed block:\n%s", string(data))
	}
}

func TestProjectInitFailsWhenManagedPathsAreTracked(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	mustWriteFile(t, filepath.Join(projectDir, ".agents", "skills", "legacy", "README.md"), "tracked\n")
	runGit(t, projectDir, "add", ".agents/skills/legacy/README.md")
	runGit(t, projectDir, "commit", "-m", "tracked managed path")

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local")
	if err == nil {
		t.Fatalf("expected project init error, stdout = %s, stderr = %s", stdout, stderr)
	}
	for _, want := range []string{
		"managed runtime paths already contain tracked Git content",
		".agents/skills/legacy/README.md",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "manifest.yaml")); !os.IsNotExist(err) {
		t.Fatalf("did not expect manifest to be created, got err=%v", err)
	}
}

func TestDoctorWarnsWhenManagedIgnoreRulesAreMissing(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	initGitRepo(t, projectDir)
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
		t.Fatalf("project sync error = %v, stderr = %s", err, stderr)
	}
	mustWriteFile(t, filepath.Join(projectDir, ".gitignore"), "# no managed rules\n")

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "doctor")
	if err != nil {
		t.Fatalf("doctor error = %v, stderr = %s", err, stderr)
	}
	for _, want := range []string{
		"ignore-rules-missing",
		"run skills init",
		"doctor: 0 errors, 1 warnings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorErrorsWhenManagedPathsAreTracked(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	mustWriteFile(t, filepath.Join(projectDir, ".agents", "manifest.yaml"), "sources: {}\nskills: []\n")
	mustWriteFile(t, filepath.Join(projectDir, ".agents", "skills", "legacy", "README.md"), "tracked\n")
	mustWriteFile(t, filepath.Join(projectDir, ".gitignore"), "# not enough\n")
	runGit(t, projectDir, "add", ".agents/manifest.yaml", ".agents/skills/legacy/README.md")
	runGit(t, projectDir, "commit", "-m", "tracked managed path")

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "doctor")
	if err == nil {
		t.Fatalf("expected doctor error, stdout = %s, stderr = %s", stdout, stderr)
	}
	for _, want := range []string{
		"tracked-managed-path",
		".agents/skills/legacy/README.md",
		"doctor: 1 errors, 4 warnings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSkillsInitCreatesRepoLocalWorkspaceWithoutGlobalConfig(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	stdout, stderr, err := executeCommandInDirWithInput(t, env, projectDir, strings.NewReader(""), "init", "--cache=local")
	if err != nil {
		t.Fatalf("skills init error = %v, stderr = %s", err, stderr)
	}
	for _, path := range []string{
		filepath.Join(projectDir, ".agents", "manifest.yaml"),
		filepath.Join(projectDir, ".agents", "local.yaml"),
		filepath.Join(projectDir, ".agents", "cache", "repos"),
		filepath.Join(projectDir, ".agents", "cache", "worktrees"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected path %q: %v", path, err)
		}
	}
	if !strings.Contains(stdout, "created manifest:") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "cache mode: local") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestSkillsInitRequiresCacheModeWhenRepoIsNotConfigured(t *testing.T) {
	env := newTestEnv(t)
	repoRoot := t.TempDir()
	initGitRepo(t, repoRoot)

	stdout, stderr, err := executeCommandInDirWithInput(t, env, repoRoot, strings.NewReader(""), "init")
	if err == nil {
		t.Fatalf("expected skills init to fail, stdout = %s, stderr = %s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "project cache mode is not configured yet; use --cache=local or --cache=global") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectInitWithGlobalCacheCreatesLocalSettingsWithoutLocalCacheDirs(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=global")
	if err != nil {
		t.Fatalf("project init --cache=global error = %v, stderr = %s", err, stderr)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "local.yaml")); err != nil {
		t.Fatalf("expected local config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "cache")); !os.IsNotExist(err) {
		t.Fatalf("did not expect local cache dir, got err=%v", err)
	}
	if !strings.Contains(stdout, "cache mode: global") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestProjectSyncWithGlobalCacheUsesSharedRootsButRepoLocalInstalls(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)
	initGitRepo(t, projectDir)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=global"); err != nil {
		t.Fatalf("project init --cache=global error = %v, stderr = %s", err, stderr)
	}

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	commit := gitOutput(t, remote, "rev-parse", "HEAD")
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
		t.Fatalf("project sync error = %v, stderr = %s", err, stderr)
	}

	canonicalPath := filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics")
	target, err := os.Readlink(canonicalPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", canonicalPath, err)
	}

	projectID, err := project.ProjectID(resolvedProjectDir)
	if err != nil {
		t.Fatalf("ProjectID() error = %v", err)
	}

	wantTarget := filepath.Join(env.dataHome, "skills", "worktrees", projectID, "repo-one", commit, "analytics")
	if target != wantTarget {
		t.Fatalf("canonical target = %q, want %q", target, wantTarget)
	}
}

func TestSkillsInitFailsOutsideRepoWithoutExplicitScope(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()

	stdout, stderr, err := executeCommandInDirWithInput(t, env, projectDir, strings.NewReader(""), "init")
	if err == nil {
		t.Fatalf("expected skills init error, stdout = %s, stderr = %s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "outside a Git repo; use skills init --global") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSkillsInitRoutesToRepoRootWhenArtifactsExist(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	repoRoot := t.TempDir()
	initGitRepo(t, repoRoot)

	if _, stderr, err := executeCommandInDir(t, env, repoRoot, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}

	nestedDir := filepath.Join(repoRoot, "nested", "app")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", nestedDir, err)
	}

	stdout, stderr, err := executeCommandInDirWithInput(t, env, nestedDir, strings.NewReader(""), "init")
	if err != nil {
		t.Fatalf("skills init error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "manifest already exists: "+filepath.Join(resolvedPath(t, repoRoot), ".agents", "manifest.yaml")) {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestProjectSyncCreatesCanonicalAndClaudeLinks(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	commit := gitOutput(t, remote, "rev-parse", "HEAD")

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "sync")
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

	wantTarget := filepath.Join(resolvedProjectDir, ".agents", "cache", "worktrees", projectID, "repo-one", commit, "analytics")
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

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "status")
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
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
		"lint/SKILL.md":      "# lint",
	})

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics", "lint"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	if !strings.Contains(statusOut, filepath.Join(resolvedProjectDir, ".agents", "skills", "lint")) {
		t.Fatalf("status output missing stale lint canonical path:\n%s", statusOut)
	}
	if !strings.Contains(statusOut, filepath.Join(resolvedProjectDir, ".claude", "skills", "lint")) {
		t.Fatalf("status output missing stale lint claude path:\n%s", statusOut)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "sync")
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
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	commitOne := gitOutput(t, remote, "rev-parse", "HEAD")

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	mustWriteFile(t, filepath.Join(remote, "README.md"), "next\n")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "advance main")
	commitTwo := gitOutput(t, remote, "rev-parse", "HEAD")

	canonicalPath := filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics")
	claudePath := filepath.Join(resolvedProjectDir, ".claude", "skills", "analytics")

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "up-to-date", "main", commitOne[:12])
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", canonicalPath)
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", claudePath)

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "update", "--dry-run")
	if err != nil {
		t.Fatalf("project update --dry-run error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Fatalf("stdout missing dry-run marker:\n%s", stdout)
	}
	assertOutputHasFields(t, stdout, "repo-one", "updated", "main", commitTwo[:12])

	statusOut, statusErr, err = executeCommandInDir(t, env, projectDir, "status")
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

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "update")
	if err != nil {
		t.Fatalf("project update error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "repo-one", "updated", "main", commitTwo[:12])

	statusOut, statusErr, err = executeCommandInDir(t, env, projectDir, "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "up-to-date", "main", commitTwo[:12])
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "stale", canonicalPath)
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", claudePath)

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "sync", "--dry-run")
	if err != nil {
		t.Fatalf("project sync --dry-run error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Fatalf("stdout missing dry-run marker:\n%s", stdout)
	}
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "would-update", canonicalPath)
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "linked", claudePath)

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "sync")
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

func TestProjectSyncUpdateAdoptsNewSkillInOneStep(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	commitOne := gitOutput(t, remote, "rev-parse", "HEAD")

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
		t.Fatalf("initial project sync error = %v, stderr = %s", err, stderr)
	}

	mustWriteFile(t, filepath.Join(remote, "lint", "SKILL.md"), "# lint")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "add lint")
	commitTwo := gitOutput(t, remote, "rev-parse", "HEAD")

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics", "lint"}))

	_, stderr, err := executeCommandInDir(t, env, projectDir, "sync")
	if err == nil {
		t.Fatalf("expected sync error, stderr = %s", stderr)
	}
	if !strings.Contains(err.Error(), "repo-one/lint: missing-skill") {
		t.Fatalf("unexpected sync error: %v", err)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "sync", "--update", "--dry-run")
	if err != nil {
		t.Fatalf("project sync --update --dry-run error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Fatalf("stdout missing dry-run marker:\n%s", stdout)
	}
	assertOutputHasFields(t, stdout, "repo-one", "updated", "main", commitTwo[:12])
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "would-update", filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics"))
	assertOutputHasFields(t, stdout, "repo-one", "lint", "would-create", filepath.Join(resolvedProjectDir, ".agents", "skills", "lint"))

	statusOut, statusErr, err := executeCommandInDir(t, env, projectDir, "status")
	if err != nil {
		t.Fatalf("project status after dry-run error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "update-available", "main", commitTwo[:12])
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics"))
	assertOutputHasFields(t, statusOut, "repo-one", "lint", "missing-skill", filepath.Join(resolvedProjectDir, ".agents", "skills", "lint"))

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "sync", "--update")
	if err != nil {
		t.Fatalf("project sync --update error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "repo-one", "updated", "main", commitTwo[:12])
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "updated", filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics"))
	assertOutputHasFields(t, stdout, "repo-one", "lint", "created", filepath.Join(resolvedProjectDir, ".agents", "skills", "lint"))

	statusOut, statusErr, err = executeCommandInDir(t, env, projectDir, "status")
	if err != nil {
		t.Fatalf("project status error = %v, stderr = %s", err, statusErr)
	}
	assertOutputHasFields(t, statusOut, "repo-one", "up-to-date", "main", commitTwo[:12])
	assertOutputHasFields(t, statusOut, "repo-one", "analytics", "linked", filepath.Join(resolvedProjectDir, ".agents", "skills", "analytics"))
	assertOutputHasFields(t, statusOut, "repo-one", "lint", "linked", filepath.Join(resolvedProjectDir, ".agents", "skills", "lint"))

	if commitOne == commitTwo {
		t.Fatal("expected distinct commits after adding lint")
	}
}

func TestProjectStatusReportsInspectFailure(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	commit := gitOutput(t, remote, "rev-parse", "HEAD")

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("project init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "sync"); err != nil {
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

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "status")
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

	stdout, stderr, err := executeCommand(t, env, "init", "--global")
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

	stdout, stderr, err = executeCommand(t, env, "sync", "--global")
	if err != nil {
		t.Fatalf("home sync error = %v, stderr = %s", err, stderr)
	}

	assertOutputHasFields(t, stdout, "repo-one", "analytics", "created", filepath.Join(env.home, ".agents", "skills", "analytics"))
	assertOutputHasFields(t, stdout, "repo-one", "analytics", "created", filepath.Join(env.home, ".claude", "skills", "analytics"))
}

func TestStatusCommandPropagatesRootContext(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stdout, stderr, err := executeCommandInDirWithContext(t, env, projectDir, ctx, nil, "status")
	if err == nil {
		t.Fatalf("expected status error, stdout = %s, stderr = %s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExitCodeForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "success", err: nil, want: exitCodeSuccess},
		{name: "doctor", err: errDoctorFoundProblems, want: exitCodeDoctor},
		{name: "help", err: pflag.ErrHelp, want: exitCodeSuccess},
		{name: "usage", err: markUsage(errors.New("bad input")), want: exitCodeUsage},
		{name: "runtime", err: errors.New("boom"), want: exitCodeFailure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := exitCodeForError(tc.err); got != tc.want {
				t.Fatalf("exitCodeForError(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func TestExitCodePolicyForUsageError(t *testing.T) {
	env := newTestEnv(t)

	_, _, err := executeCommand(t, env, "status", "--unknown-flag")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if got := exitCodeForError(err); got != exitCodeUsage {
		t.Fatalf("exitCodeForError() = %d, want %d (err=%v)", got, exitCodeUsage, err)
	}
}

func TestExitCodePolicyForMarkedUsageError(t *testing.T) {
	err := markUsage(errors.New("invalid cache mode"))
	if got := exitCodeForError(err); got != exitCodeUsage {
		t.Fatalf("exitCodeForError() = %d, want %d (err=%v)", got, exitCodeUsage, err)
	}
}

func TestExitCodePolicyForDoctorProblems(t *testing.T) {
	env := newTestEnv(t)

	configPath := filepath.Join(env.configHome, "skills", "config.yaml")
	mustWriteFile(t, configPath, "sources: [\n")

	_, _, err := executeCommand(t, env, "doctor", "--global")
	if err == nil {
		t.Fatal("expected doctor error")
	}
	if !errors.Is(err, errDoctorFoundProblems) {
		t.Fatalf("expected errDoctorFoundProblems, got %v", err)
	}
	if got := exitCodeForError(err); got != exitCodeDoctor {
		t.Fatalf("exitCodeForError() = %d, want %d", got, exitCodeDoctor)
	}
}

func TestExitCodePolicyForRuntimeFailure(t *testing.T) {
	env := newTestEnv(t)
	projectDir := t.TempDir()

	_, _, err := executeCommandInDir(t, env, projectDir, "status")
	if err == nil {
		t.Fatal("expected runtime failure")
	}
	if got := exitCodeForError(err); got != exitCodeFailure {
		t.Fatalf("exitCodeForError() = %d, want %d (err=%v)", got, exitCodeFailure, err)
	}
}

func TestBinaryExitCodes(t *testing.T) {
	requireGit(t)

	cases := []struct {
		name       string
		setup      func(t *testing.T, env testEnv) string
		args       []string
		wantCode   int
		wantStderr string
	}{
		{
			name: "help",
			setup: func(t *testing.T, env testEnv) string {
				return ""
			},
			args:     []string{"--help"},
			wantCode: exitCodeSuccess,
		},
		{
			name: "unknown flag",
			setup: func(t *testing.T, env testEnv) string {
				return ""
			},
			args:       []string{"status", "--unknown-flag"},
			wantCode:   exitCodeUsage,
			wantStderr: "unknown flag",
		},
		{
			name: "invalid cache mode",
			setup: func(t *testing.T, env testEnv) string {
				projectDir := t.TempDir()
				initGitRepo(t, projectDir)
				return projectDir
			},
			args:       []string{"init", "--cache=wat"},
			wantCode:   exitCodeUsage,
			wantStderr: "invalid cache mode",
		},
		{
			name: "doctor problems",
			setup: func(t *testing.T, env testEnv) string {
				configPath := filepath.Join(env.configHome, "skills", "config.yaml")
				mustWriteFile(t, configPath, "sources: [\n")
				return ""
			},
			args:       []string{"doctor", "--global"},
			wantCode:   exitCodeDoctor,
			wantStderr: "doctor found problems",
		},
		{
			name: "runtime failure",
			setup: func(t *testing.T, env testEnv) string {
				return t.TempDir()
			},
			args:       []string{"status"},
			wantCode:   exitCodeFailure,
			wantStderr: "outside a Git repo",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			bin := buildTestBinary(t)
			dir := tc.setup(t, env)
			stdout, stderr, code := runBuiltBinary(t, bin, env, dir, tc.args...)
			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", code, tc.wantCode, stdout, stderr)
			}
			if tc.wantStderr != "" && !strings.Contains(stderr, tc.wantStderr) {
				t.Fatalf("stderr = %q, want substring %q", stderr, tc.wantStderr)
			}
		})
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
	return executeCommandInDirWithContext(t, env, "", context.Background(), nil, args...)
}

func executeCommandInDir(t *testing.T, env testEnv, dir string, args ...string) (string, string, error) {
	return executeCommandInDirWithContext(t, env, dir, context.Background(), nil, args...)
}

func executeCommandInDirWithInput(t *testing.T, env testEnv, dir string, input io.Reader, args ...string) (string, string, error) {
	return executeCommandInDirWithContext(t, env, dir, context.Background(), input, args...)
}

func executeCommandInDirWithContext(t *testing.T, env testEnv, dir string, ctx context.Context, input io.Reader, args ...string) (string, string, error) {
	t.Helper()

	cmd := newRootCommand()
	cmd.SetArgs(args)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if input != nil {
		cmd.SetIn(input)
	}

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

	err = cmd.ExecuteContext(ctx)
	return stdout.String(), stderr.String(), err
}

func buildTestBinary(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	bin := filepath.Join(t.TempDir(), "skills-test-bin")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = wd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(output))
	}
	return bin
}

func runBuiltBinary(t *testing.T, bin string, env testEnv, dir string, args ...string) (string, string, int) {
	t.Helper()

	cmd := exec.Command(bin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(),
		"HOME="+env.home,
		"SKILLS_CONFIG_HOME="+env.configHome,
		"SKILLS_DATA_HOME="+env.dataHome,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), exitCodeSuccess
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("run binary error = %v", err)
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		t.Fatalf("unexpected exit status type %T", exitErr.Sys())
	}
	return stdout.String(), stderr.String(), status.ExitStatus()
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
	initGitRepo(t, repo)

	for path, contents := range files {
		mustWriteFile(t, filepath.Join(repo, path), contents)
	}

	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func initGitRepo(t *testing.T, repo string) {
	t.Helper()

	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Codex Test")
	runGit(t, repo, "config", "user.email", "codex@example.com")
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
