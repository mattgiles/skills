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

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/yamlx"
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
	if err := yamlx.Unmarshal(data, &cfg, yamlx.DecodeOptions{Strict: true}); err != nil {
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

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return yamlx.WriteValueFile(path, &cfg)
	} else if err != nil {
		return err
	}

	file, _, err := yamlx.ParseFile(path)
	if err != nil {
		return err
	}

	root, err := yamlx.RootMapping(file)
	if err != nil {
		return err
	}

	cacheValue := yamlx.FindMappingValue(root, "cache")
	if cacheValue == nil {
		update, err := yamlx.ParseMapping("cache: {}\n")
		if err != nil {
			return err
		}
		root.Merge(update)
		cacheValue = yamlx.FindMappingValue(root, "cache")
	}
	if cacheValue == nil {
		return fmt.Errorf("local config %s is missing cache", path)
	}

	cacheMapping, err := ensureSourceMapping(cacheValue)
	if err != nil {
		return err
	}

	update, err := yamlx.ParseMapping(fmt.Sprintf("mode: %q\n", cfg.Cache.Mode))
	if err != nil {
		return err
	}
	if existing := yamlx.FindMappingValue(cacheMapping, "mode"); existing != nil {
		existing.Value = update.Values[0].Value
	} else {
		cacheMapping.Merge(update)
	}

	return yamlx.WriteASTFile(path, file)
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
	if err := yamlx.Unmarshal(data, &manifest, yamlx.DecodeOptions{Strict: true}); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	ensureManifestDefaults(&manifest)
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

// FragmentDirPath returns the manifest fragment directory for a project.
// External tools write committed `.agents/manifest.d/*.yaml` files there; the
// CLI merges them with the main manifest when loading effective read-time state.
func FragmentDirPath(projectDir string) string {
	return filepath.Join(projectDir, ".agents", "manifest.d")
}

// LoadFragments loads every `*.yaml` manifest fragment under the project's
// fragment directory in lexicographic filename order. A missing directory
// yields (nil, nil); directories and non-`.yaml` entries are skipped.
//
// Fragments are parsed with relaxed validation: a fragment may legitimately
// declare a skill under a source defined in the main manifest (or another
// fragment), so the per-file source-existence check is deferred to the merged
// manifest's ValidateManifest (see LoadEffectiveManifest). Syntax errors and
// unknown keys are still rejected per-file.
func LoadFragments(projectDir string) ([]Manifest, error) {
	dir := FragmentDirPath(projectDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	fragments := make([]Manifest, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		fragment, err := loadFragmentAt(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("load fragment %s: %w", entry.Name(), err)
		}
		fragments = append(fragments, fragment)
	}
	return fragments, nil
}

// loadFragmentAt reads and strict-parses a single fragment file without running
// the full ValidateManifest (which would reject cross-file source references).
func loadFragmentAt(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, fmt.Errorf("manifest not found: %s", path)
		}
		return Manifest{}, err
	}

	manifest := DefaultManifest()
	if err := yamlx.Unmarshal(data, &manifest, yamlx.DecodeOptions{Strict: true}); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	ensureManifestDefaults(&manifest)
	return manifest, nil
}

// MergeManifests merges base with the given fragments into a single effective
// manifest. Source aliases must be unique across the base and all fragments;
// a collision is an error. Duplicate (source,name) skill pairs are silently
// deduped with the first occurrence winning (base before fragments, fragments
// in LoadFragments order).
func MergeManifests(base Manifest, fragments ...Manifest) (Manifest, error) {
	ensureManifestDefaults(&base)

	merged := Manifest{
		Sources: make(map[string]ManifestSource, len(base.Sources)),
		Skills:  make([]ManifestSkill, 0, len(base.Skills)),
	}
	for alias, src := range base.Sources {
		merged.Sources[alias] = src
	}

	seen := map[string]struct{}{}
	for _, sk := range base.Skills {
		key := sk.Source + "\x00" + sk.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged.Skills = append(merged.Skills, sk)
	}

	for _, fragment := range fragments {
		ensureManifestDefaults(&fragment)
		for alias, src := range fragment.Sources {
			if _, ok := merged.Sources[alias]; ok {
				return Manifest{}, fmt.Errorf("duplicate source alias %q across manifest and fragments", alias)
			}
			merged.Sources[alias] = src
		}
		for _, sk := range fragment.Skills {
			key := sk.Source + "\x00" + sk.Name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged.Skills = append(merged.Skills, sk)
		}
	}

	return merged, nil
}

// LoadEffectiveManifest loads the main manifest, merges every fragment under
// `.agents/manifest.d/`, and validates the merged result. Per-file validation
// already happens during loading; the final ValidateManifest catches cross-file
// issues such as a fragment skill referencing a source declared elsewhere.
func LoadEffectiveManifest(projectDir string) (Manifest, error) {
	manifest, err := LoadManifest(projectDir)
	if err != nil {
		return Manifest{}, err
	}
	fragments, err := LoadFragments(projectDir)
	if err != nil {
		return Manifest{}, err
	}
	merged, err := MergeManifests(manifest, fragments...)
	if err != nil {
		return Manifest{}, err
	}
	if err := ValidateManifest(merged); err != nil {
		return Manifest{}, err
	}
	return merged, nil
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

	return yamlx.WriteValueFile(path, &manifest)
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
	if err := yamlx.Unmarshal(data, &state, yamlx.DecodeOptions{}); err != nil {
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

	return yamlx.WriteValueFile(path, &state)
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
