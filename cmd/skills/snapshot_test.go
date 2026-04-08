package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	headingPattern   = regexp.MustCompile(`^(#{1,6})\s+(.*\S)\s*$`)
	pathLabelPattern = regexp.MustCompile("^`([^`]+)`:\\s*$")
	fencePattern     = regexp.MustCompile("^```(.*)$")
	exitPattern      = regexp.MustCompile(`^<!--\s*exit:\s*(\d+)\s*-->\s*$`)
	shaPattern       = regexp.MustCompile(`\b[0-9a-fA-F]{7,40}\b`)
	repoPattern      = regexp.MustCompile(`^repo\s+([a-zA-Z0-9._-]+)$`)
	commitPattern    = regexp.MustCompile(`^commit\s+"([^"]+)"\s*$`)
	tagPattern       = regexp.MustCompile(`^tag\s+"([^"]+)"\s*$`)
	branchPattern    = regexp.MustCompile(`^branch\s+"([^"]+)"\s*$`)
)

type snapshotSuite struct {
	Path  string
	Cases []snapshotCase
}

type snapshotCase struct {
	Name  string
	Steps []snapshotStep
}

type snapshotStep interface {
	isSnapshotStep()
}

type fileStep struct {
	Path    string
	Content string
}

func (fileStep) isSnapshotStep() {}

type repoStep struct {
	Name string
	Ops  []repoOp
}

func (repoStep) isSnapshotStep() {}

type repoOp struct {
	Kind  string
	Value string
}

type commandStep struct {
	Command          string
	WantStdoutAssert string
	WantStderr       string
	WantExit         int
}

func (*commandStep) isSnapshotStep() {}

type snapshotEnv struct {
	testEnv
	root       string
	projectDir string
	reposDir   string
}

func TestMarkdownSnapshots(t *testing.T) {
	requireGit(t)

	localSuites, err := loadSnapshotSuites(filepath.Join("testdata", "snapshots", "local"))
	if err != nil {
		t.Fatalf("load local snapshot suites: %v", err)
	}
	runSnapshotSuites(t, localSuites)

	if os.Getenv("RUN_LIVE_SNAPSHOT_TESTS") != "1" {
		t.Log("set RUN_LIVE_SNAPSHOT_TESTS=1 to run live GitHub snapshot suites")
		return
	}

	liveSuites, err := loadSnapshotSuites(filepath.Join("testdata", "snapshots", "live"))
	if err != nil {
		t.Fatalf("load live snapshot suites: %v", err)
	}
	runSnapshotSuites(t, liveSuites)
}

func runSnapshotSuites(t *testing.T, suites []snapshotSuite) {
	for _, suite := range suites {
		suite := suite
		suiteName := strings.TrimSuffix(filepath.Base(suite.Path), filepath.Ext(suite.Path))
		t.Run(suiteName, func(t *testing.T) {
			for _, tc := range suite.Cases {
				tc := tc
				t.Run(tc.Name, func(t *testing.T) {
					runSnapshotCase(t, tc)
				})
			}
		})
	}
}

func runSnapshotCase(t *testing.T, tc snapshotCase) {
	env := newSnapshotEnv(t)

	for _, step := range tc.Steps {
		switch step := step.(type) {
		case fileStep:
			path, err := resolveSnapshotPath(env, step.Path)
			if err != nil {
				t.Fatalf("resolve path %q: %v", step.Path, err)
			}
			writeSnapshotFile(t, path, substituteSnapshotVars(step.Content, env))
		case repoStep:
			applyRepoStep(t, env, step)
		case *commandStep:
			runSnapshotCommand(t, env, tc.Name, step)
		default:
			t.Fatalf("unknown snapshot step type %T", step)
		}
	}
}

