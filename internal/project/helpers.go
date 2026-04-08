package project

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

func fatalSourceReports(reports []SourceReport) error {
	problems := make([]string, 0)
	for _, report := range reports {
		switch report.Status {
		case "missing-source", "invalid-source", "invalid-ref", "inspect-failed":
			problem := fmt.Sprintf("%s: %s", report.Alias, report.Status)
			if report.Message != "" {
				problem += " (" + report.Message + ")"
			}
			problems = append(problems, problem)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.New(strings.Join(problems, "; "))
}

func sourceStateMap(state State) map[string]SourceState {
	out := map[string]SourceState{}
	for _, src := range state.Sources {
		out[src.Source] = src
	}
	return out
}

func managedLinkMap(links []ManagedLink) map[string]ManagedLink {
	out := map[string]ManagedLink{}
	for _, link := range links {
		out[link.Path] = link
	}
	return out
}

func desiredLinkMap(links []desiredLink) map[string]desiredLink {
	out := map[string]desiredLink{}
	for _, link := range links {
		out[link.Path] = link
	}
	return out
}

func mergeSourceStates(state State, updates []SourceState, removeAliases map[string]struct{}) State {
	current := sourceStateMap(state)
	for _, update := range updates {
		current[update.Source] = update
	}
	for alias := range removeAliases {
		delete(current, alias)
	}

	next := State{
		Sources:     make([]SourceState, 0, len(current)),
		SkillLinks:  state.SkillLinks,
		ClaudeLinks: state.ClaudeLinks,
	}
	for _, src := range current {
		next.Sources = append(next.Sources, src)
	}
	sort.Slice(next.Sources, func(i, j int) bool {
		return next.Sources[i].Source < next.Sources[j].Source
	})
	return next
}

func sanitizeIDComponent(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func shortCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

func syncSourceStatus(status string) string {
	switch status {
	case "not-synced":
		return "resolved"
	case "update-available":
		return "updated"
	default:
		return status
	}
}

func dryRunLinkStatus(status string) string {
	switch status {
	case "created":
		return "would-create"
	case "updated":
		return "would-update"
	default:
		return status
	}
}

func managedLinkPaths(links []ManagedLink) []string {
	paths := make([]string, 0, len(links))
	for _, link := range links {
		paths = append(paths, link.Path)
	}
	sort.Strings(paths)
	return paths
}

func toManagedLinks(links []desiredLink) []ManagedLink {
	out := make([]ManagedLink, 0, len(links))
	for _, link := range links {
		out = append(out, link.ManagedLink)
	}
	return out
}

func sortSourceReports(reports []SourceReport) {
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Alias < reports[j].Alias
	})
}

func sortLinkReports(reports []LinkReport) {
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Source != reports[j].Source {
			return reports[i].Source < reports[j].Source
		}
		return reports[i].Skill < reports[j].Skill
	})
}

func sortManagedLinks(links []ManagedLink) {
	sort.Slice(links, func(i, j int) bool {
		return links[i].Path < links[j].Path
	})
}

func sortedResolvedSources(sources map[string]*resolvedSource) []*resolvedSource {
	aliases := make([]string, 0, len(sources))
	for alias := range sources {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	out := make([]*resolvedSource, 0, len(aliases))
	for _, alias := range aliases {
		out = append(out, sources[alias])
	}
	return out
}
