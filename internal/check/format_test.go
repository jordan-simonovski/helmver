package check_test

import (
	"strings"
	"testing"

	"github.com/jordan-simonovski/helmver/internal/check"
)

func TestFormatMarkdown_allUpToDate(t *testing.T) {
	result := &check.Result{AllUpToDate: true}
	out, err := check.Format(result, "markdown", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(out, "All charts up to date", "abc123") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestFormatMarkdown_staleCharts(t *testing.T) {
	result := &check.Result{
		AllUpToDate: false,
		StaleCharts: []check.ChartResult{
			{Name: "api", Version: "1.2.3", Dir: "/charts/api"},
		},
	}
	out, err := check.Format(result, "markdown", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(out, "Charts need a version bump", "api", "1.2.3") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestFormatMarkdown_withChangeset(t *testing.T) {
	result := &check.Result{
		AllUpToDate: true,
		CoveredCharts: []check.ChartResult{
			{Name: "api", Version: "1.2.3", Dir: "/charts/api", HasChangeset: true},
		},
	}
	out, err := check.Format(result, "markdown", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(out, "Changeset detected") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestFormatJSON(t *testing.T) {
	result := &check.Result{
		AllUpToDate: false,
		StaleCharts: []check.ChartResult{
			{Name: "api", Version: "1.0.0", Dir: "/charts/api"},
		},
	}
	out, err := check.Format(result, "json", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(out, `"allUpToDate": false`, `"name": "api"`, `"commitSha": "abc123"`) {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
