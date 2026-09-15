package project

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// duplicateSkillRemote builds a source repo that ships the same skill twice
// under different directories, mirroring upstream repos that vendor plugin
// copies alongside a canonical skills tree.
func duplicateSkillRemote(t *testing.T) string {
	t.Helper()
	return initRemoteRepo(t, map[string]string{
		"plugins/x/bedrock/SKILL.md": "# bedrock (plugin copy)",
		"skills/bedrock/SKILL.md":    "# bedrock",
		"skills/lambda/SKILL.md":     "# lambda",
	})
}

func setupDuplicateProject(t *testing.T) (string, string) {
	t.Helper()
	requireGit(t)
	_ = newProjectTestEnv(t)
	projectDir := resolvedPath(t, t.TempDir())
	initGitRepo(t, projectDir)
	remote := duplicateSkillRemote(t)
	if _, err := InitProject(context.Background(), projectDir, InitProjectOptions{CacheMode: CacheModeLocal}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}
	return projectDir, remote
}

func findLinkReport(t *testing.T, reports []LinkReport, source string, skill string) LinkReport {
	t.Helper()
	for _, report := range reports {
		if report.Source == source && report.Skill == skill {
			return report
		}
	}
	t.Fatalf("link %s/%s not found in %+v", source, skill, reports)
	return LinkReport{}
}

func TestSyncReportsAmbiguousSkillWithCandidatePaths(t *testing.T) {
	projectDir, remote := setupDuplicateProject(t)
	writeProjectManifest(t, projectDir, manifestFor(remote, "main", []string{"bedrock"}))

	_, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err == nil {
		t.Fatal("Sync() expected ambiguous-skill error")
	}
	for _, want := range []string{
		"repo-one/bedrock: ambiguous-skill",
		"set path: to one of: plugins/x/bedrock, skills/bedrock",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Sync() error = %q, want substring %q", err.Error(), want)
		}
	}

	report, err := Status(context.Background(), projectDir)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	link := findLinkReport(t, report.SkillLinks, "repo-one", "bedrock")
	if link.Status != "ambiguous-skill" {
		t.Fatalf("status = %q, want ambiguous-skill", link.Status)
	}
	if !strings.Contains(link.Message, "plugins/x/bedrock, skills/bedrock") {
		t.Fatalf("message = %q", link.Message)
	}
}

func TestSyncExcludeScopeResolvesDuplicate(t *testing.T) {
	projectDir, remote := setupDuplicateProject(t)
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"    exclude: [plugins]",
		"skills:",
		"  - source: repo-one",
		"    name: bedrock",
		"",
	}, "\n"))

	result, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	link := findLinkReport(t, result.SkillLinks, "repo-one", "bedrock")
	if link.Status != "created" {
		t.Fatalf("status = %q, want created", link.Status)
	}
	if !strings.HasSuffix(link.Target, filepath.Join("skills", "bedrock")) {
		t.Fatalf("target = %q, want suffix skills/bedrock", link.Target)
	}
}

func TestSyncIncludeScopeResolvesDuplicate(t *testing.T) {
	projectDir, remote := setupDuplicateProject(t)
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"    include:",
		"      - skills",
		"skills:",
		"  - source: repo-one",
		"    name: bedrock",
		"",
	}, "\n"))

	result, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	link := findLinkReport(t, result.SkillLinks, "repo-one", "bedrock")
	if !strings.HasSuffix(link.Target, filepath.Join("skills", "bedrock")) {
		t.Fatalf("target = %q, want suffix skills/bedrock", link.Target)
	}
}

func TestSyncMissingSkillWhenOnlyOutsideScope(t *testing.T) {
	projectDir, remote := setupDuplicateProject(t)
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"    include: [plugins]",
		"    exclude: [plugins/x]",
		"skills:",
		"  - source: repo-one",
		"    name: bedrock",
		"",
	}, "\n"))

	_, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err == nil {
		t.Fatal("Sync() expected missing-skill error")
	}
	for _, want := range []string{
		"repo-one/bedrock: missing-skill",
		"skill directory exists only outside the source's include/exclude scope: plugins/x/bedrock, skills/bedrock",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Sync() error = %q, want substring %q", err.Error(), want)
		}
	}
}