func newSnapshotEnv(t *testing.T) snapshotEnv {
	t.Helper()

	root := resolvedPath(t, t.TempDir())
	env := snapshotEnv{
		testEnv: testEnv{
			configHome: filepath.Join(root, "skills-config"),
			dataHome:   filepath.Join(root, "skills-data"),
			home:       filepath.Join(root, "home"),
		},
		root:       root,
		projectDir: filepath.Join(root, "project"),
		reposDir:   filepath.Join(root, "repos"),
	}

	for _, path := range []string{env.configHome, env.dataHome, env.home, env.projectDir, env.reposDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", path, err)
		}
	}
	initGitRepo(t, env.projectDir)

	return env
}

func runSnapshotCommand(t *testing.T, env snapshotEnv, caseName string, step *commandStep) {
	t.Helper()

	args, err := splitCommandLine(substituteSnapshotVars(step.Command, env))
	if err != nil {
		t.Fatalf("parse command %q: %v", step.Command, err)
	}
	if len(args) == 0 || args[0] != "skills" {
		t.Fatalf("command must start with \"skills\": %q", step.Command)
	}

	stdout, stderr, err := executeCommandInDir(t, env.testEnv, env.projectDir, args[1:]...)
	gotExit := 0
	if err != nil {
		gotExit = 1
	}
	if gotExit != step.WantExit {
		t.Fatalf("%s: exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", caseName, gotExit, step.WantExit, stdout, stderr)
	}

	if err := assertSnapshotStdout(
		normalizeSnapshotText(substituteSnapshotVars(step.WantStdoutAssert, env), env),
		normalizeSnapshotText(stdout, env),
	); err != nil {
		t.Fatalf("%s: stdout mismatch: %v\nstdout:\n%s", caseName, err, normalizeSnapshotText(stdout, env))
	}

	wantStderr := normalizeSnapshotText(substituteSnapshotVars(step.WantStderr, env), env)
	gotStderr := normalizeSnapshotText(stderr, env)
	if gotStderr != wantStderr {
		t.Fatalf("%s: stderr mismatch\nwant:\n%s\n\ngot:\n%s", caseName, wantStderr, gotStderr)
	}
}

func applyRepoStep(t *testing.T, env snapshotEnv, step repoStep) {
	t.Helper()

	repoDir := filepath.Join(env.reposDir, step.Name)
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", repoDir, err)
	}

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); os.IsNotExist(err) {
		runGit(t, repoDir, "init", "-b", "main")
		runGit(t, repoDir, "config", "user.name", "Codex Snapshot")
		runGit(t, repoDir, "config", "user.email", "codex@example.com")
	} else if err != nil {
		t.Fatalf("Stat(%q): %v", filepath.Join(repoDir, ".git"), err)
	}

	for _, op := range step.Ops {
		switch op.Kind {
		case "commit":
			runGit(t, repoDir, "add", ".")
			runGit(t, repoDir, "commit", "--allow-empty", "-m", op.Value)
		case "tag":
			runGit(t, repoDir, "tag", op.Value)
		case "branch":
			runGit(t, repoDir, "branch", op.Value)
		default:
			t.Fatalf("unknown repo op %q", op.Kind)
		}
	}
}

func loadSnapshotSuites(dir string) ([]snapshotSuite, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	suites := make([]snapshotSuite, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		suite, err := parseSnapshotSuite(path)
		if err != nil {
			return nil, err
		}
		suites = append(suites, suite)
	}
	return suites, nil
}

