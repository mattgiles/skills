package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"sort"
	"strings"
)

type Info struct {
	Available bool
	Root      string
}

func Discover(ctx context.Context, dir string) (Info, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return Info{Available: false}, nil
	}

	root, err := gitOutput(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		if isNotGitRepo(err) {
			return Info{Available: true}, nil
		}
		return Info{}, err
	}

	return Info{
		Available: true,
		Root:      root,
	}, nil
}

func ListTracked(ctx context.Context, repoRoot string, paths []string) ([]string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, err
	}

	args := []string{"ls-files", "--"}
	args = append(args, paths...)
	output, err := gitOutput(ctx, repoRoot, args...)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(output) == "" {
		return []string{}, nil
	}

	seen := map[string]struct{}{}
	tracked := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		tracked = append(tracked, line)
	}
	sort.Strings(tracked)
	return tracked, nil
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
		return "", errors.New(message)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func isNotGitRepo(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "not a git repository")
}
