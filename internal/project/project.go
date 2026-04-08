package project

const (
	ManifestFilename     = ".agents/manifest.yaml"
	StateFilename        = ".agents/state.yaml"
	LocalConfigFilename  = ".agents/local.yaml"
	SkillsDirname        = ".agents/skills"
	ClaudeSkillsDirname  = ".claude/skills"
	CacheDirname         = ".agents/cache"
	RepoCacheDirname     = ".agents/cache/repos"
	WorktreeCacheDirname = ".agents/cache/worktrees"
	homeManifestFilename = "manifest.yaml"
	homeStateFilename    = "state.yaml"
	projectWorkspaceName = "project"
	sharedWorkspaceName  = "home"
	gitignoreBeginMarker = "# BEGIN skills managed runtime artifacts"
	gitignoreEndMarker   = "# END skills managed runtime artifacts"
)

type CacheMode string

const (
	CacheModeLocal  CacheMode = "local"
	CacheModeGlobal CacheMode = "global"
)

type Manifest struct {
	Sources map[string]ManifestSource `yaml:"sources"`
	Skills  []ManifestSkill           `yaml:"skills"`
}

type ManifestSource struct {
	URL string `yaml:"url,omitempty"`
	Ref string `yaml:"ref"`
}

type ManifestSkill struct {
	Source string `yaml:"source"`
	Name   string `yaml:"name"`
}

type LocalConfig struct {
	Cache LocalCacheConfig `yaml:"cache,omitempty"`
}

type LocalCacheConfig struct {
	Mode CacheMode `yaml:"mode,omitempty"`
}

type State struct {
	Sources     []SourceState `yaml:"sources,omitempty"`
	SkillLinks  []ManagedLink `yaml:"skill_links,omitempty"`
	ClaudeLinks []ManagedLink `yaml:"claude_links,omitempty"`
}

type SourceState struct {
	Source         string `yaml:"source"`
	Ref            string `yaml:"ref"`
	ResolvedCommit string `yaml:"resolved_commit"`
}

type ManagedLink struct {
	Path   string `yaml:"path"`
	Target string `yaml:"target"`
	Source string `yaml:"source"`
	Skill  string `yaml:"skill"`
}

type StatusReport struct {
	Sources          []SourceReport
	SkillLinks       []LinkReport
	ClaudeLinks      []LinkReport
	StaleSkillLinks  []ManagedLink
	StaleClaudeLinks []ManagedLink
}

type SourceReport struct {
	Alias          string
	Ref            string
	Commit         string
	PreviousCommit string
	RepoPath       string
	WorktreePath   string
	Status         string
	Message        string
}

type LinkReport struct {
	Source  string
	Skill   string
	Path    string
	Target  string
	Status  string
	Message string
}

type SyncResult struct {
	Sources           []SourceReport
	SkillLinks        []LinkReport
	ClaudeLinks       []LinkReport
	PrunedSkillLinks  []string
	PrunedClaudeLinks []string
	DryRun            bool
}

type UpdateResult struct {
	Sources []SourceReport
	Sync    *SyncResult
	DryRun  bool
}

type SyncOptions struct {
	DryRun bool
}

type UpdateOptions struct {
	SelectedSources []string
	Sync            bool
	DryRun          bool
}

type InitProjectResult struct {
	ManifestPath     string
	ManifestCreated  bool
	LocalConfigPath  string
	LocalConfigSaved bool
	CacheMode        CacheMode
	GitignorePath    string
	GitignoreUpdated bool
}

type ProjectOwnershipReport struct {
	GitAvailable  bool
	InGitRepo     bool
	GitRoot       string
	GitignorePath string
	RequiredRules []string
	MissingRules  []string
	TrackedPaths  []string
}

type ArtifactReport struct {
	HasArtifacts bool
	Paths        []string
}

type ProjectCacheConfig struct {
	Path     string
	Exists   bool
	Implicit bool
	Mode     CacheMode
}

func DefaultManifest() Manifest {
	return Manifest{
		Sources: map[string]ManifestSource{},
		Skills:  []ManifestSkill{},
	}
}

func DefaultLocalConfig() LocalConfig {
	return LocalConfig{
		Cache: LocalCacheConfig{
			Mode: CacheModeLocal,
		},
	}
}