func parseSnapshotSuite(path string) (snapshotSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return snapshotSuite{}, err
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	suite := snapshotSuite{Path: path}
	headings := []string{}
	var current *snapshotCase
	var lastCommand *commandStep
	var pendingPath string
	var pendingExit *int

	finalize := func() {
		if current == nil {
			return
		}
		hasCommand := false
		for _, step := range current.Steps {
			if _, ok := step.(*commandStep); ok {
				hasCommand = true
				break
			}
		}
		if hasCommand {
			suite.Cases = append(suite.Cases, *current)
		}
		current = nil
		lastCommand = nil
	}

	ensureCurrent := func() error {
		if current != nil {
			return nil
		}
		if len(headings) == 0 {
			return fmt.Errorf("%s: found content before any heading", path)
		}
		current = &snapshotCase{Name: strings.Join(headings, " / ")}
		return nil
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if matches := headingPattern.FindStringSubmatch(line); matches != nil {
			finalize()
			level := len(matches[1])
			title := matches[2]
			if level <= len(headings) {
				headings = append(headings[:level-1], title)
			} else {
				headings = append(headings, title)
			}
			continue
		}

		if matches := exitPattern.FindStringSubmatch(trimmed); matches != nil {
			value, err := strconv.Atoi(matches[1])
			if err != nil {
				return snapshotSuite{}, fmt.Errorf("%s:%d: invalid exit directive: %w", path, i+1, err)
			}
			pendingExit = &value
			continue
		}

		if matches := pathLabelPattern.FindStringSubmatch(line); matches != nil {
			if err := ensureCurrent(); err != nil {
				return snapshotSuite{}, err
			}
			pendingPath = matches[1]
			continue
		}

		matches := fencePattern.FindStringSubmatch(line)
		if matches == nil {
			return snapshotSuite{}, fmt.Errorf("%s:%d: unsupported line outside block: %s", path, i+1, line)
		}
		if err := ensureCurrent(); err != nil {
			return snapshotSuite{}, err
		}

		info := strings.TrimSpace(matches[1])
		startLine := i + 1
		i++
		blockLines := []string{}
		for ; i < len(lines); i++ {
			if fencePattern.MatchString(lines[i]) && strings.TrimSpace(lines[i]) == "```" {
				break
			}
			blockLines = append(blockLines, lines[i])
		}
		if i >= len(lines) {
			return snapshotSuite{}, fmt.Errorf("%s:%d: unterminated fenced block", path, startLine)
		}

		content := strings.Join(blockLines, "\n")
		switch {
		case pendingPath != "":
			current.Steps = append(current.Steps, fileStep{Path: pendingPath, Content: content})
			pendingPath = ""
			lastCommand = nil
		case info == "command":
			step := &commandStep{
				Command:  strings.TrimSpace(content),
				WantExit: 0,
			}
			if pendingExit != nil {
				step.WantExit = *pendingExit
				pendingExit = nil
			}
			current.Steps = append(current.Steps, step)
			lastCommand = step
		case info == "stdout":
			return snapshotSuite{}, fmt.Errorf("%s:%d: stdout blocks are no longer supported; use stdout-assert", path, startLine)
		case info == "stdout-assert":
			if lastCommand == nil {
				return snapshotSuite{}, fmt.Errorf("%s:%d: stdout-assert block must follow a command block", path, startLine)
			}
			lastCommand.WantStdoutAssert = content
		case info == "stderr":
			if lastCommand == nil {
				return snapshotSuite{}, fmt.Errorf("%s:%d: stderr block must follow a command block", path, startLine)
			}
			lastCommand.WantStderr = content
		default:
			repoMatches := repoPattern.FindStringSubmatch(info)
			if repoMatches == nil {
				return snapshotSuite{}, fmt.Errorf("%s:%d: unsupported fenced block %q", path, startLine, info)
			}
			ops, err := parseRepoOps(content)
			if err != nil {
				return snapshotSuite{}, fmt.Errorf("%s:%d: %w", path, startLine, err)
			}
			current.Steps = append(current.Steps, repoStep{Name: repoMatches[1], Ops: ops})
			lastCommand = nil
		}
	}

	finalize()
	return suite, nil
}

func parseRepoOps(content string) ([]repoOp, error) {
	lines := strings.Split(content, "\n")
	ops := make([]repoOp, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch {
		case commitPattern.MatchString(line):
			ops = append(ops, repoOp{Kind: "commit", Value: commitPattern.FindStringSubmatch(line)[1]})
		case tagPattern.MatchString(line):
			ops = append(ops, repoOp{Kind: "tag", Value: tagPattern.FindStringSubmatch(line)[1]})
		case branchPattern.MatchString(line):
			ops = append(ops, repoOp{Kind: "branch", Value: branchPattern.FindStringSubmatch(line)[1]})
		default:
			return nil, fmt.Errorf("unsupported repo op %q", line)
		}
	}
	return ops, nil
}

