package doctor

import "strings"

func (r Report) ErrorCount() int {
	count := 0
	for _, finding := range r.Findings {
		if finding.Severity == SeverityError {
			count++
		}
	}
	return count
}

func (r Report) WarningCount() int {
	count := 0
	for _, finding := range r.Findings {
		if finding.Severity == SeverityWarn {
			count++
		}
	}
	return count
}

func (r Report) HasErrors() bool {
	return r.ErrorCount() > 0
}

func (r Report) Hints() []string {
	seen := map[string]struct{}{}
	hints := []string{}
	for _, finding := range r.Findings {
		if finding.Hint == "" || finding.Severity == SeverityInfo {
			continue
		}
		if _, ok := seen[finding.Hint]; ok {
			continue
		}
		seen[finding.Hint] = struct{}{}
		hints = append(hints, finding.Hint)
	}
	return hints
}

func (r *Report) addFinding(finding Finding) {
	r.Findings = append(r.Findings, finding)
}

func (r *Report) addFindings(findings ...Finding) {
	r.Findings = append(r.Findings, findings...)
}

func countSectionErrors(findings []Finding, section string) int {
	count := 0
	for _, finding := range findings {
		if finding.Section == section && finding.Severity == SeverityError {
			count++
		}
	}
	return count
}

func hasFinding(findings []Finding, section string, code string) bool {
	for _, finding := range findings {
		if finding.Section == section && finding.Code == code {
			return true
		}
	}
	return false
}

func addSkippedSections(findings []Finding, reason string) []Finding {
	for _, section := range []string{SectionSources, SectionSkills, SectionClaude} {
		if hasFinding(findings, section, "not-checked") {
			continue
		}
		findings = append(findings, Finding{
			Section:  section,
			Severity: SeverityInfo,
			Code:     "not-checked",
			Subject:  section,
			Message:  "not checked because " + reason,
		})
	}
	return findings
}

func downgradeFindingSeverity(findings []Finding, section string, code string, severity Severity) {
	for i := range findings {
		if findings[i].Section == section && findings[i].Code == code {
			findings[i].Severity = severity
			return
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
