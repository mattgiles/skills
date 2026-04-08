package doctor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/project"
)

func TestCheckProjectHealthyWorkspace(t *testing.T) {
	requireGit(t)
	_ = newDoctorTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})

	if _, err := project.InitProject(projectDir, project.InitProjectOptions{CacheMode: project.CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))
	if _, err := project.Sync(context.Background(), projectDir, project.SyncOptions{}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	report, err := Check(context.Background(), projectDir, ScopeProject)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if report.Scope != ScopeProject {
		t.Fatalf("report.Scope = %q, want %q", report.Scope, ScopeProject)
	}
	if report.Target != projectDir {
		t.Fatalf("report.Target = %q, want %q", report.Target, projectDir)
	}
	if report.HasErrors() || report.ErrorCount() != 0 || report.WarningCount() != 0 {
		t.Fatalf("unexpected report counts: errors=%d warnings=%d findings=%+v", report.ErrorCount(), report.WarningCount(), report.Findings)
	}
	assertFindingCode(t, report.Findings, SectionConfig, "project-cache-mode")
	assertFindingCode(t, report.Findings, SectionConfig, "config-not-required")
}

func TestCheckProjectMissingManifest(t *testing.T) {
	requireGit(t)
	_ = newDoctorTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	report, err := Check(context.Background(), projectDir, ScopeProject)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !report.HasErrors() {
		t.Fatal("expected project report to have errors")
	}
	assertFindingCode(t, report.Findings, SectionWorkspace, "manifest-missing")
	assertFindingCode(t, report.Findings, SectionSources, "not-checked")
	assertHint(t, report.Hints(), "run skills init")
}

func TestCheckProjectMalformedLocalConfig(t *testing.T) {
	requireGit(t)
	_ = newDoctorTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	if _, err := project.InitProject(projectDir, project.InitProjectOptions{CacheMode: project.CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	mustWriteFile(t, filepath.Join(projectDir, ".agents", "local.yaml"), "cache: [\n")

	report, err := Check(context.Background(), projectDir, ScopeProject)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !report.HasErrors() {
		t.Fatal("expected malformed local config to produce errors")
	}
	assertFindingCode(t, report.Findings, SectionConfig, "local-config-parse-failed")
}

func TestCheckProjectWarnsAboutStaleManagedLinks(t *testing.T) {
	requireGit(t)
	_ = newDoctorTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	remote := initRemoteRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
		"lint/SKILL.md":      "# lint",
	})

	if _, err := project.InitProject(projectDir, project.InitProjectOptions{CacheMode: project.CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics", "lint"}))
	if _, err := project.Sync(context.Background(), projectDir, project.SyncOptions{}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"analytics"}))

	report, err := Check(context.Background(), projectDir, ScopeProject)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if report.WarningCount() < 2 {
		t.Fatalf("expected stale-link warnings, got %d findings=%+v", report.WarningCount(), report.Findings)
	}
	assertFindingCode(t, report.Findings, SectionSkills, "stale-managed-link")
	assertFindingCode(t, report.Findings, SectionClaude, "stale-managed-link")
	assertHint(t, report.Hints(), "run skills sync")
}

func TestCheckProjectLocalModeIgnoresBrokenGlobalConfig(t *testing.T) {
	requireGit(t)
	env := newDoctorTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	if _, err := project.InitProject(projectDir, project.InitProjectOptions{CacheMode: project.CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}
	mustWriteFile(t, configPath, "sources: [\n")

	report, err := Check(context.Background(), projectDir, ScopeProject)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	assertFindingCode(t, report.Findings, SectionConfig, "project-cache-mode")
	assertFindingCode(t, report.Findings, SectionConfig, "config-not-required")
	assertNoFindingCode(t, report.Findings, SectionConfig, "config-parse-failed")
	_ = env
}

func TestCheckGlobalMissingHomeManifestIsWarning(t *testing.T) {
	requireGit(t)
	env := newDoctorTestEnv(t)

	cfg := config.DefaultConfig()
	if _, err := project.InitHome(cfg); err != nil {
		t.Fatalf("InitHome() error = %v", err)
	}
	manifestPath, err := project.HomeManifestPath(cfg)
	if err != nil {
		t.Fatalf("HomeManifestPath() error = %v", err)
	}
	if err := os.Remove(manifestPath); err != nil {
		t.Fatalf("Remove(%q) error = %v", manifestPath, err)
	}

	report, err := Check(context.Background(), env.home, ScopeGlobal)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if report.HasErrors() {
		t.Fatalf("expected warning-only report, got %+v", report.Findings)
	}
	assertFindingCode(t, report.Findings, SectionWorkspace, "manifest-missing")
	assertHint(t, report.Hints(), "run skills init --global")
}

func TestCheckGlobalMalformedConfigIsError(t *testing.T) {
	requireGit(t)
	env := newDoctorTestEnv(t)

	configPath, err := config.DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}
	mustWriteFile(t, configPath, "sources: [\n")

	report, err := Check(context.Background(), env.home, ScopeGlobal)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !report.HasErrors() {
		t.Fatal("expected malformed global config to produce errors")
	}
	assertFindingCode(t, report.Findings, SectionConfig, "config-parse-failed")
}

func TestCheckUnsupportedScopeReturnsError(t *testing.T) {
	report, err := Check(context.Background(), t.TempDir(), Scope("weird"))
	if err == nil {
		t.Fatalf("expected unsupported scope error, report=%+v", report)
	}
}

type doctorTestEnv struct {
	configHome string
	dataHome   string
	home       string
}

func newDoctorTestEnv(t *testing.T) doctorTestEnv {
	t.Helper()

	root := t.TempDir()
	env := doctorTestEnv{
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
	for _, name := range skills {
		lines = append(lines,
			"  - source: repo-one",
			"    name: "+name,
		)
	}
	return strings.Join(lines, "\n") + "\n"
}

func assertFindingCode(t *testing.T, findings []Finding, section string, code string) {
	t.Helper()

	for _, finding := range findings {
		if finding.Section == section && finding.Code == code {
			return
		}
	}
	t.Fatalf("finding %s/%s not found in %+v", section, code, findings)
}

func assertNoFindingCode(t *testing.T, findings []Finding, section string, code string) {
	t.Helper()

	for _, finding := range findings {
		if finding.Section == section && finding.Code == code {
			t.Fatalf("unexpected finding %s/%s in %+v", section, code, findings)
		}
	}
}

func assertHint(t *testing.T, hints []string, want string) {
	t.Helper()

	for _, hint := range hints {
		if hint == want {
			return
		}
	}
	t.Fatalf("hint %q not found in %+v", want, hints)
}

func resolvedPath(t *testing.T, path string) string {
	t.Helper()

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", path, err)
	}
	return resolved
}
