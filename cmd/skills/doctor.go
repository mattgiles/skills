package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/doctor"
)

var errDoctorFoundProblems = errors.New("doctor found problems")

func newDoctorCommand() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose config, workspace, and syncability issues",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			scope := doctor.ScopeProject
			if global {
				scope = doctor.ScopeGlobal
			}

			report, err := doctor.Check(context.Background(), cwd, scope)
			if err != nil {
				return err
			}

			renderDoctor(cmd, report, verboseEnabled(cmd))
			if report.HasErrors() {
				return errDoctorFoundProblems
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Inspect global config and the shared home workspace")
	return cmd
}

func renderDoctor(cmd *cobra.Command, report doctor.Report, verbose bool) {
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
			fmt.Fprintln(cmd.OutOrStdout())
		}
		fmt.Fprintln(cmd.OutOrStdout(), section)

		if section == doctor.SectionHints {
			hints := report.Hints()
			if len(hints) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "INFO  ok  no action needed")
				continue
			}
			for _, hint := range hints {
				fmt.Fprintf(cmd.OutOrStdout(), "INFO  hint  %s\n", hint)
			}
			continue
		}

		findings := findingsBySection[section]
		if len(findings) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "INFO  ok  no issues found")
			continue
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		if verbose {
			fmt.Fprintln(w, "SEVERITY\tCODE\tSUBJECT\tMESSAGE\tDETAILS")
		} else {
			fmt.Fprintln(w, "SEVERITY\tCODE\tSUBJECT\tMESSAGE")
		}
		for _, finding := range findings {
			if verbose {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					finding.Severity,
					finding.Code,
					renderDoctorValue(finding.Subject),
					renderDoctorValue(finding.Message),
					renderDoctorDetails(finding),
				)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					finding.Severity,
					finding.Code,
					renderDoctorValue(finding.Subject),
					renderDoctorValue(finding.Message),
				)
			}
		}
		_ = w.Flush()
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\ndoctor: %d errors, %d warnings\n", report.ErrorCount(), report.WarningCount())
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
