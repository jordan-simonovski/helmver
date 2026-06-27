package check

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Format renders a check result as json or markdown.
func Format(result *Result, format, commitSHA string) (string, error) {
	switch format {
	case "json":
		return formatJSON(result, commitSHA)
	case "markdown":
		return formatMarkdown(result, commitSHA), nil
	default:
		return "", fmt.Errorf("unknown format %q (use json or markdown)", format)
	}
}

func formatJSON(result *Result, commitSHA string) (string, error) {
	payload := struct {
		CommitSHA     string        `json:"commitSha"`
		AllUpToDate   bool          `json:"allUpToDate"`
		StaleCharts   []ChartResult `json:"staleCharts"`
		CoveredCharts []ChartResult `json:"coveredCharts"`
		Changesets    int           `json:"changesetCount"`
	}{
		CommitSHA:     commitSHA,
		AllUpToDate:   result.AllUpToDate,
		StaleCharts:   result.StaleCharts,
		CoveredCharts: result.CoveredCharts,
		Changesets:    len(result.Changesets),
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func formatMarkdown(result *Result, commitSHA string) string {
	var b strings.Builder

	switch {
	case result.AllUpToDate && len(result.CoveredCharts) > 0:
		b.WriteString("### ✅ Changeset detected\n\n")
		writeCommitLine(&b, commitSHA)
		b.WriteString("**The changes in this PR include helmver changesets for charts that need version bumps.**\n\n")
		writeChangesetTable(&b, result)
	case result.AllUpToDate:
		b.WriteString("### ✅ All charts up to date\n\n")
		writeCommitLine(&b, commitSHA)
		b.WriteString("No chart version bumps are needed for the changes in this PR.\n")
	default:
		b.WriteString("### ⚠️ Charts need a version bump\n\n")
		writeCommitLine(&b, commitSHA)
		b.WriteString("The following charts have file changes without a version bump or pending changeset:\n\n")
		writeStaleTable(&b, result.StaleCharts)
		b.WriteString("\nRun `helmver changeset --write` locally and commit the `.helmver/` files, or bump `version` in `Chart.yaml` directly.\n")
		if len(result.CoveredCharts) > 0 {
			b.WriteString("\nThese stale charts already have a pending changeset:\n\n")
			writeCoveredTable(&b, result.CoveredCharts)
		}
	}

	b.WriteString("\n\n[Learn about helmver changesets](https://github.com/jordan-simonovski/helmver#changeset-files)")
	return b.String()
}

func writeCommitLine(b *strings.Builder, commitSHA string) {
	if commitSHA != "" {
		fmt.Fprintf(b, "Latest commit: `%s`\n\n", commitSHA)
	}
}

func writeStaleTable(b *strings.Builder, charts []ChartResult) {
	b.WriteString("| Chart | Version | Directory |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, c := range charts {
		fmt.Fprintf(b, "| %s | %s | `%s` |\n", c.Name, c.Version, c.Dir)
	}
}

func writeCoveredTable(b *strings.Builder, charts []ChartResult) {
	b.WriteString("| Chart | Version | Directory |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, c := range charts {
		fmt.Fprintf(b, "| %s | %s | `%s` |\n", c.Name, c.Version, c.Dir)
	}
}

func writeChangesetTable(b *strings.Builder, result *Result) {
	if len(result.Changesets) == 0 {
		return
	}

	b.WriteString("<details>\n<summary>Pending changesets</summary>\n\n")
	b.WriteString("| Chart | Bump |\n")
	b.WriteString("| --- | --- |\n")
	var order []string
	seen := make(map[string]string)
	for _, f := range result.Changesets {
		for _, e := range f.Entries {
			if _, ok := seen[e.Chart]; !ok {
				order = append(order, e.Chart)
				seen[e.Chart] = e.Bump
			} else if bumpRank(e.Bump) > bumpRank(seen[e.Chart]) {
				seen[e.Chart] = e.Bump
			}
		}
	}
	for _, chart := range order {
		fmt.Fprintf(b, "| %s | %s |\n", chart, seen[chart])
	}
	b.WriteString("\n</details>\n")
}

func bumpRank(bump string) int {
	switch bump {
	case "major":
		return 3
	case "minor":
		return 2
	case "patch":
		return 1
	default:
		return 0
	}
}
