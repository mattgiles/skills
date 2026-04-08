package project

import (
	"errors"
	"fmt"
	"os"
)

type CacheCleanResult struct {
	RepoRoot     string
	WorktreeRoot string
}

func CleanCachePaths(repoRoot string, worktreeRoot string) (CacheCleanResult, error) {
	if err := resetCacheRoot(repoRoot, "repo cache root"); err != nil {
		return CacheCleanResult{}, err
	}
	if err := resetCacheRoot(worktreeRoot, "worktree cache root"); err != nil {
		return CacheCleanResult{}, err
	}

	return CacheCleanResult{
		RepoRoot:     repoRoot,
		WorktreeRoot: worktreeRoot,
	}, nil
}

func resetCacheRoot(path string, label string) error {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return os.MkdirAll(path, 0o755)
	case err != nil:
		return err
	case !info.IsDir():
		return fmt.Errorf("%s exists and is not a directory: %s", label, path)
	}

	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o755)
}
