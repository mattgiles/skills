package doctor

import (
	"strings"
	"testing"

	"github.com/mattgiles/skills/internal/project"
)

func TestLinkFindingAmbiguousSkillHint(t *testing.T) {
	link := project.LinkReport{
		Source:  "repo-one",
		Skill:   "bedrock",
		Status:  "ambiguous-skill",
		Message: "multiple skills share this directory name; set path: to one of: a/bedrock, b/bedrock",
	}

	findings := linkFinding(SectionSkills, link, ScopeProject)
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}
	if findings[0].Message != link.Message {
		t.Fatalf("message = %q, want link message", findings[0].Message)
	}
	wantHint := "run skills add repo-one bedrock --path <candidate> with one of the listed paths, or scope the source with include:/exclude:"
	if findings[0].Hint != wantHint {
		t.Fatalf("hint = %q, want %q", findings[0].Hint, wantHint)
	}

	global := linkFinding(SectionSkills, link, ScopeGlobal)
	if !strings.Contains(global[0].Hint, "skills add --global repo-one bedrock --path <candidate>") {
		t.Fatalf("global hint = %q", global[0].Hint)
	}
}

func TestLinkFindingMissingSkillUsesLinkMessage(t *testing.T) {
	withMessage := linkFinding(SectionSkills, project.LinkReport{
		Source:  "repo-one",
		Skill:   "bedrock",
		Status:  "missing-skill",
		Message: `no skill directory at path "does/not/exist"`,
	}, ScopeProject)
	if withMessage[0].Message != `no skill directory at path "does/not/exist"` {
		t.Fatalf("message = %q", withMessage[0].Message)
	}

	withoutMessage := linkFinding(SectionSkills, project.LinkReport{
		Source: "repo-one",
		Skill:  "bedrock",
		Status: "missing-skill",
	}, ScopeProject)
	if withoutMessage[0].Message != "declared skill name was not found in the source" {
		t.Fatalf("default message = %q", withoutMessage[0].Message)
	}
	if !strings.Contains(withoutMessage[0].Hint, "skills skill list --source repo-one") {
		t.Fatalf("hint = %q", withoutMessage[0].Hint)
	}
}
