package main

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/doctor"
	"github.com/mattgiles/skills/internal/ui"
)

var errDoctorFoundProblems = errors.New("doctor found problems")

func newDoctorCommand() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose config, workspace, and syncability issues",
		RunE: func(cmd *cobra.Command, _ []string) error {
			target, err := resolveDoctorTarget(cmd.Context(), global)
			if err != nil {
				return err
			}
			return runDoctorCommand(cmd, target)
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Inspect global config and the shared home workspace")
	return cmd
}

func resolveDoctorTarget(ctx context.Context, global bool) (workspaceTarget, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return workspaceTarget{}, err
	}

	if global {
		return workspaceTarget{
			Scope:     scopeGlobal,
			TargetDir: cwd,
		}, nil
	}

	projectRoot, err := resolveRepoRoot(ctx, cwd, false)
	if err != nil {
		return workspaceTarget{}, errors.New("outside a Git repo; use skills doctor --global")
	}

	return workspaceTarget{
		Scope:       scopeRepo,
		TargetDir:   projectRoot,
		ProjectRoot: projectRoot,
	}, nil
}

func renderDoctorSummary(cmd *cobra.Command, ctx context.Context, target workspaceTarget) {
	var (
		summary workspaceSummary
		err     error
	)

	if target.Scope == scopeGlobal {
		cfg, loadErr := loadConfig()
		if loadErr != nil {
			return
		}
		summary, err = globalWorkspaceSummary(ctx, cfg)
	} else {
		summary, err = repoWorkspaceSummary(ctx, target.ProjectRoot)
	}
	if err != nil {
		return
	}

	renderWorkspaceSummary(cmd, summary, verboseEnabled(cmd))
}

func renderDoctor(cmd *cobra.Command, report doctor.Report, verbose bool) {
	view := ui.New(cmd)
	sections := []string{
		doctor.SectionEnvironment,
		doctor.SectionConfig,
		doctor.SectionWorkspace,
		doctor.SectionGit,
		doctor.SectionSources,
		doctor.SectionSkills,
		doctor.SectionClaude,
		doctor.SectionHints,
	}

	findingsBySection := map[string][]doctor.Finding{}
	for _, finding := range report.Findings {
		findingsBySection[finding.Section] = append(findingsBySection[finding.Section], finding)
	}

	for i, section := range sections {
		if i > 0 {
			view.Blank()
		}

		if section == doctor.SectionHints {
			hints := report.Hints()
			if len(hints) == 0 {
				view.Infof("%s: no action needed", section)
				continue
			}
			for _, hint := range hints {
				view.Infof("%s: %s", section, hint)
			}
			continue
		}

		findings := findingsBySection[section]
		if len(findings) == 0 {
			view.Infof("%s: no issues found", section)
			continue
		}

		rows := make([][]string, 0, len(findings))
		for _, finding := range findings {
			if verbose {
				rows = append(rows, []string{
					string(finding.Severity),
					finding.Code,
					renderDoctorValue(finding.Subject),
					renderDoctorValue(finding.Message),
					renderDoctorDetails(finding),
				})
			} else {
				rows = append(rows, []string{
					string(finding.Severity),
					finding.Code,
					renderDoctorValue(finding.Subject),
					renderDoctorValue(finding.Message),
				})
			}
		}

		columns := []string{"Severity", "Code", "Subject", "Message"}
		if verbose {
			columns = append(columns, "Details")
		}
		_ = view.RenderTable(ui.Table{
			Title:   section,
			Columns: columns,
			Rows:    rows,
		})
	}

	view.Blank()
	view.Infof("doctor: %d errors, %d warnings", report.ErrorCount(), report.WarningCount())
}

func renderDoctorValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func renderDoctorDetails(finding doctor.Finding) string {
	details := make([]string, 0, 3)
	if finding.Path != "" {
		details = append(details, "path="+finding.Path)
	}
	if finding.Target != "" {
		details = append(details, "target="+finding.Target)
	}
	if finding.Ref != "" {
		details = append(details, "ref="+finding.Ref)
	}
	if len(details) == 0 {
		return "-"
	}
	return strings.Join(details, " ")
}
