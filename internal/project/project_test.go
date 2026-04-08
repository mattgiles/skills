package project

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattgiles/skills/internal/config"
)

func TestValidateManifestRejectsDuplicateSkills(t *testing.T) {
	manifest := Manifest{
		Sources: map[string]ManifestSource{
			"repo-one": {Ref: "main"},
		},
		Skills: []ManifestSkill{
			{Source: "repo-one", Name: "analytics"},
			{Source: "repo-one", Name: "analytics"},
		},
	}

	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("ValidateManifest() expected duplicate skill error")
	}
}

func TestProjectIDIsStable(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())

	first, err := ProjectID(projectDir)
	if err != nil {
		t.Fatalf("ProjectID() error = %v", err)
	}
	second, err := ProjectID(projectDir)
	if err != nil {
		t.Fatalf("ProjectID() error = %v", err)
	}

	if first != second {
		t.Fatalf("ProjectID() mismatch: %q vs %q", first, second)
	}
}

func TestStateRoundTripIncludesSourceState(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())

	state := State{
		Sources: []SourceState{
			{Source: "repo-one", Ref: "main", ResolvedCommit: "abc123"},
		},
		SkillLinks: []ManagedLink{
			{Path: "/tmp/skill", Target: "/tmp/target", Source: "repo-one", Skill: "analytics"},
		},
		ClaudeLinks: []ManagedLink{
			{Path: "/tmp/claude", Target: "/tmp/skill", Source: "repo-one", Skill: "analytics"},
		},
	}

	if err := SaveState(projectDir, state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	loaded, err := LoadState(projectDir)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	if len(loaded.Sources) != 1 {
		t.Fatalf("len(loaded.Sources) = %d, want 1", len(loaded.Sources))
	}
	if loaded.Sources[0].ResolvedCommit != "abc123" {
		t.Fatalf("ResolvedCommit = %q, want %q", loaded.Sources[0].ResolvedCommit, "abc123")
	}
	if len(loaded.SkillLinks) != 1 {
		t.Fatalf("len(loaded.SkillLinks) = %d, want 1", len(loaded.SkillLinks))
	}
	if len(loaded.ClaudeLinks) != 1 {
		t.Fatalf("len(loaded.ClaudeLinks) = %d, want 1", len(loaded.ClaudeLinks))
	}
	if got := StatePath(projectDir); got != filepath.Join(projectDir, ".agents", "state.yaml") {
		t.Fatalf("StatePath() = %q", got)
	}
	if got := LocalConfigPath(projectDir); got != filepath.Join(projectDir, ".agents", "local.yaml") {
		t.Fatalf("LocalConfigPath() = %q", got)
	}
}

func TestLocalConfigRoundTripDefaultsToLocal(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())

	loaded, err := LoadLocalConfig(projectDir)
	if err != nil {
		t.Fatalf("LoadLocalConfig() error = %v", err)
	}
	if loaded.Mode != CacheModeLocal || !loaded.Implicit || loaded.Exists {
		t.Fatalf("unexpected default local config: %+v", loaded)
	}

	if err := SaveLocalConfig(projectDir, LocalConfig{
		Cache: LocalCacheConfig{Mode: CacheModeGlobal},
	}); err != nil {
		t.Fatalf("SaveLocalConfig() error = %v", err)
	}

	loaded, err = LoadLocalConfig(projectDir)
	if err != nil {
		t.Fatalf("LoadLocalConfig() error = %v", err)
	}
	if loaded.Mode != CacheModeGlobal || !loaded.Exists || loaded.Implicit {
		t.Fatalf("unexpected saved local config: %+v", loaded)
	}
}