func TestSyncPathSelectorPicksCandidate(t *testing.T) {
	for _, selector := range []string{"plugins/x/bedrock", "plugins/x/bedrock/SKILL.md", "./plugins/x/bedrock/"} {
		t.Run(selector, func(t *testing.T) {
			projectDir, remote := setupDuplicateProject(t)
			writeProjectManifest(t, projectDir, strings.Join([]string{
				"sources:",
				"  repo-one:",
				"    url: " + remote,
				"    ref: main",
				"skills:",
				"  - source: repo-one",
				"    name: bedrock",
				"    path: " + selector,
				"",
			}, "\n"))

			result, err := Sync(context.Background(), projectDir, SyncOptions{})
			if err != nil {
				t.Fatalf("Sync() error = %v", err)
			}
			link := findLinkReport(t, result.SkillLinks, "repo-one", "bedrock")
			if link.Status != "created" {
				t.Fatalf("status = %q, want created", link.Status)
			}
			if !strings.HasSuffix(link.Target, filepath.Join("plugins", "x", "bedrock")) {
				t.Fatalf("target = %q, want suffix plugins/x/bedrock", link.Target)
			}
			if link.Path != filepath.Join(SkillsDir(projectDir), "bedrock") {
				t.Fatalf("path = %q, want link dir named bedrock", link.Path)
			}
		})
	}
}

func TestSyncPathSelectorExcludedByScope(t *testing.T) {
	projectDir, remote := setupDuplicateProject(t)
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"    exclude: [plugins]",
		"skills:",
		"  - source: repo-one",
		"    name: bedrock",
		"    path: plugins/x/bedrock",
		"",
	}, "\n"))

	_, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err == nil {
		t.Fatal("Sync() expected missing-skill error")
	}
	for _, want := range []string{
		"repo-one/bedrock: missing-skill",
		`skill path "plugins/x/bedrock" is excluded by the source's include/exclude scope`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Sync() error = %q, want substring %q", err.Error(), want)
		}
	}
}

func TestSyncPathSelectorNotFound(t *testing.T) {
	projectDir, remote := setupDuplicateProject(t)
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"skills:",
		"  - source: repo-one",
		"    name: bedrock",
		"    path: does/not/exist",
		"",
	}, "\n"))

	_, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err == nil {
		t.Fatal("Sync() expected missing-skill error")
	}
	if !strings.Contains(err.Error(), `no skill directory at path "does/not/exist"`) {
		t.Fatalf("Sync() error = %q", err.Error())
	}
}

func TestSyncInstallsBothCopiesUnderDistinctNames(t *testing.T) {
	projectDir, remote := setupDuplicateProject(t)
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"skills:",
		"  - source: repo-one",
		"    name: bedrock-a",
		"    path: plugins/x/bedrock",
		"  - source: repo-one",
		"    name: bedrock-b",
		"    path: skills/bedrock",
		"",
	}, "\n"))

	result, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	a := findLinkReport(t, result.SkillLinks, "repo-one", "bedrock-a")
	b := findLinkReport(t, result.SkillLinks, "repo-one", "bedrock-b")
	if a.Status != "created" || b.Status != "created" {
		t.Fatalf("statuses = %q/%q, want created/created", a.Status, b.Status)
	}
	if !strings.HasSuffix(a.Target, filepath.Join("plugins", "x", "bedrock")) {
		t.Fatalf("bedrock-a target = %q", a.Target)
	}
	if !strings.HasSuffix(b.Target, filepath.Join("skills", "bedrock")) {
		t.Fatalf("bedrock-b target = %q", b.Target)
	}
	if a.Path == b.Path {
		t.Fatalf("both links share destination %q", a.Path)
	}
	for _, name := range []string{"bedrock-a", "bedrock-b"} {
		if _, err := os.Lstat(filepath.Join(SkillsDir(projectDir), name)); err != nil {
			t.Fatalf("expected link %q: %v", name, err)
		}
		if _, err := os.Lstat(filepath.Join(ClaudeSkillsDir(projectDir), name)); err != nil {
			t.Fatalf("expected Claude link %q: %v", name, err)
		}
	}
}

