package yamlx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

type DecodeOptions struct {
	Strict bool
}

func Unmarshal(data []byte, dst any, options DecodeOptions) error {
	decodeOptions := []yaml.DecodeOption{}
	if options.Strict {
		decodeOptions = append(decodeOptions, yaml.DisallowUnknownField())
	}

	if err := yaml.UnmarshalWithOptions(data, dst, decodeOptions...); err != nil {
		return formatError(data, err)
	}
	return nil
}

func Marshal(v any) ([]byte, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func WriteValueFile(path string, v any) error {
	data, err := Marshal(v)
	if err != nil {
		return err
	}
	return WriteFile(path, data)
}

func WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Parse(data []byte) (*ast.File, error) {
	file, err := parser.ParseBytes(data, parser.ParseComments)
	if err != nil {
		return nil, formatError(data, err)
	}
	return file, nil
}

func ParseFile(path string) (*ast.File, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	file, err := Parse(data)
	if err != nil {
		return nil, nil, err
	}
	return file, data, nil
}

func ParseMapping(snippet string) (*ast.MappingNode, error) {
	return mappingFromBytes([]byte(snippet))
}

func ParseSequence(snippet string) (*ast.SequenceNode, error) {
	file, err := Parse([]byte(snippet))
	if err != nil {
		return nil, err
	}

	node, err := documentBody(file)
	if err != nil {
		return nil, err
	}

	seq, ok := node.(*ast.SequenceNode)
	if !ok {
		return nil, fmt.Errorf("expected YAML sequence, got %s", node.Type())
	}
	return seq, nil
}

func RootMapping(file *ast.File) (*ast.MappingNode, error) {
	node, err := documentBody(file)
	if err != nil {
		return nil, err
	}

	mapping, ok := node.(*ast.MappingNode)
	if !ok {
		return nil, fmt.Errorf("expected YAML mapping, got %s", node.Type())
	}
	return mapping, nil
}

func FindMappingValue(mapping *ast.MappingNode, key string) *ast.MappingValueNode {
	for _, value := range mapping.Values {
		if value == nil || value.Key == nil {
			continue
		}
		if unquoteKey(value.Key.String()) == key {
			return value
		}
	}
	return nil
}

func MappingValue(mapping *ast.MappingNode, key string) (ast.Node, bool) {
	value := FindMappingValue(mapping, key)
	if value == nil {
		return nil, false
	}
	return value.Value, true
}

func WriteASTFile(path string, file *ast.File) error {
	return WriteFile(path, []byte(file.String()))
}

func MergeAtRoot(path string, snippet string) error {
	file, _, err := ParseFile(path)
	if err != nil {
		return err
	}

	root, err := RootMapping(file)
	if err != nil {
		return err
	}

	update, err := ParseMapping(snippet)
	if err != nil {
		return err
	}
	root.Merge(update)
	return WriteASTFile(path, file)
}

func formatError(data []byte, err error) error {
	if err == nil {
		return nil
	}

	message := strings.TrimSpace(yaml.FormatError(err, false, true))
	if message == "" {
		return err
	}
	if len(data) == 0 {
		return errors.New(message)
	}
	return errors.New(message)
}

func documentBody(file *ast.File) (ast.Node, error) {
	if file == nil || len(file.Docs) == 0 {
		return nil, errors.New("empty YAML document")
	}
	if len(file.Docs) != 1 {
		return nil, errors.New("multi-document YAML is not supported")
	}
	if file.Docs[0] == nil || file.Docs[0].Body == nil {
		return nil, errors.New("empty YAML document")
	}
	return file.Docs[0].Body, nil
}

func mappingFromBytes(data []byte) (*ast.MappingNode, error) {
	file, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return RootMapping(file)
}

func unquoteKey(value string) string {
	if len(value) >= 2 {
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			return value[1 : len(value)-1]
		}
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			return value[1 : len(value)-1]
		}
	}
	return value
}