func TestLoadManifestRejectsUnknownKey(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())
	path := ManifestPath(projectDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("sources: {}\nskills: []\nextra: true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadManifest(projectDir)
	if err == nil {
		t.Fatal("LoadManifest() expected unknown key error")
	}
	if !strings.Contains(err.Error(), `unknown field "extra"`) {
		t.Fatalf("LoadManifest() error = %v", err)
	}
}

func TestLoadLocalConfigRejectsUnknownKey(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())
	path := LocalConfigPath(projectDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("cache:\n  mode: local\n  extra: true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadLocalConfig(projectDir)
	if err == nil {
		t.Fatal("LoadLocalConfig() expected unknown key error")
	}
	if !strings.Contains(err.Error(), `unknown field "extra"`) {
		t.Fatalf("LoadLocalConfig() error = %v", err)
	}
}

func TestSaveLocalConfigPreservesComments(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())
	path := LocalConfigPath(projectDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	contents := strings.Join([]string{
		"cache:",
		"  # keep mode comment",
		"  mode: local",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := SaveLocalConfig(projectDir, LocalConfig{
		Cache: LocalCacheConfig{Mode: CacheModeGlobal},
	}); err != nil {
		t.Fatalf("SaveLocalConfig() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "# keep mode comment") {
		t.Fatalf("saved local config lost comment:\n%s", string(data))
	}

	loaded, err := LoadLocalConfig(projectDir)
	if err != nil {
		t.Fatalf("LoadLocalConfig() error = %v", err)
	}
	if loaded.Mode != CacheModeGlobal {
		t.Fatalf("LoadLocalConfig().Mode = %q, want %q", loaded.Mode, CacheModeGlobal)
	}
}

func TestProjectSyncDryRunDoesNotWriteStateOrLinks(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics"}))

	result, err := Sync(context.Background(), projectDir, SyncOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Sync() dry-run error = %v", err)
	}
	if !result.DryRun {
		t.Fatal("Sync() result should be dry-run")
	}
	assertSourceStatus(t, result.Sources, "repo-one", "not-synced")
	assertLinkStatus(t, result.SkillLinks, "repo-one", "analytics", "would-create")
	assertLinkStatus(t, result.ClaudeLinks, "repo-one", "analytics", "would-create")

	for _, path := range []string{
		StatePath(projectDir),
		filepath.Join(SkillsDir(projectDir), "analytics"),
		filepath.Join(ClaudeSkillsDir(projectDir), "analytics"),
	} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %q to be absent after dry-run, got err = %v", path, err)
		}
	}
}

func TestProjectStatusAfterSyncIsHealthy(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics"}))

	if _, err := Sync(context.Background(), projectDir, SyncOptions{}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	report, err := Status(context.Background(), projectDir)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	assertSourceStatus(t, report.Sources, "repo-one", "up-to-date")
	assertLinkStatus(t, report.SkillLinks, "repo-one", "analytics", "linked")
	assertLinkStatus(t, report.ClaudeLinks, "repo-one", "analytics", "linked")

	state, err := LoadState(projectDir)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if len(state.Sources) != 1 {
		t.Fatalf("len(state.Sources) = %d, want 1", len(state.Sources))
	}
}

func TestProjectUpdateDryRunPreservesStateAndLinks(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics"}))

	if _, err := Sync(context.Background(), projectDir, SyncOptions{}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	stateBefore, err := os.ReadFile(StatePath(projectDir))
	if err != nil {
		t.Fatalf("ReadFile(state) error = %v", err)
	}
	linkPath := filepath.Join(SkillsDir(projectDir), "analytics")
	targetBefore, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", linkPath, err)
	}

	mustWriteFile(t, filepath.Join(remote, "README.md"), "next\n")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "advance main")

	result, err := Update(context.Background(), projectDir, UpdateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Update() dry-run error = %v", err)
	}
	if !result.DryRun {
		t.Fatal("Update() result should be dry-run")
	}
	assertSourceStatus(t, result.Sources, "repo-one", "updated")

	stateAfter, err := os.ReadFile(StatePath(projectDir))
	if err != nil {
		t.Fatalf("ReadFile(state) error = %v", err)
	}
	if string(stateAfter) != string(stateBefore) {
		t.Fatalf("state changed during dry-run\nbefore:\n%s\nafter:\n%s", string(stateBefore), string(stateAfter))
	}
	targetAfter, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", linkPath, err)
	}
	if targetAfter != targetBefore {
		t.Fatalf("link target changed during dry-run: %q -> %q", targetBefore, targetAfter)
	}
}

func TestProjectUpdateWithoutSyncLeavesStaleLinks(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics"}))

	if _, err := Sync(context.Background(), projectDir, SyncOptions{}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	mustWriteFile(t, filepath.Join(remote, "README.md"), "next\n")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "advance main")

	result, err := Update(context.Background(), projectDir, UpdateOptions{})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	assertSourceStatus(t, result.Sources, "repo-one", "updated")

	report, err := Status(context.Background(), projectDir)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	assertSourceStatus(t, report.Sources, "repo-one", "up-to-date")
	assertLinkStatus(t, report.SkillLinks, "repo-one", "analytics", "stale")
	assertLinkStatus(t, report.ClaudeLinks, "repo-one", "analytics", "linked")
}

func TestProjectSyncWithoutUpdateKeepsStoredCommit(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics"}))

	if _, err := Sync(context.Background(), projectDir, SyncOptions{}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	mustWriteFile(t, filepath.Join(remote, "lint", "SKILL.md"), "# lint")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "add lint")

	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics", "lint"}))

	_, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err == nil {
		t.Fatal("Sync() expected missing-skill error without update")
	}
	if !strings.Contains(err.Error(), "repo-one/lint: missing-skill") {
		t.Fatalf("Sync() error = %v", err)
	}
}

func TestProjectSyncWithUpdateAdvancesStateAndInstallsNewSkill(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics"}))

	if _, err := Sync(context.Background(), projectDir, SyncOptions{}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	stateBefore, err := os.ReadFile(StatePath(projectDir))
	if err != nil {
		t.Fatalf("ReadFile(state) error = %v", err)
	}

	mustWriteFile(t, filepath.Join(remote, "lint", "SKILL.md"), "# lint")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "add lint")
	commitTwo := gitOutput(t, remote, "rev-parse", "HEAD")

	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics", "lint"}))

	result, err := Sync(context.Background(), projectDir, SyncOptions{
		DryRun: true,
		Update: true,
	})
	if err != nil {
		t.Fatalf("Sync() dry-run error = %v", err)
	}
	if !result.DryRun {
		t.Fatal("Sync() result should be dry-run")
	}
	assertSourceStatus(t, result.Sources, "repo-one", "updated")
	assertLinkStatus(t, result.SkillLinks, "repo-one", "analytics", "would-update")
	assertLinkStatus(t, result.SkillLinks, "repo-one", "lint", "would-create")

	stateAfterDryRun, err := os.ReadFile(StatePath(projectDir))
	if err != nil {
		t.Fatalf("ReadFile(state) error = %v", err)
	}
	if string(stateAfterDryRun) != string(stateBefore) {
		t.Fatalf("state changed during dry-run\nbefore:\n%s\nafter:\n%s", string(stateBefore), string(stateAfterDryRun))
	}

	result, err = Sync(context.Background(), projectDir, SyncOptions{Update: true})
	if err != nil {
		t.Fatalf("Sync() with update error = %v", err)
	}
	assertSourceStatus(t, result.Sources, "repo-one", "updated")
	assertLinkStatus(t, result.SkillLinks, "repo-one", "analytics", "updated")
	assertLinkStatus(t, result.SkillLinks, "repo-one", "lint", "created")

	state, err := LoadState(projectDir)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if len(state.Sources) != 1 || state.Sources[0].ResolvedCommit != commitTwo {
		t.Fatalf("unexpected state sources: %+v", state.Sources)
	}

	report, err := Status(context.Background(), projectDir)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	assertSourceStatus(t, report.Sources, "repo-one", "up-to-date")
	assertLinkStatus(t, report.SkillLinks, "repo-one", "analytics", "linked")
	assertLinkStatus(t, report.SkillLinks, "repo-one", "lint", "linked")
}

func TestInitProjectPropagatesContext(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := InitProject(ctx, projectDir, InitProjectOptions{CacheMode: CacheModeLocal})
	if err == nil {
		t.Fatal("expected InitProject() error")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectSyncPrunesStaleLinks(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
		"lint/SKILL.md":      "# lint",
	})

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics", "lint"}))

	if _, err := Sync(context.Background(), projectDir, SyncOptions{}); err != nil {
		t.Fatalf("Sync() initial error = %v", err)
	}

	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics"}))

	result, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync() prune error = %v", err)
	}
	if len(result.PrunedSkillLinks) != 1 || !strings.HasSuffix(result.PrunedSkillLinks[0], filepath.Join(".agents", "skills", "lint")) {
		t.Fatalf("unexpected pruned skill links: %+v", result.PrunedSkillLinks)
	}
	if len(result.PrunedClaudeLinks) != 1 || !strings.HasSuffix(result.PrunedClaudeLinks[0], filepath.Join(".claude", "skills", "lint")) {
		t.Fatalf("unexpected pruned claude links: %+v", result.PrunedClaudeLinks)
	}
	for _, path := range []string{
		filepath.Join(SkillsDir(projectDir), "lint"),
		filepath.Join(ClaudeSkillsDir(projectDir), "lint"),
	} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %q to be pruned, got err = %v", path, err)
		}
	}
}