func TestSyncPathSelectorRenamesLinkDirectory(t *testing.T) {
	projectDir, remote := setupDuplicateProject(t)
	writeProjectManifest(t, projectDir, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: " + remote,
		"    ref: main",
		"skills:",
		"  - source: repo-one",
		"    name: renamed",
		"    path: skills/bedrock",
		"",
	}, "\n"))

	result, err := Sync(context.Background(), projectDir, SyncOptions{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	link := findLinkReport(t, result.SkillLinks, "repo-one", "renamed")
	wantPath := filepath.Join(SkillsDir(projectDir), "renamed")
	if link.Path != wantPath {
		t.Fatalf("path = %q, want %q", link.Path, wantPath)
	}
	target, err := os.Readlink(wantPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", wantPath, err)
	}
	if !strings.HasSuffix(target, filepath.Join("skills", "bedrock")) {
		t.Fatalf("target = %q", target)
	}
}

func TestValidateManifestAcceptsScopeAndPath(t *testing.T) {
	manifest := Manifest{
		Sources: map[string]ManifestSource{
			"repo-one": {Ref: "main", Include: []string{"skills", "."}, Exclude: []string{"plugins", "./vendor/"}},
		},
		Skills: []ManifestSkill{
			{Source: "repo-one", Name: "a", Path: "skills/a"},
			{Source: "repo-one", Name: "b", Path: "skills/b/SKILL.md"},
			{Source: "repo-one", Name: "c"},
			{Source: "repo-one", Name: "root", Path: "."},
		},
	}
	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("ValidateManifest() error = %v", err)
	}
}

