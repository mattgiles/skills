package project

import (
	"path/filepath"
	"testing"
)

func TestValidateManifestRejectsDuplicateSkills(t *testing.T) {
	manifest := Manifest{
		Sources: map[string]ProjectSource{
			"repo-one": {Ref: "main"},
		},
		Skills: []ProjectSkill{
			{Source: "repo-one", Name: "analytics", Agents: []string{"codex"}},
			{Source: "repo-one", Name: "analytics", Agents: []string{"claude"}},
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
		Sources: []ProjectSourceState{
			{Source: "repo-one", Ref: "main", ResolvedCommit: "abc123"},
		},
		Links: []ManagedLink{
			{Path: "/tmp/skill", Target: "/tmp/target", Source: "repo-one", Skill: "analytics", Agent: "codex"},
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
	if len(loaded.Links) != 1 {
		t.Fatalf("len(loaded.Links) = %d, want 1", len(loaded.Links))
	}
	if got := StatePath(projectDir); got != filepath.Join(projectDir, ".skills", "state.yaml") {
		t.Fatalf("StatePath() = %q", got)
	}
}
