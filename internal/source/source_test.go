package source

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusIncludesDefaultRemote(t *testing.T) {
	requireGit(t)

	remote := initRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	local := cloneRepo(t, remote)

	status := Status(context.Background(), Source{
		Alias:    "repo-one",
		URL:      remote,
		RepoPath: local,
	})

	if !status.IsGitRepo {
		t.Fatalf("Status().IsGitRepo = false, error = %q", status.LastError)
	}
	if status.DefaultRemoteRef != "main" {
		t.Fatalf("DefaultRemoteRef = %q, want main", status.DefaultRemoteRef)
	}
	if status.DefaultRemoteCommit == "" {
		t.Fatal("DefaultRemoteCommit is empty")
	}
	if status.HeadCommit == "" {
		t.Fatal("HeadCommit is empty")
	}
}

func TestResolveCommitPrefersRemoteBranchOverTag(t *testing.T) {
	requireGit(t)

	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Codex Test")
	runGit(t, repo, "config", "user.email", "codex@example.com")

	writeFile(t, filepath.Join(repo, "README.md"), "one\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "one")
	branchCommit := gitCmdOutput(t, repo, "rev-parse", "HEAD")

	runGit(t, repo, "branch", "release")

	writeFile(t, filepath.Join(repo, "README.md"), "two\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "two")
	tagCommit := gitCmdOutput(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "tag", "release")

	local := cloneRepo(t, repo)
	src := Source{Alias: "repo-one", URL: repo, RepoPath: local}

	got, err := ResolveCommit(context.Background(), src, "release")
	if err != nil {
		t.Fatalf("ResolveCommit(release) error = %v", err)
	}
	if got != branchCommit {
		t.Fatalf("ResolveCommit(release) = %s, want branch commit %s", got, branchCommit)
	}

	got, err = ResolveCommit(context.Background(), src, "refs/tags/release")
	if err != nil {
		t.Fatalf("ResolveCommit(refs/tags/release) error = %v", err)
	}
	if got != tagCommit {
		t.Fatalf("ResolveCommit(refs/tags/release) = %s, want tag commit %s", got, tagCommit)
	}

	got, err = ResolveCommit(context.Background(), src, tagCommit)
	if err != nil {
		t.Fatalf("ResolveCommit(commit) error = %v", err)
	}
	if got != tagCommit {
		t.Fatalf("ResolveCommit(commit) = %s, want %s", got, tagCommit)
	}
}

func TestSyncFetchesRemoteWithoutChangingLocalHead(t *testing.T) {
	requireGit(t)

	remote := initRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	local := cloneRepo(t, remote)
	src := Source{Alias: "repo-one", URL: remote, RepoPath: local}

	headBefore := gitCmdOutput(t, local, "rev-parse", "HEAD")

	writeFile(t, filepath.Join(remote, "lint", "SKILL.md"), "# lint")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "add lint")
	remoteHead := gitCmdOutput(t, remote, "rev-parse", "HEAD")

	cloned, err := Sync(context.Background(), src)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if cloned {
		t.Fatal("Sync() reported cloned for existing repo")
	}

	headAfter := gitCmdOutput(t, local, "rev-parse", "HEAD")
	if headAfter != headBefore {
		t.Fatalf("HEAD changed after fetch: before %s after %s", headBefore, headAfter)
	}

	remoteTracking := gitCmdOutput(t, local, "rev-parse", "refs/remotes/origin/main")
	if remoteTracking != remoteHead {
		t.Fatalf("origin/main = %s, want %s", remoteTracking, remoteHead)
	}
}

func TestListFilesAtCommit(t *testing.T) {
	requireGit(t)

	remote := initRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
		"nested/other.txt":   "hello\n",
	})
	local := cloneRepo(t, remote)
	commit := gitCmdOutput(t, remote, "rev-parse", "HEAD")

	files, err := ListFilesAtCommit(context.Background(), Source{
		Alias:    "repo-one",
		URL:      remote,
		RepoPath: local,
	}, commit)
	if err != nil {
		t.Fatalf("ListFilesAtCommit() error = %v", err)
	}

	got := strings.Join(files, "\n")
	if !strings.Contains(got, "analytics/SKILL.md") {
		t.Fatalf("files missing analytics/SKILL.md:\n%s", got)
	}
	if !strings.Contains(got, "nested/other.txt") {
		t.Fatalf("files missing nested/other.txt:\n%s", got)
	}
}

func TestEnsureWorktreeReusesAndRejectsWrongCommit(t *testing.T) {
	requireGit(t)

	remote := initRepo(t, map[string]string{
		"analytics/SKILL.md": "# analytics",
	})
	local := cloneRepo(t, remote)
	src := Source{Alias: "repo-one", URL: remote, RepoPath: local}
	commitOne := gitCmdOutput(t, remote, "rev-parse", "HEAD")

	worktreePath := filepath.Join(t.TempDir(), "worktree")
	created, err := EnsureWorktree(context.Background(), src, worktreePath, commitOne)
	if err != nil {
		t.Fatalf("EnsureWorktree(create) error = %v", err)
	}
	if !created {
		t.Fatal("EnsureWorktree(create) = false, want true")
	}

	created, err = EnsureWorktree(context.Background(), src, worktreePath, commitOne)
	if err != nil {
		t.Fatalf("EnsureWorktree(reuse) error = %v", err)
	}
	if created {
		t.Fatal("EnsureWorktree(reuse) = true, want false")
	}

	writeFile(t, filepath.Join(remote, "README.md"), "next\n")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "next")
	commitTwo := gitCmdOutput(t, remote, "rev-parse", "HEAD")
	runGit(t, local, "fetch", "--all", "--prune")

	_, err = EnsureWorktree(context.Background(), src, worktreePath, commitTwo)
	if err == nil {
		t.Fatal("EnsureWorktree() expected mismatched-commit error")
	}
	if !strings.Contains(err.Error(), "want "+commitTwo) {
		t.Fatalf("EnsureWorktree() error = %v", err)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Codex Test")
	runGit(t, repo, "config", "user.email", "codex@example.com")
	for path, contents := range files {
		writeFile(t, filepath.Join(repo, path), contents)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func cloneRepo(t *testing.T, remote string) string {
	t.Helper()

	local := filepath.Join(t.TempDir(), "clone")
	runGit(t, "", "clone", remote, local)
	return local
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func gitCmdOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return strings.TrimSpace(string(output))
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
