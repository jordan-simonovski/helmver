package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jordan-simonovski/helmver/internal/chart"
	"github.com/jordan-simonovski/helmver/internal/git"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if any chart versions are stale",
	Long:  "Scans for Chart.yaml files and reports which charts have file changes relative to --base without a corresponding version bump. Exits 1 if any charts are stale (CI-friendly).",
	RunE:  runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	if !git.IsRepo(absDir) {
		return fmt.Errorf("%s is not inside a git repository", absDir)
	}

	repoRoot, err := git.RepoRoot(absDir)
	if err != nil {
		return err
	}

	baseRef := base
	if baseRef == "" {
		baseRef = git.ResolveBase()
	}

	if !git.RefExists(repoRoot, baseRef) {
		// Derive a useful fetch hint: "origin/develop" → "git fetch origin develop --depth=1"
		fetchHint := baseRef
		if strings.HasPrefix(baseRef, "origin/") {
			fetchHint = "git fetch origin " + strings.TrimPrefix(baseRef, "origin/") + " --depth=1"
		}
		return fmt.Errorf("base ref %q not found; fetch it first (e.g. %s) or set --base", baseRef, fetchHint)
	}

	charts, err := chart.Discover(absDir)
	if err != nil {
		return fmt.Errorf("discovering charts: %w", err)
	}

	if len(charts) == 0 {
		fmt.Println("no Chart.yaml files found")
		return nil
	}

	var stale []*chart.Chart
	for _, path := range charts {
		c, err := chart.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s\n", err)
			continue
		}

		isStale, err := git.IsStale(repoRoot, c.Dir, c.Path, baseRef, c.Version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: checking %s: %s\n", c.Name, err)
			continue
		}

		if isStale {
			stale = append(stale, c)
		}
	}

	if len(stale) == 0 {
		fmt.Println("all charts up to date")
		return nil
	}

	fmt.Printf("%d chart(s) need a version bump:\n\n", len(stale))
	for _, c := range stale {
		fmt.Printf("  %-30s %s  (%s)\n", c.Name, c.Version, c.Dir)
	}
	fmt.Println()

	os.Exit(1)
	return nil
}
