package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var aliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

type Config struct {
	RepoRoot     string                  `yaml:"repo_root,omitempty"`
	WorktreeRoot string                  `yaml:"worktree_root,omitempty"`
	Agents       map[string]AgentConfig  `yaml:"agents,omitempty"`
	Sources      map[string]SourceConfig `yaml:"sources,omitempty"`
}

type SourceConfig struct {
	URL string `yaml:"url"`
}

type AgentConfig struct {
	SkillsDir string `yaml:"skills_dir"`
}

func DefaultConfig() Config {
	return Config{
		RepoRoot:     defaultRepoRootValue(),
		WorktreeRoot: defaultWorktreeRootValue(),
		Agents: map[string]AgentConfig{
			"claude": {SkillsDir: "~/.claude/skills"},
			"codex":  {SkillsDir: "~/.codex/skills"},
		},
		Sources: map[string]SourceConfig{},
	}
}

func DefaultConfigPath() (string, error) {
	configHome, err := xdgConfigHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(configHome, "skills", "config.yaml"), nil
}

func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	resolvedPath, err := ExpandPath(path)
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(resolvedPath)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", resolvedPath, err)
	}

	ensureDefaults(&cfg)
	return cfg, nil
}

func Save(path string, cfg Config) error {
	ensureDefaults(&cfg)

	resolvedPath, err := ExpandPath(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(resolvedPath, data, 0o644)
}

func RepoRootPath(cfg Config) (string, error) {
	root := cfg.RepoRoot
	if strings.TrimSpace(root) == "" {
		root = defaultRepoRootValue()
	}
	return ExpandPath(root)
}

func WorktreeRootPath(cfg Config) (string, error) {
	root := cfg.WorktreeRoot
	if strings.TrimSpace(root) == "" {
		root = defaultWorktreeRootValue()
	}
	return ExpandPath(root)
}

func ExpandPath(path string) (string, error) {
	return ResolvePath("", path)
}

func ResolvePath(baseDir string, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}

	expanded := os.ExpandEnv(path)
	if expanded == "~" || strings.HasPrefix(expanded, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		if expanded == "~" {
			expanded = home
		} else {
			expanded = filepath.Join(home, expanded[2:])
		}
	}

	if !filepath.IsAbs(expanded) && strings.TrimSpace(baseDir) != "" {
		expanded = filepath.Join(baseDir, expanded)
	}

	return filepath.Abs(expanded)
}

func ValidateAlias(alias string) error {
	if !aliasPattern.MatchString(alias) {
		return fmt.Errorf("invalid alias %q: use lowercase letters, numbers, '-' or '_'", alias)
	}
	return nil
}

func ensureDefaults(cfg *Config) {
	if cfg.RepoRoot == "" {
		cfg.RepoRoot = defaultRepoRootValue()
	}
	if cfg.WorktreeRoot == "" {
		cfg.WorktreeRoot = defaultWorktreeRootValue()
	}
	if cfg.Agents == nil {
		cfg.Agents = DefaultConfig().Agents
	}
	if cfg.Sources == nil {
		cfg.Sources = map[string]SourceConfig{}
	}
}

func defaultRepoRootValue() string {
	if value := os.Getenv("XDG_DATA_HOME"); strings.TrimSpace(value) != "" {
		return filepath.Join(value, "skills", "repos")
	}
	return "~/.local/share/skills/repos"
}

func defaultWorktreeRootValue() string {
	if value := os.Getenv("XDG_DATA_HOME"); strings.TrimSpace(value) != "" {
		return filepath.Join(value, "skills", "worktrees")
	}
	return "~/.local/share/skills/worktrees"
}

func xdgConfigHome() (string, error) {
	if value := os.Getenv("XDG_CONFIG_HOME"); strings.TrimSpace(value) != "" {
		return ExpandPath(value)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config"), nil
}
