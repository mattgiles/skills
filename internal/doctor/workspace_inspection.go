package doctor

import (
	"context"
	"errors"
	"os"

	"github.com/mattgiles/skills/internal/project"
)

type workspaceInputInspection struct {
	manifest   project.Manifest
	usable     bool
	skipReason string
	findings   []Finding
}

type scopeInspection struct {
	target   string
	findings []Finding
}

func inspectProjectWorkspace(ctx context.Context, cwd string, gitAvailable bool) scopeInspection {
	findings := []Finding{}
	manifestPath := project.ManifestPath(cwd)
	statePath := project.StatePath(cwd)
	localConfigPath := project.LocalConfigPath(cwd)

	cacheInspection := inspectProjectCacheConfig(cwd, localConfigPath)
	findings = append(findings, cacheInspection.findings...)

	ownershipFindings := inspectProjectOwnershipFindings(ctx, cwd)
	findings = append(findings, ownershipFindings...)

	if !cacheInspection.usable {
		findings = addSkippedSections(findings, cacheInspection.skipReason)
		return scopeInspection{target: cwd, findings: findings}
	}

	inputInspection := inspectWorkspaceInputs(
		manifestPath,
		statePath,
		"project",
		"skills init",
	)
	findings = append(findings, inputInspection.findings...)
	if !inputInspection.usable {
		findings = addSkippedSections(findings, inputInspection.skipReason)
		return scopeInspection{target: cwd, findings: findings}
	}

	findings = append(findings, declaredWorkspaceFindings(inputInspection.manifest, manifestPath, "project")...)
	if !gitAvailable {
		findings = addSkippedSections(findings, "git is not available")
		return scopeInspection{target: cwd, findings: findings}
	}

	status, err := project.Status(ctx, cwd)
	if err != nil {
		findings = append(findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "workspace-invalid",
			Subject:  manifestPath,
			Message:  err.Error(),
			Path:     manifestPath,
		})
		findings = addSkippedSections(findings, "workspace inputs could not be resolved")
		return scopeInspection{target: cwd, findings: findings}
	}

	findings = append(findings, statusFindings(status, ScopeProject)...)
	return scopeInspection{target: cwd, findings: findings}
}

func inspectGlobalWorkspace(ctx context.Context, cwd string, gitAvailable bool) scopeInspection {
	findings := []Finding{}
	configPath, err := defaultConfigPath()
	if err != nil {
		findings = append(findings, Finding{
			Section:  SectionConfig,
			Severity: SeverityError,
			Code:     "config-path-failed",
			Subject:  "config",
			Message:  err.Error(),
		})
		return scopeInspection{target: cwd, findings: findings}
	}

	configInspection := inspectGlobalConfig(configPath)
	findings = append(findings, configInspection.findings...)
	if !configInspection.usable {
		findings = append(findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityInfo,
			Code:     "not-checked",
			Subject:  "home workspace",
			Message:  "home workspace was not checked because global config could not be loaded",
		})
		findings = addSkippedSections(findings, "global config could not be loaded")
		return scopeInspection{target: cwd, findings: findings}
	}

	manifestPath, err := project.HomeManifestPath(configInspection.cfg)
	if err != nil {
		findings = append(findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "home-manifest-path-failed",
			Subject:  "home workspace",
			Message:  err.Error(),
		})
		findings = addSkippedSections(findings, "home workspace paths could not be resolved")
		return scopeInspection{target: cwd, findings: findings}
	}

	statePath, err := project.HomeStatePath(configInspection.cfg)
	if err != nil {
		findings = append(findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "home-state-path-failed",
			Subject:  "home workspace",
			Message:  err.Error(),
		})
		findings = addSkippedSections(findings, "home workspace paths could not be resolved")
		return scopeInspection{target: cwd, findings: findings}
	}

	target := configInspection.paths.sharedSkills
	inputInspection := inspectWorkspaceInputs(
		manifestPath,
		statePath,
		"home",
		"skills init --global",
	)
	findings = append(findings, inputInspection.findings...)
	if !inputInspection.usable {
		downgradeFindingSeverity(findings, SectionWorkspace, "manifest-missing", SeverityWarn)
		findings = addSkippedSections(findings, inputInspection.skipReason)
		return scopeInspection{target: target, findings: findings}
	}

	findings = append(findings, declaredWorkspaceFindings(inputInspection.manifest, manifestPath, "home")...)
	if !gitAvailable {
		findings = addSkippedSections(findings, "git is not available")
		return scopeInspection{target: target, findings: findings}
	}

	status, err := project.HomeStatus(ctx, configInspection.cfg)
	if err != nil {
		findings = append(findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "workspace-invalid",
			Subject:  manifestPath,
			Message:  err.Error(),
			Path:     manifestPath,
		})
		findings = addSkippedSections(findings, "workspace inputs could not be resolved")
		return scopeInspection{target: target, findings: findings}
	}

	findings = append(findings, statusFindings(status, ScopeGlobal)...)
	return scopeInspection{target: target, findings: findings}
}

