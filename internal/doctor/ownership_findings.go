package doctor

import (
	"context"

	"github.com/mattgiles/skills/internal/project"
)

func inspectProjectOwnershipFindings(ctx context.Context, projectDir string) []Finding {
	findings := []Finding{}

	ownership, err := project.InspectProjectOwnershipContext(ctx, projectDir)
	if err != nil {
		return append(findings, Finding{
			Section:  SectionGit,
			Severity: SeverityError,
			Code:     "git-inspection-failed",
			Subject:  projectDir,
			Message:  err.Error(),
			Path:     projectDir,
		})
	}

	if !ownership.GitAvailable {
		return append(findings, Finding{
			Section:  SectionGit,
			Severity: SeverityInfo,
			Code:     "git-unavailable",
			Subject:  ownership.GitignorePath,
			Message:  "git-aware ownership checks were skipped because git is not available",
			Path:     ownership.GitignorePath,
		})
	}

	if !ownership.InGitRepo {
		findings = append(findings, Finding{
			Section:  SectionGit,
			Severity: SeverityInfo,
			Code:     "git-repo-not-found",
			Subject:  ownership.GitignorePath,
			Message:  "no enclosing Git repo found; using the project-local .gitignore",
			Path:     ownership.GitignorePath,
		})
	} else {
		findings = append(findings, Finding{
			Section:  SectionGit,
			Severity: SeverityInfo,
			Code:     "git-repo-root",
			Subject:  ownership.GitRoot,
			Message:  "using the enclosing Git repo root for ignore management",
			Path:     ownership.GitignorePath,
		})
	}

	if len(ownership.MissingRules) > 0 {
		findings = append(findings, Finding{
			Section:  SectionGit,
			Severity: SeverityWarn,
			Code:     "ignore-rules-missing",
			Subject:  ownership.GitignorePath,
			Message:  "missing ignore rules for managed runtime artifacts",
			Hint:     "run skills init",
			Path:     ownership.GitignorePath,
		})
	}

	for _, trackedPath := range ownership.TrackedPaths {
		findings = append(findings, Finding{
			Section:  SectionGit,
			Severity: SeverityError,
			Code:     "tracked-managed-path",
			Subject:  trackedPath,
			Message:  "managed runtime artifacts should not be tracked by Git",
			Hint:     "move or remove the tracked content from managed paths, then run skills init",
			Path:     trackedPath,
		})
	}

	return findings
}
