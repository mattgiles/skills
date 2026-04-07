package project

import "testing"

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
