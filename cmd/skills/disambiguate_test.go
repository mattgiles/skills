package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// duplicateSkillRemote builds a source repo that ships the same skill twice
// under different directories.
func duplicateSkillRemote(t *testing.T) string {
	t.Helper()
	return initRemoteRepo(t, map[string]string{
		"plugins/x/bedrock/SKILL.md": "# bedrock (plugin copy)",
		"skills/bedrock/SKILL.md":    "# bedrock",
		"skills/lambda/SKILL.md":     "# lambda",
	})
}

func readManifest(t *testing.T, projectDir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectDir, ".agents", "manifest.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	return string(data)
}

func TestSourceAddWritesAndCarriesForwardScope(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)
	remote := duplicateSkillRemote(t)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "source", "add", "--ref", "main", "repo-one", remote, "--exclude", "plugins")
	if err != nil {
		t.Fatalf("source add error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `registered source "repo-one" (exclude: plugins)`) {
		t.Fatalf("stdout = %q", stdout)
	}
	manifest := readManifest(t, projectDir)
	if !strings.Contains(manifest, "exclude:") || !strings.Contains(manifest, "plugins") {
		t.Fatalf("manifest missing exclude:\n%s", manifest)
	}

	// Re-running without scope flags keeps the existing exclude list.
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "source", "add", "repo-one", remote); err != nil {
		t.Fatalf("source add (re-run) error = %v, stderr = %s", err, stderr)
	}
	manifest = readManifest(t, projectDir)
	if !strings.Contains(manifest, "exclude:") || !strings.Contains(manifest, "plugins") {
		t.Fatalf("re-run dropped exclude:\n%s", manifest)
	}

	// Passing --include adds include and keeps exclude.
	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "source", "add", "repo-one", remote, "--include", "skills/", "--include", "./skills")
	if err != nil {
		t.Fatalf("source add (--include) error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "(include: skills; exclude: plugins)") {
		t.Fatalf("stdout = %q", stdout)
	}
	manifest = readManifest(t, projectDir)
	for _, want := range []string{"include:", "exclude:", "plugins"} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("manifest missing %q:\n%s", want, manifest)
		}
	}
	if strings.Count(manifest, "skills") < 1 {
		t.Fatalf("manifest missing normalized include entry:\n%s", manifest)
	}

	// The duplicate name now resolves without a path selector because the
	// plugin copy is out of scope.
	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "add", "repo-one", "bedrock")
	if err != nil {
		t.Fatalf("add error = %v, stderr = %s\nmanifest:\n%s", err, stderr, manifest)
	}
	if !strings.Contains(stdout, filepath.Join("skills", "bedrock")) {
		t.Fatalf("add did not link skills/bedrock:\n%s", stdout)
	}
}

func TestSourceAddRejectsExcludeDot(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)
	remote := duplicateSkillRemote(t)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	original := readManifest(t, projectDir)

	_, _, err := executeCommandInDir(t, env, projectDir, "source", "add", "--ref", "main", "repo-one", remote, "--exclude", ".")
	if err == nil || !strings.Contains(err.Error(), "would exclude the whole repository") {
		t.Fatalf("source add error = %v, want whole-repository rejection", err)
	}
	if readManifest(t, projectDir) != original {
		t.Fatal("manifest was modified despite validation failure")
	}
}

func TestSkillListHonorsScopeAndAllFlag(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)
	remote := duplicateSkillRemote(t)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"    exclude: [plugins]",
		"skills: []",
		"",
	}, "\n"))
	if _, stderr, err := executeCommandInDir(t, env, projectDir, "source", "sync"); err != nil {
		t.Fatalf("source sync error = %v, stderr = %s", err, stderr)
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "skill", "list", "--source", "repo-one")
	if err != nil {
		t.Fatalf("skill list error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "repo-one", "bedrock", filepath.Join("skills", "bedrock"))
	assertOutputHasFields(t, stdout, "repo-one", "lambda", filepath.Join("skills", "lambda"))
	if strings.Contains(stdout, filepath.Join("plugins", "x", "bedrock")) {
		t.Fatalf("scoped skill list should omit excluded skill:\n%s", stdout)
	}
	if strings.Contains(stdout, "Scope") {
		t.Fatalf("scoped skill list should not show Scope column:\n%s", stdout)
	}

	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "skill", "list", "--source", "repo-one", "--all")
	if err != nil {
		t.Fatalf("skill list --all error = %v, stderr = %s", err, stderr)
	}
	assertOutputHasFields(t, stdout, "Source", "Name", "Path", "Scope")
	assertOutputHasFields(t, stdout, "repo-one", "bedrock", filepath.Join("plugins", "x", "bedrock"), "excluded")
	assertOutputHasFields(t, stdout, "repo-one", "bedrock", filepath.Join("skills", "bedrock"), "included")
	assertOutputHasFields(t, stdout, "repo-one", "lambda", filepath.Join("skills", "lambda"), "included")
}

func TestAddCommandWithPathAppendsAndLinks(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)
	initGitRepo(t, projectDir)
	remote := duplicateSkillRemote(t)

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

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "bedrock", "--path", "plugins/x/bedrock/SKILL.md")
	if err != nil {
		t.Fatalf("add error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `added skill "bedrock" from source "repo-one" (path plugins/x/bedrock)`) {
		t.Fatalf("stdout = %q", stdout)
	}

	manifest := readManifest(t, projectDir)
	if !strings.Contains(manifest, "path: plugins/x/bedrock") {
		t.Fatalf("manifest missing normalized path entry:\n%s", manifest)
	}

	canonicalPath := filepath.Join(resolvedProjectDir, ".agents", "skills", "bedrock")
	target, err := os.Readlink(canonicalPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", canonicalPath, err)
	}
	if !strings.HasSuffix(target, filepath.Join("plugins", "x", "bedrock")) {
		t.Fatalf("canonical target = %q", target)
	}
	if _, err := os.Lstat(filepath.Join(resolvedProjectDir, ".claude", "skills", "bedrock")); err != nil {
		t.Fatalf("expected Claude link: %v", err)
	}
}