func inspectWorkspaceInputs(
	manifestPath string,
	statePath string,
	workspaceLabel string,
	initHint string,
) workspaceInputInspection {
	result := workspaceInputInspection{
		findings: []Finding{},
	}

	if _, err := os.Stat(manifestPath); errors.Is(err, os.ErrNotExist) {
		result.findings = append(result.findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "manifest-missing",
			Subject:  manifestPath,
			Message:  workspaceLabel + " manifest not found",
			Hint:     "run " + initHint,
			Path:     manifestPath,
		})
		result.skipReason = workspaceLabel + " manifest is missing"
		return result
	} else if err != nil {
		result.findings = append(result.findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "manifest-stat-failed",
			Subject:  manifestPath,
			Message:  err.Error(),
			Path:     manifestPath,
		})
		result.skipReason = workspaceLabel + " manifest could not be read"
		return result
	}

	manifest, err := project.LoadManifestAt(manifestPath)
	if err != nil {
		result.findings = append(result.findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "manifest-parse-failed",
			Subject:  manifestPath,
			Message:  err.Error(),
			Path:     manifestPath,
		})
		result.skipReason = workspaceLabel + " manifest could not be parsed"
		return result
	}
	result.manifest = manifest

	if _, err := os.Stat(statePath); errors.Is(err, os.ErrNotExist) {
		result.findings = append(result.findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityInfo,
			Code:     "state-missing",
			Subject:  statePath,
			Message:  "state file not found; sync has not been run yet",
			Path:     statePath,
		})
	} else if err != nil {
		result.findings = append(result.findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "state-stat-failed",
			Subject:  statePath,
			Message:  err.Error(),
			Path:     statePath,
		})
		result.skipReason = workspaceLabel + " state could not be read"
		return result
	} else if _, err := project.LoadStateAt(statePath); err != nil {
		result.findings = append(result.findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityError,
			Code:     "state-parse-failed",
			Subject:  statePath,
			Message:  err.Error(),
			Path:     statePath,
		})
		result.skipReason = workspaceLabel + " state could not be parsed"
		return result
	}

	result.usable = true
	return result
}

func declaredWorkspaceFindings(manifest project.Manifest, manifestPath string, workspaceLabel string) []Finding {
	findings := []Finding{}

	if len(manifest.Sources) == 0 {
		findings = append(findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityWarn,
			Code:     "no-sources-declared",
			Subject:  manifestPath,
			Message:  "no " + workspaceLabel + " sources declared",
			Hint:     "declare at least one source in " + manifestLocationLabel(workspaceLabel),
			Path:     manifestPath,
		})
	}
	if len(manifest.Skills) == 0 {
		findings = append(findings, Finding{
			Section:  SectionWorkspace,
			Severity: SeverityWarn,
			Code:     "no-skills-declared",
			Subject:  manifestPath,
			Message:  "no " + workspaceLabel + " skills declared",
			Hint:     "declare at least one skill in " + manifestLocationLabel(workspaceLabel),
			Path:     manifestPath,
		})
	}

	return findings
}

func manifestLocationLabel(workspaceLabel string) string {
	if workspaceLabel == "project" {
		return ".agents/manifest.yaml"
	}
	return "the home manifest"
}
