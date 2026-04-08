package source

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type Source struct {
	Alias    string
	Ref      string
	URL      string
	RepoPath string
}

type SourceStatus struct {
	Alias               string
	URL                 string
	RepoPath            string
	Exists              bool
	IsGitRepo           bool
	HeadRef             string
	HeadCommit          string
	DefaultRemoteRef    string
	DefaultRemoteCommit string
	LastError           string
}

func RepoPathForURL(repoRoot string, sourceURL string) string {
	return filepath.Join(repoRoot, RepoCacheKey(sourceURL))
}

func RepoCacheKey(sourceURL string) string {
	base := sourceIdentityLabel(sourceURL)
	if base == "" {
		base = "source"
	}

	identity := canonicalSourceIdentity(sourceURL)
	sum := sha256.Sum256([]byte(identity))
	suffix := hex.EncodeToString(sum[:])[:12]
	return sanitizePathComponent(base) + "-" + suffix
}

func WorktreePath(worktreeRoot string, projectID string, alias string, commit string) string {
	return filepath.Join(worktreeRoot, projectID, alias, commit)
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

	defaultRef, defaultCommit := defaultRemoteStatus(ctx, src.RepoPath)
	status.DefaultRemoteRef = defaultRef
	status.DefaultRemoteCommit = defaultCommit

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

	if err := ensureOriginURL(ctx, src); err != nil {
		return false, err
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

func ensureOriginURL(ctx context.Context, src Source) error {
	wantURL := strings.TrimSpace(src.URL)
	if wantURL == "" {
		return nil
	}

	currentURL, err := gitOutput(ctx, "", "-C", src.RepoPath, "remote", "get-url", "origin")
	if err != nil {
		if _, addErr := gitOutput(ctx, "", "-C", src.RepoPath, "remote", "add", "origin", wantURL); addErr != nil {
			return addErr
		}
		return nil
	}

	if strings.TrimSpace(currentURL) == wantURL {
		return nil
	}

	_, err = gitOutput(ctx, "", "-C", src.RepoPath, "remote", "set-url", "origin", wantURL)
	return err
}

func ResolveCommit(ctx context.Context, src Source, ref string) (string, error) {
	for _, candidate := range resolveCommitCandidates(ref) {
		commit, err := gitOutput(ctx, "", "-C", src.RepoPath, "rev-parse", candidate+"^{commit}")
		if err == nil {
			return commit, nil
		}
	}

	return "", fmt.Errorf("could not resolve ref %q to a commit", ref)
}

func InferDefaultRef(ctx context.Context, url string) (string, error) {
	output, err := gitOutput(ctx, "", "ls-remote", "--symref", url, "HEAD")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "ref:" || fields[2] != "HEAD" {
			continue
		}
		ref := strings.TrimPrefix(fields[1], "refs/heads/")
		if strings.TrimSpace(ref) != "" {
			return ref, nil
		}
	}

	return "", errors.New("could not determine default ref from remote HEAD")
}

func defaultRemoteStatus(ctx context.Context, repoPath string) (string, string) {
	ref, err := gitOutput(ctx, "", "-C", repoPath, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", ""
	}

	commit, err := gitOutput(ctx, "", "-C", repoPath, "rev-parse", ref+"^{commit}")
	if err != nil {
		return "", ""
	}

	return strings.TrimPrefix(ref, "refs/remotes/origin/"), commit
}

func resolveCommitCandidates(ref string) []string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}

	candidates := []string{}
	seen := map[string]struct{}{}
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	if looksLikeCommit(ref) {
		add(ref)
	}
	if strings.HasPrefix(ref, "refs/") {
		add(ref)
	}

	add("refs/remotes/origin/" + ref)
	add("refs/tags/" + ref)
	add(ref)

	return candidates
}