func TestResolveProjectWorkspaceUsesConfiguredCacheRoots(t *testing.T) {
	requireGit(t)
	env := newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}

	localWS, err := resolveProjectWorkspace(projectDir)
	if err != nil {
		t.Fatalf("resolveProjectWorkspace() local error = %v", err)
	}
	if localWS.RepoRoot != RepoRoot(projectDir) {
		t.Fatalf("local RepoRoot = %q, want %q", localWS.RepoRoot, RepoRoot(projectDir))
	}
	if localWS.WorktreeRoot != WorktreeRoot(projectDir) {
		t.Fatalf("local WorktreeRoot = %q, want %q", localWS.WorktreeRoot, WorktreeRoot(projectDir))
	}

	cfg := config.Config{
		RepoRoot:              filepath.Join(env.dataHome, "shared-repos"),
		WorktreeRoot:          filepath.Join(env.dataHome, "shared-worktrees"),
		SharedSkillsDir:       filepath.Join(env.home, ".agents", "skills"),
		SharedClaudeSkillsDir: filepath.Join(env.home, ".claude", "skills"),
	}
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}
	if err := SaveLocalConfig(projectDir, LocalConfig{
		Cache: LocalCacheConfig{Mode: CacheModeGlobal},
	}); err != nil {
		t.Fatalf("SaveLocalConfig() error = %v", err)
	}

	globalWS, err := resolveProjectWorkspace(projectDir)
	if err != nil {
		t.Fatalf("resolveProjectWorkspace() global error = %v", err)
	}
	wantRepoRoot, _ := config.RepoRootPath(cfg)
	wantWorktreeRoot, _ := config.WorktreeRootPath(cfg)
	if globalWS.RepoRoot != wantRepoRoot {
		t.Fatalf("global RepoRoot = %q, want %q", globalWS.RepoRoot, wantRepoRoot)
	}
	if globalWS.WorktreeRoot != wantWorktreeRoot {
		t.Fatalf("global WorktreeRoot = %q, want %q", globalWS.WorktreeRoot, wantWorktreeRoot)
	}
}

