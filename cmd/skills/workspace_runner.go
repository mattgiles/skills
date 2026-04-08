package main

import (
	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/doctor"
	"github.com/mattgiles/skills/internal/project"
	"github.com/mattgiles/skills/internal/ui"
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
	view := ui.New(cmd)
	var (
		result project.SyncResult
		err    error
	)

	err = view.RunTask("Syncing workspace", ui.TaskOptions{
		UseErrorWriter: true,
		SuccessText:    "Synced workspace",
		FailureText:    "Failed to sync workspace",
	}, func() error {
		if target.Scope == scopeGlobal {
			result, err = project.HomeSync(cmd.Context(), target.Config, options)
		} else {
			result, err = project.Sync(cmd.Context(), target.ProjectRoot, options)
		}
		return err
	})
	if err != nil {
		return err
	}

	renderWorkspaceSummary(cmd, target.Summary, verboseEnabled(cmd))
	renderWorkspaceSync(cmd, result, verboseEnabled(cmd))
	return nil
}

func runUpdateCommand(cmd *cobra.Command, target workspaceTarget, options project.UpdateOptions) error {
	view := ui.New(cmd)
	var (
		result project.UpdateResult
		err    error
	)

	err = view.RunTask("Updating workspace", ui.TaskOptions{
		UseErrorWriter: true,
		SuccessText:    "Updated workspace",
		FailureText:    "Failed to update workspace",
	}, func() error {
		if target.Scope == scopeGlobal {
			result, err = project.HomeUpdate(cmd.Context(), target.Config, options)
		} else {
			result, err = project.Update(cmd.Context(), target.ProjectRoot, options)
		}
		return err
	})
	if err != nil {
		return err
	}

	renderWorkspaceSummary(cmd, target.Summary, verboseEnabled(cmd))
	renderProjectUpdate(cmd, result, verboseEnabled(cmd))
	return nil
}

func runDoctorCommand(cmd *cobra.Command, target workspaceTarget) error {
	scope := doctorScope(target.Scope)
	view := ui.New(cmd)
	var (
		report doctor.Report
		err    error
	)
	err = view.RunTask("Running doctor checks", ui.TaskOptions{
		UseErrorWriter: true,
		SuccessText:    "Doctor checks completed",
		FailureText:    "Doctor checks failed",
	}, func() error {
		report, err = doctorCheck(cmd.Context(), doctorTargetDir(target), scope)
		return err
	})
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
	view := ui.New(cmd)
	if target.Scope == scopeGlobal {
		var result project.SyncResult
		err := view.RunTask("Syncing added skill", ui.TaskOptions{
			UseErrorWriter: true,
			SuccessText:    "Synced added skill",
			FailureText:    "Failed to sync added skill",
		}, func() error {
			var syncErr error
			result, syncErr = project.HomeSync(cmd.Context(), target.Config, project.SyncOptions{})
			return syncErr
		})
		if err != nil {
			return addSyncOutcome{}, err
		}
		return addSyncOutcome{summary: target.Summary, result: result}, nil
	}

	var result project.SyncResult
	err := view.RunTask("Syncing added skill", ui.TaskOptions{
		UseErrorWriter: true,
		SuccessText:    "Synced added skill",
		FailureText:    "Failed to sync added skill",
	}, func() error {
		var syncErr error
		result, syncErr = project.Sync(cmd.Context(), target.ProjectRoot, project.SyncOptions{})
		return syncErr
	})
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
