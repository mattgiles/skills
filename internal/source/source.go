package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Source struct {
	Alias    string
	URL      string
	RepoPath string
}

type SourceStatus struct {
	Alias      string
	URL        string
	RepoPath   string
	Exists     bool
	IsGitRepo  bool
	HeadRef    string
	HeadCommit string
	LastError  string
}

func RepoPath(repoRoot string, alias string) string {
	return filepath.Join(repoRoot, alias)
}

func EnsureGitAvailable() error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git is required but was not found in PATH")
	}
	return nil
}

func Status(ctx context.Context, src Source) SourceStatus {
	status := SourceStatus{
		Alias:    src.Alias,
		URL:      src.URL,
		RepoPath: src.RepoPath,
	}

	info, err := os.Stat(src.RepoPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status
		}
		status.LastError = err.Error()
		return status
	}

	if !info.IsDir() {
		status.Exists = true
		status.LastError = "path is not a directory"
		return status
	}

	status.Exists = true

	if _, err := gitOutput(ctx, "", "-C", src.RepoPath, "rev-parse", "--is-inside-work-tree"); err != nil {
		status.LastError = err.Error()
		return status
	}

	status.IsGitRepo = true

	ref, err := gitOutput(ctx, "", "-C", src.RepoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		status.LastError = err.Error()
		return status
	}
	status.HeadRef = ref

	commit, err := gitOutput(ctx, "", "-C", src.RepoPath, "rev-parse", "HEAD")
	if err != nil {
		status.LastError = err.Error()
		return status
	}
	status.HeadCommit = commit

	return status
}

func Sync(ctx context.Context, src Source) (bool, error) {
	if err := EnsureGitAvailable(); err != nil {
		return false, err
	}

	if _, err := os.Stat(src.RepoPath); errors.Is(err, os.ErrNotExist) {
		if err := clone(ctx, src); err != nil {
			return false, err
		}
		return true, nil
	} else if err != nil {
		return false, err
	}

	status := Status(ctx, src)
	if !status.IsGitRepo {
		if status.LastError == "" {
			status.LastError = "not a git repository"
		}
		return false, errors.New(status.LastError)
	}

	if err := fetch(ctx, src); err != nil {
		return false, err
	}

	return false, nil
}

func clone(ctx context.Context, src Source) error {
	if err := os.MkdirAll(filepath.Dir(src.RepoPath), 0o755); err != nil {
		return err
	}

	_, err := gitOutput(ctx, "", "clone", src.URL, src.RepoPath)
	return err
}

func fetch(ctx context.Context, src Source) error {
	_, err := gitOutput(ctx, "", "-C", src.RepoPath, "fetch", "--all", "--prune")
	return err
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}

	return strings.TrimSpace(stdout.String()), nil
}
