package discovery

import (
	"io/fs"
	"path/filepath"
	"sort"
)

type DiscoveredSkill struct {
	SourceAlias  string
	Name         string
	Path         string
	RelativePath string
}

func Discover(sourceAlias string, repoPath string) ([]DiscoveredSkill, error) {
	skills := make([]DiscoveredSkill, 0)
	seen := map[string]struct{}{}

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

		dir := filepath.Dir(path)
		rel, err := filepath.Rel(repoPath, dir)
		if err != nil {
			return err
		}

		if _, ok := seen[rel]; ok {
			return nil
		}
		seen[rel] = struct{}{}

		skills = append(skills, DiscoveredSkill{
			SourceAlias:  sourceAlias,
			Name:         filepath.Base(dir),
			Path:         dir,
			RelativePath: rel,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].RelativePath < skills[j].RelativePath
	})

	return skills, nil
}