func resolveSnapshotPath(env snapshotEnv, label string) (string, error) {
	parts := strings.SplitN(filepath.ToSlash(label), "/", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("path %q must start with a root like project/ or repo/", label)
	}

	switch parts[0] {
	case "project":
		return filepath.Join(env.projectDir, parts[1], strings.TrimPrefix(strings.Join(parts[2:], "/"), "/")), nil
	case "config":
		return filepath.Join(env.configHome, "skills", parts[1], strings.TrimPrefix(strings.Join(parts[2:], "/"), "/")), nil
	case "data":
		return filepath.Join(env.dataHome, "skills", parts[1], strings.TrimPrefix(strings.Join(parts[2:], "/"), "/")), nil
	case "home":
		return filepath.Join(env.home, parts[1], strings.TrimPrefix(strings.Join(parts[2:], "/"), "/")), nil
	case "repo":
		if len(parts) < 3 {
			return "", fmt.Errorf("repo path %q must include a repo name and file path", label)
		}
		return filepath.Join(env.reposDir, parts[1], parts[2]), nil
	default:
		return "", fmt.Errorf("unsupported snapshot path root %q", parts[0])
	}
}

func writeSnapshotFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func substituteSnapshotVars(value string, env snapshotEnv) string {
	replacements := map[string]string{
		"{{project}}": env.projectDir,
		"{{config}}":  filepath.Join(env.configHome, "skills"),
		"{{data}}":    filepath.Join(env.dataHome, "skills"),
		"{{home}}":    env.home,
		"{{repos}}":   env.reposDir,
	}
	for token, replacement := range replacements {
		value = strings.ReplaceAll(value, token, replacement)
	}

	return regexp.MustCompile(`\{\{repo:([a-zA-Z0-9._-]+)\}\}`).ReplaceAllStringFunc(value, func(match string) string {
		name := regexp.MustCompile(`\{\{repo:([a-zA-Z0-9._-]+)\}\}`).FindStringSubmatch(match)[1]
		return filepath.Join(env.reposDir, name)
	})
}

func normalizeSnapshotText(value string, env snapshotEnv) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")

	replacements := []struct {
		from string
		to   string
	}{
		{env.projectDir, "<project>"},
		{filepath.Join(env.configHome, "skills"), "<config>"},
		{filepath.Join(env.dataHome, "skills"), "<data>"},
		{env.reposDir, "<repos>"},
		{env.home, "<home>"},
		{env.root, "<tmp>"},
	}
	for _, replacement := range replacements {
		value = strings.ReplaceAll(value, replacement.from, replacement.to)
	}

	value = shaPattern.ReplaceAllString(value, "<sha>")

	lines := strings.Split(value, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			lines[i] = ""
			continue
		}
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func assertSnapshotStdout(want string, got string) error {
	wantSections, err := parseSnapshotSections(want)
	if err != nil {
		return fmt.Errorf("parse expected stdout: %w", err)
	}
	gotSections, err := parseSnapshotSections(got)
	if err != nil {
		return fmt.Errorf("parse actual stdout: %w", err)
	}

	for _, section := range wantSections.order {
		wantLines := wantSections.sections[section]
		gotLines, ok := gotSections.sections[section]
		if !ok {
			return fmt.Errorf("missing section %q", section)
		}

		for _, wantLine := range wantLines {
			if !containsLine(gotLines, wantLine) {
				return fmt.Errorf(
					"section %q missing line %q\nactual section:\n%s",
					section,
					wantLine,
					strings.Join(gotLines, "\n"),
				)
			}
		}
	}

	return nil
}

type snapshotSections struct {
	order    []string
	sections map[string][]string
}