func looksLikeCommit(ref string) bool {
	if len(ref) < 7 || len(ref) > 40 {
		return false
	}
	for _, r := range ref {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

func ListFilesAtCommit(ctx context.Context, src Source, commit string) ([]string, error) {
	output, err := gitOutput(ctx, "", "-C", src.RepoPath, "ls-tree", "-r", "--name-only", commit)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(output) == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}

func RepoBasename(src Source) string {
	for _, candidate := range []string{src.URL, src.RepoPath, src.Alias} {
		if name := repoBasename(candidate); name != "" {
			return name
		}
	}
	return ""
}

func repoBasename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" {
		return trimGitSuffix(path.Base(parsed.Path))
	}

	if strings.Contains(value, "@") && strings.Contains(value, ":") && !strings.Contains(value, "://") {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) == 2 {
			return trimGitSuffix(path.Base(parts[1]))
		}
	}

	return trimGitSuffix(filepath.Base(value))
}

func sourceIdentityLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" {
		switch parsed.Scheme {
		case "file":
			return trimGitSuffix(filepath.Base(parsed.Path))
		default:
			repoPath := trimGitSuffix(path.Clean(parsed.Path))
			repoPath = strings.Trim(repoPath, "/")
			if repoPath != "" {
				return strings.ReplaceAll(repoPath, "/", "-")
			}
		}
	}

	if strings.Contains(value, "@") && strings.Contains(value, ":") && !strings.Contains(value, "://") {
		parts := strings.SplitN(value, ":", 2)
		repoPart := strings.Trim(parts[1], "/")
		repoPart = trimGitSuffix(path.Clean(repoPart))
		if repoPart != "" {
			return strings.ReplaceAll(repoPart, "/", "-")
		}
	}

	return trimGitSuffix(filepath.Base(value))
}

func canonicalSourceIdentity(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" {
		switch parsed.Scheme {
		case "file":
			if parsed.Path != "" {
				return canonicalLocalPath(parsed.Path)
			}
		default:
			host := strings.ToLower(parsed.Hostname())
			repoPath := trimGitSuffix(path.Clean(parsed.Path))
			repoPath = strings.TrimPrefix(repoPath, "/")
			if host != "" && repoPath != "" {
				return host + "/" + repoPath
			}
		}
	}

	if strings.Contains(value, "@") && strings.Contains(value, ":") && !strings.Contains(value, "://") {
		parts := strings.SplitN(value, ":", 2)
		hostPart := parts[0]
		repoPart := parts[1]
		if at := strings.LastIndex(hostPart, "@"); at >= 0 {
			hostPart = hostPart[at+1:]
		}
		hostPart = strings.ToLower(strings.TrimSpace(hostPart))
		repoPart = strings.TrimPrefix(trimGitSuffix(path.Clean(repoPart)), "/")
		if hostPart != "" && repoPart != "" {
			return hostPart + "/" + repoPart
		}
	}

	return canonicalLocalPath(value)
}

func canonicalLocalPath(value string) string {
	if abs, err := filepath.Abs(value); err == nil {
		value = abs
	}
	if eval, err := filepath.EvalSymlinks(value); err == nil {
		value = eval
	}
	return filepath.Clean(value)
}

func trimGitSuffix(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, ".git")
	if value == "." || value == string(filepath.Separator) {
		return ""
	}
	return value
}

func sanitizePathComponent(value string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(value) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "source"
	}
	return result
}

func EnsureWorktree(ctx context.Context, src Source, path string, commit string) (bool, error) {
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return false, fmt.Errorf("worktree path exists and is not a directory: %s", path)
		}

		head, err := gitOutput(ctx, "", "-C", path, "rev-parse", "HEAD")
		if err != nil {
			return false, fmt.Errorf("invalid worktree at %s: %w", path, err)
		}
		if head != commit {
			return false, fmt.Errorf("worktree at %s points to %s, want %s", path, head, commit)
		}
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}

	_, err := gitOutput(ctx, "", "-C", src.RepoPath, "worktree", "add", "--detach", path, commit)
	if err != nil {
		return false, err
	}
	return true, nil
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