func TestProjectStatusReportsRefChangeAsUpdateAvailable(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	mustWriteFile(t, filepath.Join(remote, "branch.txt"), "feature\n")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "prepare feature")
	runGit(t, remote, "branch", "feature")

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"analytics"}))

	if _, err := Sync(context.Background(), projectDir, SyncOptions{}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	writeProjectManifest(t, projectDir, manifestFor(remote, "feature", []string{"analytics"}))

	report, err := Status(context.Background(), projectDir)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	assertSourceStatus(t, report.Sources, "repo-one", "update-available")
	if report.Sources[0].Message != "state recorded for ref main" {
		t.Fatalf("unexpected source message: %q", report.Sources[0].Message)
	}
}

func TestProjectSyncSupportsRepoRootSkill(t *testing.T) {
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	root := t.TempDir()
	remote := filepath.Join(root, "terraform-skill")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", remote, err)
	}
	initGitRepo(t, remote)
	mustWriteFile(t, filepath.Join(remote, "SKILL.md"), "# terraform-skill")
	runGit(t, remote, "add", "SKILL.md")
	runGit(t, remote, "commit", "-m", "initial")

	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"skills:",
		"  - source: repo-one",
		"    name: terraform-skill",
		"",
	}, "\n"))

	result, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	assertSourceStatus(t, result.Sources, "repo-one", "resolved")
	assertLinkStatus(t, result.SkillLinks, "repo-one", "terraform-skill", "created")

	canonicalPath := filepath.Join(SkillsDir(projectDir), "terraform-skill")
	target, err := os.Readlink(canonicalPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", canonicalPath, err)
	}

	state, err := LoadState(projectDir)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if len(state.Sources) != 1 {
		t.Fatalf("len(state.Sources) = %d, want 1", len(state.Sources))
	}

	projectID, err := ProjectID(projectDir)
	if err != nil {
		t.Fatalf("ProjectID() error = %v", err)
	}
	wantTarget := filepath.Join(WorktreeRoot(projectDir), projectID, "repo-one", state.Sources[0].ResolvedCommit)
	if target != wantTarget {
		t.Fatalf("canonical target = %q, want %q", target, wantTarget)
	}
}

