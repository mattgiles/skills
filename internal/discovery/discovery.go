package discovery

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type DiscoveredSkill struct {
	SourceAlias  string
	Name         string
	Path         string
	RelativePath string
}

func Discover(sourceAlias string, repoPath string) ([]DiscoveredSkill, error) {
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

	return DiscoverFromPaths(sourceAlias, repoPath, paths), nil
}

func DiscoverFromPaths(sourceAlias string, repoPath string, paths []string) []DiscoveredSkill {
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

		skills = append(skills, DiscoveredSkill{
			SourceAlias:  sourceAlias,
			Name:         filepath.Base(relativePath),
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
