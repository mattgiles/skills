package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFragment(t *testing.T, projectDir string, name string, contents string) {
	t.Helper()
	dir := FragmentDirPath(projectDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", name, err)
	}
}

func writeMainManifest(t *testing.T, projectDir string, contents string) {
	t.Helper()
	path := ManifestPath(projectDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func TestLoadEffectiveManifestWithoutFragmentsEqualsLoadManifest(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())
	writeMainManifest(t, projectDir, "sources:\n  repo-one:\n    url: https://example.com/repo\n    ref: main\nskills:\n  - source: repo-one\n    name: analytics\n")

	base, err := LoadManifest(projectDir)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	effective, err := LoadEffectiveManifest(projectDir)
	if err != nil {
		t.Fatalf("LoadEffectiveManifest() error = %v", err)
	}

	if len(effective.Sources) != len(base.Sources) || len(effective.Skills) != len(base.Skills) {
		t.Fatalf("effective manifest differs from base: %+v vs %+v", effective, base)
	}
	if _, ok := effective.Sources["repo-one"]; !ok {
		t.Fatalf("expected repo-one source, got %+v", effective.Sources)
	}
}

func TestLoadEffectiveManifestMergesFragment(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())
	writeMainManifest(t, projectDir, "sources:\n  repo-one:\n    url: https://example.com/one\n    ref: main\nskills:\n  - source: repo-one\n    name: analytics\n")
	writeFragment(t, projectDir, "perk.yaml", "sources:\n  repo-two:\n    url: https://example.com/two\n    ref: main\nskills:\n  - source: repo-one\n    name: reporting\n  - source: repo-two\n    name: lint\n")

	effective, err := LoadEffectiveManifest(projectDir)
	if err != nil {
		t.Fatalf("LoadEffectiveManifest() error = %v", err)
	}

	if _, ok := effective.Sources["repo-one"]; !ok {
		t.Fatalf("expected repo-one source")
	}
	if _, ok := effective.Sources["repo-two"]; !ok {
		t.Fatalf("expected fragment source repo-two")
	}
	if len(effective.Skills) != 3 {
		t.Fatalf("len(Skills) = %d, want 3: %+v", len(effective.Skills), effective.Skills)
	}
}

func TestMergeManifestsDuplicateSourceAlias(t *testing.T) {
	base := Manifest{
		Sources: map[string]ManifestSource{"repo-one": {URL: "https://example.com/one", Ref: "main"}},
	}
	fragment := Manifest{
		Sources: map[string]ManifestSource{"repo-one": {URL: "https://example.com/dup", Ref: "main"}},
	}

	_, err := MergeManifests(base, fragment)
	if err == nil {
		t.Fatal("MergeManifests() expected duplicate source alias error")
	}
	if !strings.Contains(err.Error(), "duplicate source alias") {
		t.Fatalf("error = %v, want duplicate source alias", err)
	}
}

func TestMergeManifestsDedupesSkillAcrossBaseAndFragment(t *testing.T) {
	base := Manifest{
		Sources: map[string]ManifestSource{"repo-one": {URL: "https://example.com/one", Ref: "main"}},
		Skills:  []ManifestSkill{{Source: "repo-one", Name: "analytics"}},
	}
	fragment := Manifest{
		Skills: []ManifestSkill{{Source: "repo-one", Name: "analytics"}},
	}

	merged, err := MergeManifests(base, fragment)
	if err != nil {
		t.Fatalf("MergeManifests() error = %v", err)
	}
	if len(merged.Skills) != 1 {
		t.Fatalf("len(Skills) = %d, want 1 (deduped): %+v", len(merged.Skills), merged.Skills)
	}
}

func TestMergeManifestsDedupesSkillAcrossFragmentsByFilenameOrder(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())
	writeMainManifest(t, projectDir, "sources:\n  repo-one:\n    url: https://example.com/one\n    ref: main\nskills: []\n")
	// Both fragments declare the same (source,name); a.yaml sorts before b.yaml.
	writeFragment(t, projectDir, "a.yaml", "skills:\n  - source: repo-one\n    name: analytics\n")
	writeFragment(t, projectDir, "b.yaml", "skills:\n  - source: repo-one\n    name: analytics\n")

	fragments, err := LoadFragments(projectDir)
	if err != nil {
		t.Fatalf("LoadFragments() error = %v", err)
	}
	if len(fragments) != 2 {
		t.Fatalf("len(fragments) = %d, want 2", len(fragments))
	}

	base, err := LoadManifest(projectDir)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	merged, err := MergeManifests(base, fragments...)
	if err != nil {
		t.Fatalf("MergeManifests() error = %v", err)
	}
	if len(merged.Skills) != 1 {
		t.Fatalf("len(Skills) = %d, want 1 (deduped): %+v", len(merged.Skills), merged.Skills)
	}
}

func TestLoadFragmentsMissingDir(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())
	fragments, err := LoadFragments(projectDir)
	if err != nil {
		t.Fatalf("LoadFragments() error = %v", err)
	}
	if fragments != nil {
		t.Fatalf("expected nil fragments for missing dir, got %+v", fragments)
	}
}

func TestLoadFragmentsIgnoresNonYamlAndSubdirs(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())
	writeFragment(t, projectDir, "perk.yaml", "sources:\n  repo-one:\n    url: https://example.com/one\n    ref: main\nskills: []\n")
	dir := FragmentDirPath(projectDir)
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not yaml"), 0o644); err != nil {
		t.Fatalf("WriteFile(notes.txt) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll(nested) error = %v", err)
	}

	fragments, err := LoadFragments(projectDir)
	if err != nil {
		t.Fatalf("LoadFragments() error = %v", err)
	}
	if len(fragments) != 1 {
		t.Fatalf("len(fragments) = %d, want 1 (ignore non-yaml + subdir)", len(fragments))
	}
}

func TestLoadFragmentsReportsUnparseableFile(t *testing.T) {
	projectDir := resolvedPath(t, t.TempDir())
	writeFragment(t, projectDir, "broken.yaml", "sources: [\n")

	_, err := LoadFragments(projectDir)
	if err == nil {
		t.Fatal("LoadFragments() expected parse error")
	}
	if !strings.Contains(err.Error(), "broken.yaml") {
		t.Fatalf("error = %v, want mention of broken.yaml", err)
	}
}

func TestInitProjectGitignoreOmitsFragmentDir(t *testing.T) {
	requireGit(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)

	result, err := InitProject(t.Context(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal})
	if err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}

	data, err := os.ReadFile(result.GitignorePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", result.GitignorePath, err)
	}
	if strings.Contains(string(data), "manifest.d") {
		t.Fatalf("gitignore unexpectedly contains manifest.d: %s", string(data))
	}
}