type projectTestEnv struct {
	configHome string
	dataHome   string
	home       string
}

func newProjectTestEnv(t *testing.T) projectTestEnv {
	t.Helper()

	root := t.TempDir()
	env := projectTestEnv{
		configHome: filepath.Join(root, "config"),
		dataHome:   filepath.Join(root, "data"),
		home:       filepath.Join(root, "home"),
	}

	t.Setenv("HOME", env.home)
	t.Setenv("SKILLS_CONFIG_HOME", env.configHome)
	t.Setenv("SKILLS_DATA_HOME", env.dataHome)
	return env
}

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initRemoteRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	repo := resolvedPath(t, t.TempDir())
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
	mustWriteFile(t, ManifestPath(projectDir), contents)
}

func manifestFor(remoteURL string, ref string, skills []string) string {
	lines := []string{
		"sources:",
		"  repo-one:",
		"    url: " + remoteURL,
		"    ref: " + ref,
		"skills:",
	}
	for _, name := range skills {
		lines = append(lines,
			"  - source: repo-one",
			"    name: "+name,
		)
	}
	return strings.Join(lines, "\n") + "\n"
}

func assertSourceStatus(t *testing.T, reports []SourceReport, alias string, want string) {
	t.Helper()

	for _, report := range reports {
		if report.Alias == alias {
			if report.Status != want {
				t.Fatalf("source %q status = %q, want %q", alias, report.Status, want)
			}
			return
		}
	}
	t.Fatalf("source %q not found in %+v", alias, reports)
}

func assertLinkStatus(t *testing.T, reports []LinkReport, sourceAlias string, skill string, want string) {
	t.Helper()

	for _, report := range reports {
		if report.Source == sourceAlias && report.Skill == skill {
			if report.Status != want {
				t.Fatalf("link %s/%s status = %q, want %q", sourceAlias, skill, report.Status, want)
			}
			return
		}
	}
	t.Fatalf("link %s/%s not found in %+v", sourceAlias, skill, reports)
}

func resolvedPath(t *testing.T, path string) string {
	t.Helper()

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", path, err)
	}
	return resolved
}
