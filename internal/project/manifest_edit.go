package project

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml/ast"

	"github.com/mattgiles/skills/internal/yamlx"
)

func UpsertManifestSourceAt(path string, alias string, source ManifestSource) error {
	file, _, err := yamlx.ParseFile(path)
	if err != nil {
		return err
	}

	root, err := yamlx.RootMapping(file)
	if err != nil {
		return err
	}

	value := yamlx.FindMappingValue(root, "sources")
	if value == nil {
		if err := yamlx.MergeAtRoot(path, "sources: {}\n"); err != nil {
			return err
		}
		file, _, err = yamlx.ParseFile(path)
		if err != nil {
			return err
		}
		root, err = yamlx.RootMapping(file)
		if err != nil {
			return err
		}
		value = yamlx.FindMappingValue(root, "sources")
	}
	if value == nil {
		return fmt.Errorf("manifest %s is missing sources", path)
	}

	target, err := ensureSourceMapping(value)
	if err != nil {
		return err
	}

	if target.IsFlowStyle && len(target.Values) == 0 {
		target.SetIsFlowStyle(false)
	}

	flowStyle := sourceEntryFlowStyle(target, alias)
	snippet, err := manifestSourceSnippet(alias, source, flowStyle)
	if err != nil {
		return err
	}

	update, err := yamlx.ParseMapping(snippet)
	if err != nil {
		return err
	}

	if existing := yamlx.FindMappingValue(target, alias); existing != nil {
		existing.Value = update.Values[0].Value
	} else {
		target.Merge(update)
	}
	return yamlx.WriteASTFile(path, file)
}

func AppendManifestSkillAt(path string, skill ManifestSkill) error {
	if strings.TrimSpace(skill.Source) == "" {
		return fmt.Errorf("skill is missing source")
	}
	if strings.TrimSpace(skill.Name) == "" {
		return fmt.Errorf("skill in source %q is missing name", skill.Source)
	}

	file, _, err := yamlx.ParseFile(path)
	if err != nil {
		return err
	}

	root, err := yamlx.RootMapping(file)
	if err != nil {
		return err
	}

	value := yamlx.FindMappingValue(root, "skills")
	if value == nil {
		snippet := "skills:\n"
		if err := yamlx.MergeAtRoot(path, snippet); err != nil {
			return err
		}
		file, _, err = yamlx.ParseFile(path)
		if err != nil {
			return err
		}
		root, err = yamlx.RootMapping(file)
		if err != nil {
			return err
		}
		value = yamlx.FindMappingValue(root, "skills")
	}
	if value == nil {
		return fmt.Errorf("manifest %s is missing skills", path)
	}

	target, err := ensureSkillSequence(value)
	if err != nil {
		return err
	}

	if target.IsFlowStyle && len(target.Values) == 0 {
		target.SetIsFlowStyle(false)
	}

	update, err := yamlx.ParseSequence(skillSnippet(skill, target.IsFlowStyle))
	if err != nil {
		return err
	}
	target.Merge(update)
	return yamlx.WriteASTFile(path, file)
}

// SetManifestSkillPathAt sets (or replaces) the `path:` selector on the skill
// entry identified by (source, name), preserving comments and formatting in
// the rest of the file. It fails when no such entry exists.
func SetManifestSkillPathAt(path string, source string, name string, skillPath string) error {
	file, _, err := yamlx.ParseFile(path)
	if err != nil {
		return err
	}

	root, err := yamlx.RootMapping(file)
	if err != nil {
		return err
	}

	value := yamlx.FindMappingValue(root, "skills")
	if value == nil {
		return fmt.Errorf("skill %s/%s not found in manifest", source, name)
	}
	seq, err := ensureSkillSequence(value)
	if err != nil {
		return err
	}

	for _, entry := range seq.Values {
		mapping, ok := entry.(*ast.MappingNode)
		if !ok {
			continue
		}
		if !skillEntryMatches(mapping, source, name) {
			continue
		}

		update, err := yamlx.ParseMapping(fmt.Sprintf("path: %q\n", skillPath))
		if err != nil {
			return err
		}
		if existing := yamlx.FindMappingValue(mapping, "path"); existing != nil {
			existing.Value = update.Values[0].Value
		} else {
			mapping.Merge(update)
		}
		return yamlx.WriteASTFile(path, file)
	}

	return fmt.Errorf("skill %s/%s not found in manifest", source, name)
}

