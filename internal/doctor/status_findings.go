package doctor

import (
	"github.com/mattgiles/skills/internal/project"
)

func statusFindings(status project.StatusReport, scope Scope) []Finding {
	findings := []Finding{}

	for _, src := range status.Sources {
		switch src.Status {
		case "up-to-date":
			continue
		case "not-synced":
			findings = append(findings, Finding{
				Section:  SectionSources,
				Severity: SeverityWarn,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source has not been synced into this workspace"),
				Hint:     syncHint(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "update-available":
			findings = append(findings, Finding{
				Section:  SectionSources,
				Severity: SeverityWarn,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source has a newer resolved commit available"),
				Hint:     updateHint(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "missing-source":
			findings = append(findings, Finding{
				Section:  SectionSources,
				Severity: SeverityError,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "canonical source repo is not cloned"),
				Hint:     syncHint(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "invalid-source":
			findings = append(findings, Finding{
				Section:  SectionSources,
				Severity: SeverityError,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source path exists but is not a valid git repository"),
				Hint:     "fix or remove the source path and re-run " + syncCommand(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "invalid-ref":
			findings = append(findings, Finding{
				Section:  SectionSources,
				Severity: SeverityError,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source ref could not be resolved"),
				Hint:     "confirm the ref and re-run " + syncCommand(scope),
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		case "inspect-failed":
			findings = append(findings, Finding{
				Section:  SectionSources,
				Severity: SeverityError,
				Code:     src.Status,
				Subject:  src.Alias,
				Message:  firstNonEmpty(src.Message, "source contents could not be inspected"),
				Hint:     "fix the source repo state and re-run skills doctor",
				Path:     src.RepoPath,
				Ref:      src.Ref,
			})
		}
	}

	for _, link := range status.SkillLinks {
		findings = append(findings, linkFinding(SectionSkills, link, scope)...)
	}
	for _, link := range status.ClaudeLinks {
		findings = append(findings, linkFinding(SectionClaude, link, scope)...)
	}
	for _, link := range status.StaleSkillLinks {
		findings = append(findings, Finding{
			Section:  SectionSkills,
			Severity: SeverityWarn,
			Code:     "stale-managed-link",
			Subject:  link.Source + "/" + link.Skill,
			Message:  "managed skill link is no longer declared",
			Hint:     syncHint(scope),
			Path:     link.Path,
			Target:   link.Target,
		})
	}
	for _, link := range status.StaleClaudeLinks {
		findings = append(findings, Finding{
			Section:  SectionClaude,
			Severity: SeverityWarn,
			Code:     "stale-managed-link",
			Subject:  link.Source + "/" + link.Skill,
			Message:  "managed Claude adapter link is no longer declared",
			Hint:     syncHint(scope),
			Path:     link.Path,
			Target:   link.Target,
		})
	}

	return findings
}

func linkFinding(section string, link project.LinkReport, scope Scope) []Finding {
	subject := link.Source + "/" + link.Skill
	switch link.Status {
	case "linked":
		return nil
	case "missing":
		return []Finding{{
			Section:  section,
			Severity: SeverityWarn,
			Code:     link.Status,
			Subject:  subject,
			Message:  "declared link is missing",
			Hint:     syncHint(scope),
			Path:     link.Path,
			Target:   link.Target,
		}}
	case "stale":
		return []Finding{{
			Section:  section,
			Severity: SeverityWarn,
			Code:     link.Status,
			Subject:  subject,
			Message:  "managed link points at an older target",
			Hint:     syncHint(scope),
			Path:     link.Path,
			Target:   link.Target,
		}}
	case "conflict":
		return []Finding{{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "destination exists as a non-symlink or unmanaged symlink",
			Hint:     "remove or move the conflicting path and re-run " + syncCommand(scope),
			Path:     link.Path,
			Target:   link.Target,
		}}
	case "invalid":
		return []Finding{{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "link could not be inspected",
			Hint:     "inspect the destination path and re-run " + syncCommand(scope),
			Path:     link.Path,
			Target:   link.Target,
		}}
	case "unknown-source":
		return []Finding{{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "skill references a source that is not declared",
			Hint:     "update the manifest so the skill references a declared source",
			Path:     link.Path,
		}}
	case "source-not-ready":
		return []Finding{{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "source is not ready for skill resolution",
			Hint:     "fix source errors and re-run " + syncCommand(scope),
			Path:     link.Path,
		}}
	case "inspect-failed":
		return []Finding{{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  firstNonEmpty(link.Message, "source contents could not be inspected"),
			Hint:     "fix the source repo state and re-run " + syncCommand(scope),
			Path:     link.Path,
		}}
	case "missing-skill":
		return []Finding{{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  "declared skill name was not found in the source",
			Hint:     "run skills skill list --source " + link.Source + " and update the manifest",
			Path:     link.Path,
		}}
	case "ambiguous-skill":
		return []Finding{{
			Section:  section,
			Severity: SeverityError,
			Code:     link.Status,
			Subject:  subject,
			Message:  firstNonEmpty(link.Message, "multiple skills share this directory name"),
			Hint:     "rename one of the duplicate skill directories upstream or choose another source",
			Path:     link.Path,
		}}
	default:
		return nil
	}
}

func syncCommand(scope Scope) string {
	if scope == ScopeGlobal {
		return "skills sync --global"
	}
	return "skills sync"
}

func syncHint(scope Scope) string {
	return "run " + syncCommand(scope)
}

func updateHint(scope Scope) string {
	if scope == ScopeGlobal {
		return "run skills update --global --sync"
	}
	return "run skills update --sync"
}