func TestValidateManifestRejectsBadScopeEntries(t *testing.T) {
	cases := []struct {
		name string
		src  ManifestSource
		want string
	}{
		{"absolute include", ManifestSource{Ref: "main", Include: []string{"/abs"}}, "is absolute"},
		{"absolute exclude", ManifestSource{Ref: "main", Exclude: []string{"/abs"}}, "is absolute"},
		{"escaping include", ManifestSource{Ref: "main", Include: []string{"../up"}}, "escapes the repository"},
		{"escaping exclude", ManifestSource{Ref: "main", Exclude: []string{"a/../../up"}}, "escapes the repository"},
		{"dotdot exclude", ManifestSource{Ref: "main", Exclude: []string{".."}}, "escapes the repository"},
		{"empty include", ManifestSource{Ref: "main", Include: []string{"  "}}, "empty include entry"},
		{"exclude dot", ManifestSource{Ref: "main", Exclude: []string{"."}}, "would exclude the whole repository"},
		{"exclude dot slash", ManifestSource{Ref: "main", Exclude: []string{"./"}}, "would exclude the whole repository"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateManifest(Manifest{Sources: map[string]ManifestSource{"repo-one": tc.src}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateManifest() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestValidateManifestRejectsBadSkillPaths(t *testing.T) {
	sources := map[string]ManifestSource{"repo-one": {Ref: "main"}}
	cases := []struct {
		name   string
		skills []ManifestSkill
		want   string
	}{
		{
			"absolute path",
			[]ManifestSkill{{Source: "repo-one", Name: "a", Path: "/abs/a"}},
			"has an absolute path",
		},
		{
			"escaping path",
			[]ManifestSkill{{Source: "repo-one", Name: "a", Path: "../a"}},
			"escapes the repository",
		},
		{
			"duplicate normalized path",
			[]ManifestSkill{
				{Source: "repo-one", Name: "a", Path: "a/x"},
				{Source: "repo-one", Name: "b", Path: "a/x/SKILL.md"},
			},
			"duplicate skill path for repo-one: a/x (declared by a and b)",
		},
		{
			"duplicate name still rejected",
			[]ManifestSkill{
				{Source: "repo-one", Name: "a", Path: "a/x"},
				{Source: "repo-one", Name: "a", Path: "a/y"},
			},
			"duplicate skill declaration for repo-one/a",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateManifest(Manifest{Sources: sources, Skills: tc.skills})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateManifest() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestValidateManifestAllowsSamePathAcrossSources(t *testing.T) {
	manifest := Manifest{
		Sources: map[string]ManifestSource{
			"repo-one": {Ref: "main"},
			"repo-two": {Ref: "main"},
		},
		Skills: []ManifestSkill{
			{Source: "repo-one", Name: "a", Path: "skills/a"},
			{Source: "repo-two", Name: "a", Path: "skills/a"},
		},
	}
	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("ValidateManifest() error = %v", err)
	}
}

func TestLoadManifestRoundTripsScopeAndPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	mustWriteFile(t, path, strings.Join([]string{
		"sources:",
		"  repo-one:",
		"    url: https://example.com/repo.git",
		"    ref: main",
		"    include: [skills]",
		"    exclude: [plugins, vendor]",
		"skills:",
		"  - source: repo-one",
		"    name: bedrock",
		"    path: skills/bedrock",
		"",
	}, "\n"))

	manifest, err := LoadManifestAt(path)
	if err != nil {
		t.Fatalf("LoadManifestAt() error = %v", err)
	}
	src := manifest.Sources["repo-one"]
	if len(src.Include) != 1 || src.Include[0] != "skills" {
		t.Fatalf("include = %#v", src.Include)
	}
	if len(src.Exclude) != 2 || src.Exclude[0] != "plugins" || src.Exclude[1] != "vendor" {
		t.Fatalf("exclude = %#v", src.Exclude)
	}
	if manifest.Skills[0].Path != "skills/bedrock" {
		t.Fatalf("path = %q", manifest.Skills[0].Path)
	}

	// Saving a manifest with no scope/path must not emit the new keys.
	plain := filepath.Join(dir, "plain.yaml")
	if err := SaveManifestAt(plain, Manifest{
		Sources: map[string]ManifestSource{"repo-one": {URL: "x", Ref: "main"}},
		Skills:  []ManifestSkill{{Source: "repo-one", Name: "a"}},
	}); err != nil {
		t.Fatalf("SaveManifestAt() error = %v", err)
	}
	data, err := os.ReadFile(plain)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	for _, key := range []string{"include", "exclude", "path"} {
		if strings.Contains(string(data), key+":") {
			t.Fatalf("plain manifest unexpectedly contains %q:\n%s", key, data)
		}
	}
}

func TestUpsertManifestSourceAtWritesScopeBlockStyle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	mustWriteFile(t, path, strings.Join([]string{
		"# top comment",
		"sources:",
		"  repo-one:",
		"    url: https://example.com/one.git # inline",
		"    ref: main",
		"skills: []",
		"",
	}, "\n"))

	if err := UpsertManifestSourceAt(path, "repo-one", ManifestSource{
		URL:     "https://example.com/one.git",
		Ref:     "main",
		Include: []string{"skills"},
		Exclude: []string{"plugins", "vendor"},
	}); err != nil {
		t.Fatalf("UpsertManifestSourceAt() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	for _, want := range []string{"# top comment", "include:", "- skills", "exclude:", "- plugins", "- vendor"} {
		if !strings.Contains(text, want) {
			t.Fatalf("manifest missing %q:\n%s", want, text)
		}
	}
	manifest, err := LoadManifestAt(path)
	if err != nil {
		t.Fatalf("LoadManifestAt() error = %v", err)
	}
	if got := manifest.Sources["repo-one"].Exclude; len(got) != 2 {
		t.Fatalf("exclude = %#v", got)
	}
}

func TestUpsertManifestSourceAtWritesScopeFlowStyle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	mustWriteFile(t, path, strings.Join([]string{
		"sources:",
		"  repo-one: {url: https://example.com/one.git, ref: main}",
		"skills: []",
		"",
	}, "\n"))

	if err := UpsertManifestSourceAt(path, "repo-one", ManifestSource{
		URL:     "https://example.com/one.git",
		Ref:     "main",
		Exclude: []string{"plugins"},
	}); err != nil {
		t.Fatalf("UpsertManifestSourceAt() error = %v", err)
	}
	if err := UpsertManifestSourceAt(path, "repo-two", ManifestSource{
		URL:     "https://example.com/two.git",
		Ref:     "main",
		Include: []string{"a", "b"},
		Exclude: []string{"a/c"},
	}); err != nil {
		t.Fatalf("UpsertManifestSourceAt() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	for _, want := range []string{
		`repo-one: {url: "https://example.com/one.git", ref: "main", exclude: ["plugins"]}`,
		`repo-two: {url: "https://example.com/two.git", ref: "main", include: ["a", "b"], exclude: ["a/c"]}`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("manifest missing %q:\n%s", want, text)
		}
	}
	if _, err := LoadManifestAt(path); err != nil {
		t.Fatalf("LoadManifestAt() error = %v", err)
	}
}

func TestAppendManifestSkillAtWritesPath(t *testing.T) {
	t.Run("block", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "manifest.yaml")
		mustWriteFile(t, path, strings.Join([]string{
			"sources:",
			"  repo-one: {url: x, ref: main}",
			"skills:",
			"  - source: repo-one",
			"    name: a",
			"",
		}, "\n"))
		if err := AppendManifestSkillAt(path, ManifestSkill{Source: "repo-one", Name: "b", Path: "skills/b"}); err != nil {
			t.Fatalf("AppendManifestSkillAt() error = %v", err)
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "path: skills/b") {
			t.Fatalf("manifest missing path line:\n%s", data)
		}
		manifest, err := LoadManifestAt(path)
		if err != nil {
			t.Fatalf("LoadManifestAt() error = %v", err)
		}
		if manifest.Skills[1].Path != "skills/b" {
			t.Fatalf("path = %q", manifest.Skills[1].Path)
		}
	})

	t.Run("flow", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "manifest.yaml")
		mustWriteFile(t, path, strings.Join([]string{
			"sources:",
			"  repo-one: {url: x, ref: main}",
			"skills: [{source: repo-one, name: a}]",
			"",
		}, "\n"))
		if err := AppendManifestSkillAt(path, ManifestSkill{Source: "repo-one", Name: "b", Path: "skills/b"}); err != nil {
			t.Fatalf("AppendManifestSkillAt() error = %v", err)
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), `{source: "repo-one", name: "b", path: "skills/b"}`) {
			t.Fatalf("manifest missing flow entry:\n%s", data)
		}
		if _, err := LoadManifestAt(path); err != nil {
			t.Fatalf("LoadManifestAt() error = %v", err)
		}
	})
}

