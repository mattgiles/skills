package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mattgiles/skills/internal/config"
)

func LoadLocalConfig(projectDir string) (ProjectCacheConfig, error) {
	return LoadLocalConfigAt(LocalConfigPath(projectDir))
}

func LoadLocalConfigAt(path string) (ProjectCacheConfig, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ProjectCacheConfig{
			Path:     path,
			Implicit: true,
			Mode:     CacheModeLocal,
		}, nil
	}
	if err != nil {
		return ProjectCacheConfig{}, err
	}

	cfg := DefaultLocalConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ProjectCacheConfig{}, fmt.Errorf("parse local config %s: %w", path, err)
	}
	ensureLocalConfigDefaults(&cfg)
	if err := ValidateLocalConfig(cfg); err != nil {
		return ProjectCacheConfig{}, err
	}

	return ProjectCacheConfig{
		Path:   path,
		Exists: true,
		Mode:   cfg.Cache.Mode,
	}, nil
}

func SaveLocalConfig(projectDir string, cfg LocalConfig) error {
	return SaveLocalConfigAt(LocalConfigPath(projectDir), cfg)
}

func SaveLocalConfigAt(path string, cfg LocalConfig) error {
	ensureLocalConfigDefaults(&cfg)
	if err := ValidateLocalConfig(cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ValidateLocalConfig(cfg LocalConfig) error {
	switch cfg.Cache.Mode {
	case CacheModeLocal, CacheModeGlobal:
		return nil
	default:
		return fmt.Errorf("invalid cache mode %q: use local or global", cfg.Cache.Mode)
	}
}

func ensureLocalConfigDefaults(cfg *LocalConfig) {
	if cfg.Cache.Mode == "" {
		cfg.Cache.Mode = CacheModeLocal
	}
}

type InitProjectOptions struct {
	CacheMode CacheMode
}

func InitProject(ctx context.Context, projectDir string, options InitProjectOptions) (InitProjectResult, error) {
	cacheMode := options.CacheMode
	if cacheMode == "" {
		current, err := LoadLocalConfig(projectDir)
		if err != nil {
			return InitProjectResult{}, err
		}
		cacheMode = current.Mode
	}
	if cacheMode != CacheModeLocal && cacheMode != CacheModeGlobal {
		return InitProjectResult{}, fmt.Errorf("invalid cache mode %q: use local or global", cacheMode)
	}

	ws, err := projectWorkspace(projectDir, cacheMode)
	if err != nil {
		return InitProjectResult{}, err
	}

	ownership, err := InspectProjectOwnershipContext(ctx, projectDir)
	if err != nil {
		return InitProjectResult{}, err
	}
	if len(ownership.TrackedPaths) > 0 {
		return InitProjectResult{}, fmt.Errorf("managed runtime paths already contain tracked Git content: %s", strings.Join(ownership.TrackedPaths, ", "))
	}
	if err := validateManagedPathTypes(ws); err != nil {
		return InitProjectResult{}, err
	}

	result := InitProjectResult{
		ManifestPath:    ws.ManifestPath,
		LocalConfigPath: ws.LocalConfigPath,
		CacheMode:       cacheMode,
		GitignorePath:   ownership.GitignorePath,
	}

	if _, err := os.Stat(ws.ManifestPath); errors.Is(err, os.ErrNotExist) {
		if err := SaveManifestAt(ws.ManifestPath, DefaultManifest()); err != nil {
			return InitProjectResult{}, err
		}
		result.ManifestCreated = true
	} else if err != nil {
		return InitProjectResult{}, err
	}

	currentLocalConfig, err := LoadLocalConfig(projectDir)
	if err != nil {
		return InitProjectResult{}, err
	}
	if !currentLocalConfig.Exists || currentLocalConfig.Mode != cacheMode {
		if err := SaveLocalConfig(projectDir, LocalConfig{
			Cache: LocalCacheConfig{Mode: cacheMode},
		}); err != nil {
			return InitProjectResult{}, err
		}
		result.LocalConfigSaved = true
	}

	if err := os.MkdirAll(ws.SkillsDir, 0o755); err != nil {
		return InitProjectResult{}, err
	}
	if err := os.MkdirAll(ws.ClaudeSkillsDir, 0o755); err != nil {
		return InitProjectResult{}, err
	}
	if cacheMode == CacheModeLocal {
		if err := os.MkdirAll(ws.RepoRoot, 0o755); err != nil {
			return InitProjectResult{}, err
		}
		if err := os.MkdirAll(ws.WorktreeRoot, 0o755); err != nil {
			return InitProjectResult{}, err
		}
	}
	updated, err := ensureProjectGitignore(ownership)
	if err != nil {
		return InitProjectResult{}, err
	}
	result.GitignoreUpdated = updated
	return result, nil
}

func InitHome(cfg config.Config) (string, error) {
	ws, err := homeWorkspace(cfg)
	if err != nil {
		return "", err
	}
	if err := SaveManifestAt(ws.ManifestPath, DefaultManifest()); err != nil {
		return "", err
	}
	if err := os.MkdirAll(ws.SkillsDir, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(ws.ClaudeSkillsDir, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(ws.RepoRoot, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(ws.WorktreeRoot, 0o755); err != nil {
		return "", err
	}
	return ws.ManifestPath, nil
}

func LoadManifest(projectDir string) (Manifest, error) {
	return LoadManifestAt(ManifestPath(projectDir))
}

func LoadManifestAt(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, fmt.Errorf("manifest not found: %s", path)
		}
		return Manifest{}, err
	}

	manifest := DefaultManifest()
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	ensureManifestDefaults(&manifest)
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func SaveManifest(projectDir string, manifest Manifest) error {
	return SaveManifestAt(ManifestPath(projectDir), manifest)
}

func SaveManifestAt(path string, manifest Manifest) error {
	ensureManifestDefaults(&manifest)
	if err := ValidateManifest(manifest); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&manifest)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func LoadState(projectDir string) (State, error) {
	return LoadStateAt(StatePath(projectDir))
}

func LoadStateAt(path string) (State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}

	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse state %s: %w", path, err)
	}
	return state, nil
}

func SaveState(projectDir string, state State) error {
	return SaveStateAt(StatePath(projectDir), state)
}

func SaveStateAt(path string, state State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ValidateManifest(manifest Manifest) error {
	ensureManifestDefaults(&manifest)

	for alias, src := range manifest.Sources {
		if err := config.ValidateAlias(alias); err != nil {
			return err
		}
		if strings.TrimSpace(src.Ref) == "" {
			return fmt.Errorf("source %q is missing ref", alias)
		}
	}

	seenSkills := map[string]struct{}{}
	for _, skill := range manifest.Skills {
		if strings.TrimSpace(skill.Source) == "" {
			return errors.New("skill is missing source")
		}
		if strings.TrimSpace(skill.Name) == "" {
			return fmt.Errorf("skill in source %q is missing name", skill.Source)
		}
		if _, ok := manifest.Sources[skill.Source]; !ok {
			return fmt.Errorf("skill %q references unknown source %q", skill.Name, skill.Source)
		}

		key := skill.Source + "\x00" + skill.Name
		if _, ok := seenSkills[key]; ok {
			return fmt.Errorf("duplicate skill declaration for %s/%s", skill.Source, skill.Name)
		}
		seenSkills[key] = struct{}{}
	}

	return nil
}

func ProjectID(projectDir string) (string, error) {
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256([]byte(absProjectDir))
	hash := hex.EncodeToString(sum[:])[:12]
	base := sanitizeIDComponent(filepath.Base(absProjectDir))
	if base == "" {
		base = "project"
	}

	return base + "-" + hash, nil
}

func ensureManifestDefaults(manifest *Manifest) {
	if manifest.Sources == nil {
		manifest.Sources = map[string]ManifestSource{}
	}
	if manifest.Skills == nil {
		manifest.Skills = []ManifestSkill{}
	}
}
