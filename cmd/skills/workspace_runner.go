package main

import (
	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/doctor"
	"github.com/mattgiles/skills/internal/project"
)

func runStatusCommand(cmd *cobra.Command, target workspaceTarget) error {
	var (
		report project.StatusReport
		err    error
	)

	if target.Scope == scopeGlobal {
		report, err = project.HomeStatus(cmd.Context(), target.Config)
		if err != nil {
			return err
		}
		renderWorkspaceSummary(cmd, target.Summary, verboseEnabled(cmd))
		renderWorkspaceStatus(cmd, report, verboseEnabled(cmd), "no home sources declared", "no home skills declared")
		return nil
	}

	report, err = project.Status(cmd.Context(), target.ProjectRoot)
	if err != nil {
		return err
	}
	renderWorkspaceSummary(cmd, target.Summary, verboseEnabled(cmd))
	renderWorkspaceStatus(cmd, report, verboseEnabled(cmd), "no repo sources declared", "no repo skills declared")
	return nil
}

func runSyncCommand(cmd *cobra.Command, target workspaceTarget, options project.SyncOptions) error {
	var (
		result project.SyncResult
		err    error
	)

	if target.Scope == scopeGlobal {
		result, err = project.HomeSync(cmd.Context(), target.Config, options)
	} else {
		result, err = project.Sync(cmd.Context(), target.ProjectRoot, options)
	}
	if err != nil {
		return err
	}

	renderWorkspaceSummary(cmd, target.Summary, verboseEnabled(cmd))
	renderWorkspaceSync(cmd, result, verboseEnabled(cmd))
	return nil
}

func runUpdateCommand(cmd *cobra.Command, target workspaceTarget, options project.UpdateOptions) error {
	var (
		result project.UpdateResult
		err    error
	)

	if target.Scope == scopeGlobal {
		result, err = project.HomeUpdate(cmd.Context(), target.Config, options)
	} else {
		result, err = project.Update(cmd.Context(), target.ProjectRoot, options)
	}
	if err != nil {
		return err
	}

	renderWorkspaceSummary(cmd, target.Summary, verboseEnabled(cmd))
	renderProjectUpdate(cmd, result, verboseEnabled(cmd))
	return nil
}

func runDoctorCommand(cmd *cobra.Command, target workspaceTarget) error {
	scope := doctorScope(target.Scope)
	report, err := doctorCheck(cmd.Context(), doctorTargetDir(target), scope)
	if err != nil {
		return err
	}

	renderDoctorSummary(cmd, cmd.Context(), target)
	renderDoctor(cmd, report, verboseEnabled(cmd))
	if report.HasErrors() {
		return errDoctorFoundProblems
	}
	return nil
}

func runAddSync(cmd *cobra.Command, target sourceManifestTarget) (addSyncOutcome, error) {
	if target.Scope == scopeGlobal {
		result, err := project.HomeSync(cmd.Context(), target.Config, project.SyncOptions{})
		if err != nil {
			return addSyncOutcome{}, err
		}
		return addSyncOutcome{summary: target.Summary, result: result}, nil
	}

	result, err := project.Sync(cmd.Context(), target.ProjectRoot, project.SyncOptions{})
	if err != nil {
		return addSyncOutcome{}, err
	}
	return addSyncOutcome{summary: target.Summary, result: result}, nil
}

func doctorTargetDir(target workspaceTarget) string {
	return target.TargetDir
}

func doctorScope(scope commandScope) doctor.Scope {
	if scope == scopeGlobal {
		return doctor.ScopeGlobal
	}
	return doctor.ScopeProject
}

var doctorCheck = doctor.Check
