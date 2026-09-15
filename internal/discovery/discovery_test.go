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

func TestNormalizeSkillPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"a/x", "a/x"},
		{"a/x/", "a/x"},
		{"./a/x", "a/x"},
		{"a//x", "a/x"},
		{"a/x/SKILL.md", "a/x"},
		{"SKILL.md", "."},
		{".", "."},
		{"./", "."},
		{"  a/x  ", "a/x"},
		{"a/./x/../y", "a/y"},
		{"..", ".."},
		{"../x", "../x"},
	}
	for _, tc := range cases {
		if got := NormalizeSkillPath(tc.in); got != tc.want {
			t.Errorf("NormalizeSkillPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestScopeAllows(t *testing.T) {
	cases := []struct {
		name  string
		scope Scope
		path  string
		want  bool
	}{
		{"zero scope allows everything", Scope{}, "a/x", true},
		{"zero scope allows root", Scope{}, ".", true},
		{"include only: inside", Scope{Include: []string{"skills"}}, "skills/x", true},
		{"include only: exact", Scope{Include: []string{"skills"}}, "skills", true},
		{"include only: outside", Scope{Include: []string{"skills"}}, "plugins/x", false},
		{"include only: sibling prefix does not match", Scope{Include: []string{"a"}}, "ab/x", false},
		{"include only: root not under include", Scope{Include: []string{"skills"}}, ".", false},
		{"include dot allows everything", Scope{Include: []string{"."}}, "plugins/x", true},
		{"include dot allows root", Scope{Include: []string{"."}}, ".", true},
		{"exclude only: inside", Scope{Exclude: []string{"plugins"}}, "plugins/x", false},
		{"exclude only: exact", Scope{Exclude: []string{"plugins"}}, "plugins", false},
		{"exclude only: outside", Scope{Exclude: []string{"plugins"}}, "skills/x", true},
		{"exclude only: sibling prefix does not match", Scope{Exclude: []string{"a"}}, "ab/x", true},
		{"both: included and not excluded", Scope{Include: []string{"plugins"}, Exclude: []string{"plugins/x"}}, "plugins/y/z", true},
		{"both: exclude wins", Scope{Include: []string{"plugins"}, Exclude: []string{"plugins/x"}}, "plugins/x/z", false},
		{"both: outside include", Scope{Include: []string{"plugins"}, Exclude: []string{"plugins/x"}}, "skills/z", false},
		{"normalizes prefix trailing slash", Scope{Include: []string{"skills/"}}, "skills/x", true},
		{"normalizes prefix leading dot slash", Scope{Exclude: []string{"./plugins"}}, "plugins/x", false},
		{"empty prefix ignored", Scope{Include: []string{""}}, "a/x", false},
	}
	for _, tc := range cases {
		if got := tc.scope.Allows(tc.path); got != tc.want {
			t.Errorf("%s: Scope%+v.Allows(%q) = %v, want %v", tc.name, tc.scope, tc.path, got, tc.want)
		}
	}
}

func TestScopeIsZero(t *testing.T) {
	if !(Scope{}).IsZero() {
		t.Fatal("empty Scope should be zero")
	}
	if (Scope{Include: []string{"a"}}).IsZero() {
		t.Fatal("Scope with include should not be zero")
	}
	if (Scope{Exclude: []string{"a"}}).IsZero() {
		t.Fatal("Scope with exclude should not be zero")
	}
}

func TestFilterScopePreservesOrder(t *testing.T) {
	skills := []DiscoveredSkill{
		{Name: "bedrock", RelativePath: "plugins/x/bedrock"},
		{Name: "bedrock", RelativePath: "skills/bedrock"},
		{Name: "lambda", RelativePath: "plugins/x/lambda"},
		{Name: "root", RelativePath: "."},
		{Name: "s3", RelativePath: "skills/s3"},
	}

	kept, excluded := FilterScope(skills, Scope{Exclude: []string{"plugins"}})

	wantKept := []string{"skills/bedrock", ".", "skills/s3"}
	wantExcluded := []string{"plugins/x/bedrock", "plugins/x/lambda"}
	assertRelativePaths(t, "kept", kept, wantKept)
	assertRelativePaths(t, "excluded", excluded, wantExcluded)

	kept, excluded = FilterScope(skills, Scope{Include: []string{"skills"}})
	assertRelativePaths(t, "kept", kept, []string{"skills/bedrock", "skills/s3"})
	assertRelativePaths(t, "excluded", excluded, []string{"plugins/x/bedrock", "plugins/x/lambda", "."})

	kept, excluded = FilterScope(skills, Scope{})
	if len(kept) != len(skills) || len(excluded) != 0 {
		t.Fatalf("zero scope: kept=%d excluded=%d, want %d/0", len(kept), len(excluded), len(skills))
	}

	kept, excluded = FilterScope(skills, Scope{Include: []string{"."}})
	if len(kept) != len(skills) || len(excluded) != 0 {
		t.Fatalf("include \".\": kept=%d excluded=%d, want %d/0", len(kept), len(excluded), len(skills))
	}
}

func assertRelativePaths(t *testing.T, label string, skills []DiscoveredSkill, want []string) {
	t.Helper()
	if len(skills) != len(want) {
		t.Fatalf("%s: len = %d, want %d (%+v)", label, len(skills), len(want), skills)
	}
	for i, skill := range skills {
		if skill.RelativePath != want[i] {
			t.Fatalf("%s[%d] = %q, want %q", label, i, skill.RelativePath, want[i])
		}
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
