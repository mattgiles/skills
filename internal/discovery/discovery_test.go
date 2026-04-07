package discovery

import (
	"os"
	"path/filepath"
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

func TestDiscoverAllowsEmptyRepos(t *testing.T) {
	skills, err := Discover("demo", t.TempDir())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(skills) != 0 {
		t.Fatalf("len(skills) = %d, want 0", len(skills))
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