func parseSnapshotSections(value string) (snapshotSections, error) {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	result := snapshotSections{
		order:    []string{},
		sections: map[string][]string{},
	}

	current := ""
	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if current == "" {
				return snapshotSections{}, fmt.Errorf("line %d: empty section name", idx+1)
			}
			if _, exists := result.sections[current]; !exists {
				result.order = append(result.order, current)
				result.sections[current] = []string{}
			}
			continue
		}

		if strings.HasPrefix(line, "# ") {
			current = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			if current == "" {
				return snapshotSections{}, fmt.Errorf("line %d: empty rendered section", idx+1)
			}
			if _, exists := result.sections[current]; !exists {
				result.order = append(result.order, current)
				result.sections[current] = []string{}
			}
			continue
		}

		if current == "" {
			current = "Prelude"
			if _, exists := result.sections[current]; !exists {
				result.order = append(result.order, current)
				result.sections[current] = []string{}
			}
		}
		result.sections[current] = append(result.sections[current], line)
	}

	return result, nil
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

func TestParseSnapshotSections(t *testing.T) {
	got, err := parseSnapshotSections(strings.TrimSpace(`
[Workspace]
Scope repo

[Sources]
repo-one resolved main <sha>
`))
	if err != nil {
		t.Fatalf("parseSnapshotSections() error = %v", err)
	}

	if len(got.order) != 2 || got.order[0] != "Workspace" || got.order[1] != "Sources" {
		t.Fatalf("section order = %#v", got.order)
	}
	if !containsLine(got.sections["Workspace"], "Scope repo") {
		t.Fatalf("workspace lines = %#v", got.sections["Workspace"])
	}
	if !containsLine(got.sections["Sources"], "repo-one resolved main <sha>") {
		t.Fatalf("source lines = %#v", got.sections["Sources"])
	}
}

func TestParseSnapshotSectionsCapturesPrelude(t *testing.T) {
	got, err := parseSnapshotSections("SUCCESS: added skill\n# Workspace\nScope repo")
	if err != nil {
		t.Fatalf("parseSnapshotSections() error = %v", err)
	}
	if len(got.order) != 2 || got.order[0] != "Prelude" || got.order[1] != "Workspace" {
		t.Fatalf("section order = %#v", got.order)
	}
	if !containsLine(got.sections["Prelude"], "SUCCESS: added skill") {
		t.Fatalf("prelude lines = %#v", got.sections["Prelude"])
	}
}

func TestAssertSnapshotStdout(t *testing.T) {
	want := strings.TrimSpace(`
[Workspace]
Scope repo

[Sources]
repo-one resolved main <sha>
`)

	got := strings.TrimSpace(`
# Workspace
Scope repo
Root <project>

# Sources
Source Status Ref Commit
repo-one resolved main <sha>
`)

	if err := assertSnapshotStdout(want, got); err != nil {
		t.Fatalf("assertSnapshotStdout() error = %v", err)
	}
}

func TestAssertSnapshotStdoutMissingSection(t *testing.T) {
	want := "[Sources]\nrepo-one resolved main <sha>"
	got := "# Workspace\nScope repo"

	err := assertSnapshotStdout(want, got)
	if err == nil || !strings.Contains(err.Error(), `missing section "Sources"`) {
		t.Fatalf("assertSnapshotStdout() error = %v, want missing section", err)
	}
}

func TestAssertSnapshotStdoutMissingLine(t *testing.T) {
	want := "[Sources]\nrepo-one resolved main <sha>"
	got := "# Sources\nrepo-one linked main <sha>"

	err := assertSnapshotStdout(want, got)
	if err == nil || !strings.Contains(err.Error(), `section "Sources" missing line`) {
		t.Fatalf("assertSnapshotStdout() error = %v, want missing line", err)
	}
}

func splitCommandLine(value string) ([]string, error) {
	args := []string{}
	var current strings.Builder
	var quote rune
	escaped := false

	flush := func() {
		if current.Len() == 0 {
			return
		}
		args = append(args, current.String())
		current.Reset()
	}

	for _, r := range value {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '\n':
			if current.Len() != 0 {
				flush()
			}
		case r == ' ' || r == '\t':
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return nil, fmt.Errorf("trailing escape in command %q", value)
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in command %q", value)
	}
	flush()
	return args, nil
}
