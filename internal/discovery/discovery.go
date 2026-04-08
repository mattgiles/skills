package discovery

import (
	"context"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mattgiles/skills/internal/gitrepo"
	"github.com/mattgiles/skills/internal/source"
)

type DiscoveredSkill struct {
	SourceAlias  string
	Name         string
	Path         string
	RelativePath string
}

func Discover(sourceAlias string, repoPath string) ([]DiscoveredSkill, error) {
	if tracked, ok, err := discoverTrackedPaths(repoPath); err != nil {
		return nil, err
	} else if ok {
		return DiscoverFromPaths(sourceAlias, repoPath, tracked, filepath.Base(repoPath)), nil
	}

	return discoverFromFilesystem(sourceAlias, repoPath)
}

func DiscoverAtCommit(ctx context.Context, src source.Source, rootPath string, commit string) ([]DiscoveredSkill, error) {
	paths, err := source.ListFilesAtCommit(ctx, src, commit)
	if err != nil {
		return nil, err
	}
	return DiscoverFromPaths(src.Alias, rootPath, paths, source.RepoBasename(src)), nil
}

func discoverTrackedPaths(repoPath string) ([]string, bool, error) {
	info, err := gitrepo.Discover(context.Background(), repoPath)
	if err != nil {
		return nil, false, err
	}
	if info.Root == "" {
		return nil, false, nil
	}

	tracked, err := gitrepo.ListTracked(context.Background(), info.Root, nil)
	if err != nil {
		return nil, false, err
	}

	base := relativeTrackedBase(info.Root, repoPath)
	if base == "" {
		return tracked, true, nil
	}

	normalizedBase := filepath.ToSlash(base)
	paths := make([]string, 0, len(tracked))
	for _, path := range tracked {
		normalizedPath := filepath.ToSlash(path)
		prefix := normalizedBase + "/"
		if strings.HasPrefix(normalizedPath, prefix) {
			paths = append(paths, strings.TrimPrefix(normalizedPath, prefix))
		}
	}
	return paths, true, nil
}

func relativeTrackedBase(repoRoot string, repoPath string) string {
	resolvedRoot, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		resolvedRoot = repoRoot
	}
	resolvedPath, err := filepath.EvalSymlinks(repoPath)
	if err != nil {
		resolvedPath = repoPath
	}

	rel, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || rel == "." {
		return ""
	}
	return rel
}

func discoverFromFilesystem(sourceAlias string, repoPath string) ([]DiscoveredSkill, error) {
	paths := make([]string, 0)

	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		if d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}

		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return DiscoverFromPaths(sourceAlias, repoPath, paths, filepath.Base(repoPath)), nil
}

func DiscoverFromPaths(sourceAlias string, repoPath string, paths []string, rootSkillName string) []DiscoveredSkill {
	skills := make([]DiscoveredSkill, 0)
	seen := map[string]struct{}{}

	for _, path := range paths {
		cleanPath := filepath.Clean(path)
		if filepath.Base(cleanPath) != "SKILL.md" {
			continue
		}

		dir := filepath.Dir(cleanPath)
		relativePath := dir
		absolutePath := dir

		if strings.TrimSpace(repoPath) != "" {
			if filepath.IsAbs(cleanPath) {
				absDir := filepath.Dir(cleanPath)
				rel, err := filepath.Rel(repoPath, absDir)
				if err != nil {
					continue
				}
				relativePath = rel
				absolutePath = absDir
			} else {
				relativePath = dir
				absolutePath = filepath.Join(repoPath, dir)
			}
		}

		if _, ok := seen[relativePath]; ok {
			continue
		}
		seen[relativePath] = struct{}{}

		name := filepath.Base(relativePath)
		if isRepoRootPath(relativePath) {
			name = rootSkillName
		}

		skills = append(skills, DiscoveredSkill{
			SourceAlias:  sourceAlias,
			Name:         name,
			Path:         absolutePath,
			RelativePath: relativePath,
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].RelativePath < skills[j].RelativePath
	})

	return skills
}

func isRepoRootPath(relativePath string) bool {
	relativePath = filepath.Clean(strings.TrimSpace(relativePath))
	return relativePath == "." || relativePath == ""
}
