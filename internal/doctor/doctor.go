package doctor

import (
	"context"
	"fmt"

	"github.com/mattgiles/skills/internal/config"
	"github.com/mattgiles/skills/internal/source"
)

const (
	SectionEnvironment = "ENVIRONMENT"
	SectionConfig      = "CONFIG"
	SectionWorkspace   = "WORKSPACE"
	SectionGit         = "GIT"
	SectionSources     = "SOURCES"
	SectionSkills      = "SKILLS"
	SectionClaude      = "CLAUDE"
	SectionHints       = "HINTS"
)

type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

type Severity string

const (
	SeverityError Severity = "ERROR"
	SeverityWarn  Severity = "WARN"
	SeverityInfo  Severity = "INFO"
)

type Finding struct {
	Section  string
	Severity Severity
	Code     string
	Subject  string
	Message  string
	Hint     string
	Path     string
	Target   string
	Ref      string
}

type Report struct {
	Scope    Scope
	Target   string
	Findings []Finding
}

func Check(ctx context.Context, cwd string, scope Scope) (Report, error) {
	report := Report{
		Scope:  scope,
		Target: cwd,
	}

	gitAvailable := true
	if err := source.EnsureGitAvailable(); err != nil {
		gitAvailable = false
		report.addFinding(Finding{
			Section:  SectionEnvironment,
			Severity: SeverityError,
			Code:     "git-missing",
			Subject:  "git",
			Message:  err.Error(),
			Hint:     "install git and re-run skills doctor",
		})
	}

	var result scopeInspection
	switch scope {
	case ScopeProject:
		result = inspectProjectWorkspace(ctx, cwd, gitAvailable)
	case ScopeGlobal:
		result = inspectGlobalWorkspace(ctx, cwd, gitAvailable)
	default:
		return Report{}, fmt.Errorf("unsupported doctor scope %q", scope)
	}

	if result.target != "" {
		report.Target = result.target
	}
	report.addFindings(result.findings...)
	return report, nil
}

func defaultConfigPath() (string, error) {
	return config.DefaultConfigPath()
}
