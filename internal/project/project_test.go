package project

import (
	"path/filepath"
	"testing"
)

func TestValidateManifestRejectsDuplicateSkills(t *testing.T) {
	manifest := Manifest{
		Sources: map[string]ManifestSource{
			"repo-one": {Ref: "main"},
		},
		Skills: []ManifestSkill{
			{Source: "repo-one", Name: "analytics"},
			{Source: "repo-one", Name: "analytics"},
		},
	}

	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("ValidateManifest() expected duplicate skill error")
	}
}

func TestProjectIDIsStable(t *testing.T) {
	projectDir := t.TempDir()

	first, err := ProjectID(projectDir)
	if err != nil {
		t.Fatalf("ProjectID() error = %v", err)
	}
	second, err := ProjectID(projectDir)
	if err != nil {
		t.Fatalf("ProjectID() error = %v", err)
	}

	if first != second {
		t.Fatalf("ProjectID() mismatch: %q vs %q", first, second)
	}
}

func TestStateRoundTripIncludesSourceState(t *testing.T) {
	projectDir := t.TempDir()

	state := State{
		Sources: []SourceState{
			{Source: "repo-one", Ref: "main", ResolvedCommit: "abc123"},
		},
		SkillLinks: []ManagedLink{
			{Path: "/tmp/skill", Target: "/tmp/target", Source: "repo-one", Skill: "analytics"},
		},
		ClaudeLinks: []ManagedLink{
			{Path: "/tmp/claude", Target: "/tmp/skill", Source: "repo-one", Skill: "analytics"},
		},
	}

	if err := SaveState(projectDir, state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	loaded, err := LoadState(projectDir)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	if len(loaded.Sources) != 1 {
		t.Fatalf("len(loaded.Sources) = %d, want 1", len(loaded.Sources))
	}
	if loaded.Sources[0].ResolvedCommit != "abc123" {
		t.Fatalf("ResolvedCommit = %q, want %q", loaded.Sources[0].ResolvedCommit, "abc123")
	}
	if len(loaded.SkillLinks) != 1 {
		t.Fatalf("len(loaded.SkillLinks) = %d, want 1", len(loaded.SkillLinks))
	}
	if len(loaded.ClaudeLinks) != 1 {
		t.Fatalf("len(loaded.ClaudeLinks) = %d, want 1", len(loaded.ClaudeLinks))
	}
	if got := StatePath(projectDir); got != filepath.Join(projectDir, ".agents", "state.yaml") {
		t.Fatalf("StatePath() = %q", got)
	}
}
