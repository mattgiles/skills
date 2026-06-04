package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattgiles/skills/internal/project"
)

func writeProjectFragment(t *testing.T, projectDir string, name string, contents string) {
	t.Helper()
	dir := project.FragmentDirPath(projectDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", dir, err)
	}
	mustWriteFile(t, filepath.Join(dir, name), contents)
}

func TestCheckProjectReportsMalformedFragment(t *testing.T) {
	requireGit(t)
	_ = newDoctorTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	if _, err := project.InitProject(context.Background(), projectDir, project.InitProjectOptions{CacheMode: project.CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))
	writeProjectFragment(t, projectDir, "broken.yaml", "sources: [\n")

	report, err := Check(context.Background(), projectDir, ScopeProject)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !report.HasErrors() {
		t.Fatal("expected malformed fragment to produce errors")
	}
	assertFindingCode(t, report.Findings, SectionWorkspace, "fragment-parse-failed")
}

func TestCheckProjectReportsFragmentMergeConflict(t *testing.T) {
	requireGit(t)
	_ = newDoctorTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	if _, err := project.InitProject(context.Background(), projectDir, project.InitProjectOptions{CacheMode: project.CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))
	// Fragment redeclares the repo-one source alias already in the main manifest.
	writeProjectFragment(t, projectDir, "perk.yaml", "sources:\n  repo-one:\n    url: "+remote+"\n    ref: main\nskills: []\n")

	report, err := Check(context.Background(), projectDir, ScopeProject)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !report.HasErrors() {
		t.Fatal("expected fragment merge conflict to produce errors")
	}
	assertFindingCode(t, report.Findings, SectionWorkspace, "fragment-merge-conflict")
}