func TestSetManifestSkillPathAt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	original := strings.Join([]string{
		"# header",
		"sources:",
		"  repo-one: {url: x, ref: main}",
		"skills:",
		"  - source: repo-one",
		"    name: a # keep me",
		"  - source: repo-one",
		"    name: b",
		"    path: old/b",
		"  - {source: repo-one, name: c}",
		"",
	}, "\n")
	mustWriteFile(t, path, original)

	// Adds path to a path-less entry.
	if err := SetManifestSkillPathAt(path, "repo-one", "a", "skills/a"); err != nil {
		t.Fatalf("SetManifestSkillPathAt(a) error = %v", err)
	}
	// Updates an existing path in place.
	if err := SetManifestSkillPathAt(path, "repo-one", "b", "new/b"); err != nil {
		t.Fatalf("SetManifestSkillPathAt(b) error = %v", err)
	}
	// Works on a flow-style entry.
	if err := SetManifestSkillPathAt(path, "repo-one", "c", "skills/c"); err != nil {
		t.Fatalf("SetManifestSkillPathAt(c) error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	for _, want := range []string{"# header", "# keep me", `path: "skills/a"`, `path: "new/b"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("manifest missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "old/b") {
		t.Fatalf("manifest still contains old path:\n%s", text)
	}

	manifest, err := LoadManifestAt(path)
	if err != nil {
		t.Fatalf("LoadManifestAt() error = %v", err)
	}
	got := map[string]string{}
	for _, skill := range manifest.Skills {
		got[skill.Name] = skill.Path
	}
	if got["a"] != "skills/a" || got["b"] != "new/b" || got["c"] != "skills/c" {
		t.Fatalf("paths = %#v", got)
	}

	// Unknown entry is an error and leaves the file untouched.
	before, _ := os.ReadFile(path)
	err = SetManifestSkillPathAt(path, "repo-one", "zzz", "x")
	if err == nil || !strings.Contains(err.Error(), "skill repo-one/zzz not found in manifest") {
		t.Fatalf("SetManifestSkillPathAt(zzz) error = %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("manifest changed after failed SetManifestSkillPathAt")
	}
}
