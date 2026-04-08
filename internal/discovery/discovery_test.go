package discovery

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFindsSkillsAndIgnoresGit(t *testing.T) {
	repo := t.TempDir()

	mustWriteFile(t, filepath.Join(repo, "analytics", "SKILL.md"), "# analytics")
	mustWriteFile(t, filepath.Join(repo, "nested", "dbt", "SKILL.md"), "# dbt")
	mustWriteFile(t, filepath.Join(repo, ".git", "ignored", "SKILL.md"), "# ignored")

	skills, err := Discover("demo", repo)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("len(skills) = %d, want 2", len(skills))
	}

	got := map[string]string{}
	for _, skill := range skills {
		got[skill.Name] = skill.RelativePath
	}

	if got["analytics"] != "analytics" {
		t.Fatalf("analytics path = %q, want %q", got["analytics"], "analytics")
	}
	if got["dbt"] != filepath.Join("nested", "dbt") {
		t.Fatalf("dbt path = %q, want %q", got["dbt"], filepath.Join("nested", "dbt"))
	}
}

func TestDiscoverUsesTrackedGitFiles(t *testing.T) {
	requireGit(t)

	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Codex Test")
	runGit(t, repo, "config", "user.email", "codex@example.com")

	mustWriteFile(t, filepath.Join(repo, "analytics", "SKILL.md"), "# analytics")
	mustWriteFile(t, filepath.Join(repo, "nested", "dbt", "SKILL.md"), "# dbt")
	mustWriteFile(t, filepath.Join(repo, "ignored", "SKILL.md"), "# ignored")
	runGit(t, repo, "add", "analytics/SKILL.md", "nested/dbt/SKILL.md")
	runGit(t, repo, "commit", "-m", "initial")

	skills, err := Discover("demo", repo)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	got := map[string]string{}
	for _, skill := range skills {
		got[skill.Name] = skill.RelativePath
	}

	if len(skills) != 2 {
		t.Fatalf("len(skills) = %d, want 2", len(skills))
	}
	if got["analytics"] != "analytics" {
		t.Fatalf("analytics path = %q, want %q", got["analytics"], "analytics")
	}
	if got["dbt"] != filepath.Join("nested", "dbt") {
		t.Fatalf("dbt path = %q, want %q", got["dbt"], filepath.Join("nested", "dbt"))
	}
	if _, ok := got["ignored"]; ok {
		t.Fatalf("unexpected untracked skill discovered: %+v", got)
	}
}

func TestDiscoverAllowsEmptyRepos(t *testing.T) {
	skills, err := Discover("demo", t.TempDir())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 0 {
		t.Fatalf("len(skills) = %d, want 0", len(skills))
	}
}

func TestDiscoverFiltersToProvidedGitSubdirectory(t *testing.T) {
	requireGit(t)

	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Codex Test")
	runGit(t, repo, "config", "user.email", "codex@example.com")

	mustWriteFile(t, filepath.Join(repo, "skills", "analytics", "SKILL.md"), "# analytics")
	mustWriteFile(t, filepath.Join(repo, "other", "lint", "SKILL.md"), "# lint")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")

	skills, err := Discover("demo", filepath.Join(repo, "skills"))
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("len(skills) = %d, want 1", len(skills))
	}
	if skills[0].Name != "analytics" {
		t.Fatalf("skill name = %q, want analytics", skills[0].Name)
	}
	if skills[0].RelativePath != "analytics" {
		t.Fatalf("skill path = %q, want analytics", skills[0].RelativePath)
	}
}

func TestDiscoverUsesRepoBasenameForRootSkill(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repo := filepath.Join(root, "terraform-skill")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", repo, err)
	}
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Codex Test")
	runGit(t, repo, "config", "user.email", "codex@example.com")

	mustWriteFile(t, filepath.Join(repo, "SKILL.md"), "# terraform-skill")
	runGit(t, repo, "add", "SKILL.md")
	runGit(t, repo, "commit", "-m", "initial")

	skills, err := Discover("demo", repo)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("len(skills) = %d, want 1", len(skills))
	}
	if skills[0].Name != "terraform-skill" {
		t.Fatalf("skill name = %q, want terraform-skill", skills[0].Name)
	}
	if skills[0].RelativePath != "." {
		t.Fatalf("skill path = %q, want .", skills[0].RelativePath)
	}
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

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
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
