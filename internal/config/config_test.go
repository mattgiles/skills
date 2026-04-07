package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingConfigUsesDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RepoRoot != defaultRepoRootValue() {
		t.Fatalf("RepoRoot = %q, want %q", cfg.RepoRoot, defaultRepoRootValue())
	}
	if cfg.WorktreeRoot != defaultWorktreeRootValue() {
		t.Fatalf("WorktreeRoot = %q, want %q", cfg.WorktreeRoot, defaultWorktreeRootValue())
	}
	if cfg.SharedSkillsDir != DefaultConfig().SharedSkillsDir {
		t.Fatalf("SharedSkillsDir = %q, want %q", cfg.SharedSkillsDir, DefaultConfig().SharedSkillsDir)
	}
	if cfg.SharedClaudeSkillsDir != DefaultConfig().SharedClaudeSkillsDir {
		t.Fatalf("SharedClaudeSkillsDir = %q, want %q", cfg.SharedClaudeSkillsDir, DefaultConfig().SharedClaudeSkillsDir)
	}
	if len(cfg.Sources) != 0 {
		t.Fatalf("Sources length = %d, want 0", len(cfg.Sources))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	path := filepath.Join(t.TempDir(), "config.yaml")
	want := Config{
		RepoRoot:              "~/custom/repos",
		WorktreeRoot:          "~/custom/worktrees",
		SharedSkillsDir:       "~/shared/.agents/skills",
		SharedClaudeSkillsDir: "~/shared/.claude/skills",
		Sources: map[string]SourceConfig{
			"dbt-agent-skills": {URL: "git@github.com:dbt-labs/dbt-agent-skills.git"},
			"sample":           {URL: "https://github.com/example/sample.git"},
		},
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.RepoRoot != want.RepoRoot {
		t.Fatalf("RepoRoot = %q, want %q", got.RepoRoot, want.RepoRoot)
	}
	if got.WorktreeRoot != want.WorktreeRoot {
		t.Fatalf("WorktreeRoot = %q, want %q", got.WorktreeRoot, want.WorktreeRoot)
	}
	if got.SharedSkillsDir != want.SharedSkillsDir {
		t.Fatalf("SharedSkillsDir = %q, want %q", got.SharedSkillsDir, want.SharedSkillsDir)
	}
	if got.SharedClaudeSkillsDir != want.SharedClaudeSkillsDir {
		t.Fatalf("SharedClaudeSkillsDir = %q, want %q", got.SharedClaudeSkillsDir, want.SharedClaudeSkillsDir)
	}
	if len(got.Sources) != len(want.Sources) {
		t.Fatalf("Sources length = %d, want %d", len(got.Sources), len(want.Sources))
	}
	for alias, wantSource := range want.Sources {
		gotSource, ok := got.Sources[alias]
		if !ok {
			t.Fatalf("missing source %q", alias)
		}
		if gotSource.URL != wantSource.URL {
			t.Fatalf("source %q URL = %q, want %q", alias, gotSource.URL, wantSource.URL)
		}
	}
}

func TestValidateAlias(t *testing.T) {
	valid := []string{"abc", "dbt-agent-skills", "source_1", "1source"}
	for _, alias := range valid {
		if err := ValidateAlias(alias); err != nil {
			t.Fatalf("ValidateAlias(%q) unexpected error = %v", alias, err)
		}
	}

	invalid := []string{"", "Upper", "white space", "../bad", "bad/slash", "-leading-dash"}
	for _, alias := range invalid {
		if err := ValidateAlias(alias); err == nil {
			t.Fatalf("ValidateAlias(%q) expected error", alias)
		}
	}
}

func TestRepoRootPathExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := RepoRootPath(Config{RepoRoot: "~/skills/repos"})
	if err != nil {
		t.Fatalf("RepoRootPath() error = %v", err)
	}

	want := filepath.Join(home, "skills", "repos")
	if got != want {
		t.Fatalf("RepoRootPath() = %q, want %q", got, want)
	}
}

func TestDefaultConfigPathUsesSkillsConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("SKILLS_CONFIG_HOME", configHome)
	t.Setenv("HOME", t.TempDir())

	got, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}

	want := filepath.Join(configHome, "skills", "config.yaml")
	if got != want {
		t.Fatalf("DefaultConfigPath() = %q, want %q", got, want)
	}
}

func TestDefaultConfigUsesSkillsDataHome(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("SKILLS_DATA_HOME", dataHome)

	cfg := DefaultConfig()
	want := filepath.Join(dataHome, "skills", "repos")
	if cfg.RepoRoot != want {
		t.Fatalf("DefaultConfig().RepoRoot = %q, want %q", cfg.RepoRoot, want)
	}
	worktreeWant := filepath.Join(dataHome, "skills", "worktrees")
	if cfg.WorktreeRoot != worktreeWant {
		t.Fatalf("DefaultConfig().WorktreeRoot = %q, want %q", cfg.WorktreeRoot, worktreeWant)
	}
}

func TestSharedSkillsDirPathExpandsDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := SharedSkillsDirPath(Config{})
	if err != nil {
		t.Fatalf("SharedSkillsDirPath() error = %v", err)
	}

	want := filepath.Join(home, ".agents", "skills")
	if got != want {
		t.Fatalf("SharedSkillsDirPath() = %q, want %q", got, want)
	}
}

func TestSharedClaudeSkillsDirPathExpandsDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := SharedClaudeSkillsDirPath(Config{})
	if err != nil {
		t.Fatalf("SharedClaudeSkillsDirPath() error = %v", err)
	}

	want := filepath.Join(home, ".claude", "skills")
	if got != want {
		t.Fatalf("SharedClaudeSkillsDirPath() = %q, want %q", got, want)
	}
}

func TestExpandPathRelativeBecomesAbsolute(t *testing.T) {
	wd := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	got, err := ExpandPath("relative/path")
	if err != nil {
		t.Fatalf("ExpandPath() error = %v", err)
	}

	resolvedWD, err := filepath.EvalSymlinks(wd)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks(%q) error = %v", wd, err)
	}

	want := filepath.Join(resolvedWD, "relative", "path")
	if got != want {
		t.Fatalf("ExpandPath() = %q, want %q", got, want)
	}
}