func skillEntryMatches(mapping *ast.MappingNode, source string, name string) bool {
	sourceValue := yamlx.FindMappingValue(mapping, "source")
	nameValue := yamlx.FindMappingValue(mapping, "name")
	if sourceValue == nil || nameValue == nil {
		return false
	}
	return yamlx.ScalarString(sourceValue.Value) == source && yamlx.ScalarString(nameValue.Value) == name
}

func ensureSourceMapping(value *ast.MappingValueNode) (*ast.MappingNode, error) {
	if value.Value == nil {
		update, err := yamlx.ParseMapping("{}")
		if err != nil {
			return nil, err
		}
		value.Value = update
		return update, nil
	}

	mapping, ok := value.Value.(*ast.MappingNode)
	if ok {
		return mapping, nil
	}

	update, err := yamlx.ParseMapping("{}")
	if err != nil {
		return nil, err
	}
	if err := value.Replace(update); err != nil {
		return nil, err
	}
	return update, nil
}

func ensureSkillSequence(value *ast.MappingValueNode) (*ast.SequenceNode, error) {
	if value.Value == nil {
		update, err := yamlx.ParseSequence("[]")
		if err != nil {
			return nil, err
		}
		value.Value = update
		return update, nil
	}

	seq, ok := value.Value.(*ast.SequenceNode)
	if ok {
		return seq, nil
	}

	update, err := yamlx.ParseSequence("[]")
	if err != nil {
		return nil, err
	}
	if err := value.Replace(update); err != nil {
		return nil, err
	}
	return update, nil
}

func manifestSourceSnippet(alias string, source ManifestSource, flowStyle bool) (string, error) {
	data, err := yamlx.Marshal(map[string]ManifestSource{alias: source})
	if err != nil {
		return "", err
	}

	snippet := string(data)
	if !flowStyle {
		return snippet, nil
	}

	fields := make([]string, 0, 4)
	if strings.TrimSpace(source.URL) != "" {
		fields = append(fields, fmt.Sprintf("url: %q", source.URL))
	}
	fields = append(fields, fmt.Sprintf("ref: %q", source.Ref))
	if len(source.Include) > 0 {
		fields = append(fields, "include: "+flowStringList(source.Include))
	}
	if len(source.Exclude) > 0 {
		fields = append(fields, "exclude: "+flowStringList(source.Exclude))
	}
	return fmt.Sprintf("%s: {%s}\n", alias, strings.Join(fields, ", ")), nil
}

func flowStringList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func skillSnippet(skill ManifestSkill, flowStyle bool) string {
	if flowStyle {
		if skill.Path != "" {
			return fmt.Sprintf("[{source: %q, name: %q, path: %q}]\n", skill.Source, skill.Name, skill.Path)
		}
		return fmt.Sprintf("[{source: %q, name: %q}]\n", skill.Source, skill.Name)
	}

	data, err := yamlx.Marshal([]ManifestSkill{skill})
	if err != nil {
		fallback := fmt.Sprintf("- source: %q\n  name: %q\n", skill.Source, skill.Name)
		if skill.Path != "" {
			fallback += fmt.Sprintf("  path: %q\n", skill.Path)
		}
		return fallback
	}
	return string(data)
}

func sourceEntryFlowStyle(sources *ast.MappingNode, alias string) bool {
	if existing := yamlx.FindMappingValue(sources, alias); existing != nil {
		if mapping, ok := existing.Value.(*ast.MappingNode); ok {
			return mapping.IsFlowStyle
		}
	}

	for _, value := range sources.Values {
		if value == nil {
			continue
		}
		mapping, ok := value.Value.(*ast.MappingNode)
		if !ok {
			continue
		}
		return mapping.IsFlowStyle
	}
	return false
}