func TestAddCommandFailsOnAmbiguousNameAndRollsBack(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)
	remote := duplicateSkillRemote(t)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	original := strings.Join([]string{
		"sources:",
		"  repo-one: {url: " + remote + ", ref: main}",
		"skills: []",
		"",
	}, "\n")
	writeProjectManifest(t, projectDir, original)

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "bedrock")
	if err == nil {
		t.Fatalf("expected add error, stdout = %s, stderr = %s", stdout, stderr)
	}
	for _, want := range []string{
		"repo-one/bedrock: ambiguous-skill",
		"plugins/x/bedrock",
		"skills/bedrock",
		"set path:",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err.Error(), want)
		}
	}
	if readManifest(t, projectDir) != original {
		t.Fatalf("manifest rollback mismatch:\nwant:\n%s\ngot:\n%s", original, readManifest(t, projectDir))
	}
}

func TestAddCommandUpsertsPathOnExistingEntry(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	resolvedProjectDir := resolvedPath(t, projectDir)
	initGitRepo(t, projectDir)
	remote := duplicateSkillRemote(t)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	// A path-less entry that is ambiguous on its own.
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"# project skills",
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"skills:",
		"  - source: repo-one",
		"    name: bedrock # pinned below",
		"",
	}, "\n"))

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "bedrock", "--path", "skills/bedrock")
	if err != nil {
		t.Fatalf("add --path error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `updated path for skill "bedrock" from source "repo-one" to skills/bedrock`) {
		t.Fatalf("stdout = %q", stdout)
	}
	manifest := readManifest(t, projectDir)
	for _, want := range []string{"# project skills", "# pinned below", `path: "skills/bedrock"`} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("manifest missing %q:\n%s", want, manifest)
		}
	}
	if strings.Count(manifest, "name: bedrock") != 1 {
		t.Fatalf("expected a single bedrock entry:\n%s", manifest)
	}

	target, err := os.Readlink(filepath.Join(resolvedProjectDir, ".agents", "skills", "bedrock"))
	if err != nil {
		t.Fatalf("Readlink error = %v", err)
	}
	if !strings.HasSuffix(target, filepath.Join("skills", "bedrock")) {
		t.Fatalf("target = %q", target)
	}

	// Repeating with the same path (in a differently-spelled form) is a no-op.
	before := manifest
	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "add", "repo-one", "bedrock", "--path", "./skills/bedrock/")
	if err != nil {
		t.Fatalf("add (repeat) error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, `skill "bedrock" from source "repo-one" is already declared`) {
		t.Fatalf("stdout = %q", stdout)
	}
	if readManifest(t, projectDir) != before {
		t.Fatal("no-op add modified the manifest")
	}

	// Omitting --path on an entry that already has one is also a no-op.
	stdout, stderr, err = executeCommandInDir(t, env, projectDir, "add", "repo-one", "bedrock")
	if err != nil {
		t.Fatalf("add (no path) error = %v, stderr = %s", err, stderr)
	}
	if !strings.Contains(stdout, "is already declared") {
		t.Fatalf("stdout = %q", stdout)
	}
	if readManifest(t, projectDir) != before {
		t.Fatal("no-op add modified the manifest")
	}
}

func TestAddCommandRejectsDuplicatePathBeforeWriting(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)
	remote := duplicateSkillRemote(t)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	original := strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"skills:",
		"  - source: repo-one",
		"    name: bedrock",
		"    path: skills/bedrock",
		"",
	}, "\n")
	writeProjectManifest(t, projectDir, original)

	_, _, err := executeCommandInDir(t, env, projectDir, "add", "repo-one", "bedrock-copy", "--path", "skills/bedrock")
	if err == nil || !strings.Contains(err.Error(), "duplicate skill path for repo-one: skills/bedrock") {
		t.Fatalf("add error = %v, want duplicate skill path", err)
	}
	if readManifest(t, projectDir) != original {
		t.Fatal("manifest was modified despite validation failure")
	}
}

func TestDoctorAmbiguousSkillHintSuggestsPathAndScope(t *testing.T) {
	requireGit(t)
	env := newTestEnv(t)
	projectDir := t.TempDir()
	initGitRepo(t, projectDir)
	remote := duplicateSkillRemote(t)

	if _, stderr, err := executeCommandInDir(t, env, projectDir, "init", "--cache=local"); err != nil {
		t.Fatalf("init error = %v, stderr = %s", err, stderr)
	}
	writeProjectManifest(t, projectDir, manifestFor(remote, []string{"bedrock"}))
	// Sync fails on the ambiguity, but clones the source so doctor can inspect it.
	if _, _, err := executeCommandInDir(t, env, projectDir, "sync"); err == nil {
		t.Fatal("expected sync to fail on ambiguous skill")
	}

	stdout, stderr, err := executeCommandInDir(t, env, projectDir, "doctor")
	if err == nil {
		t.Fatalf("expected doctor to report errors, stdout = %s, stderr = %s", stdout, stderr)
	}

	for _, want := range []string{
		"ambiguous-skill",
		"set path: to one of: plugins/x/bedrock, skills/bedrock",
		"run skills add repo-one bedrock --path <candidate> with one of the listed paths, or scope the source with include:/exclude:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}
