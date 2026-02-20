package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jordan-simonovski/helmver/internal/changeset"
	"github.com/jordan-simonovski/helmver/internal/chart"
	"github.com/jordan-simonovski/helmver/internal/git"
)

var requireChangeset bool

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if any chart versions are stale",
	Long:  "Scans for Chart.yaml files and reports which charts have file changes relative to --base without a corresponding version bump. Exits 1 if any charts are stale (CI-friendly). Use --require-changeset to accept pending .helmver/ changeset files as a valid intent to bump.",
	RunE:  runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&requireChangeset, "require-changeset", false, "accept pending .helmver/ changeset files; stale charts with a changeset are not flagged")
}

func runCheck(cmd *cobra.Command, args []string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	charts, err := chart.Discover(absDir, exclude)
	if err != nil {
		return fmt.Errorf("discovering charts: %w", err)
	}

	if len(charts) == 0 {
		fmt.Println("no Chart.yaml files found")
		return nil
	}

	if !git.IsRepo(absDir) {
		fmt.Fprintln(os.Stderr, "warning: not a git repository; staleness cannot be determined")
		for _, path := range charts {
			c, err := chart.Load(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: %s\n", err)
				continue
			}
			fmt.Printf("  %-30s %s  (%s)\n", c.Name, c.Version, c.Dir)
		}
		return nil
	}

	repoRoot, err := git.RepoRoot(absDir)
	if err != nil {
		return err
	}

	baseRef := base
	if baseRef == "" {
		baseRef = git.ResolveBase(repoRoot)
	}

	if !git.RefExists(repoRoot, baseRef) {
		fetchHint := baseRef
		if strings.HasPrefix(baseRef, "origin/") {
			fetchHint = "git fetch origin " + strings.TrimPrefix(baseRef, "origin/") + " --depth=1"
		}
		return fmt.Errorf("base ref %q not found; fetch it first (e.g. %s) or set --base", baseRef, fetchHint)
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

	if requireChangeset && len(stale) > 0 {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return cwdErr
		}
		stale, err = filterByChangesets(cwd, stale)
		if err != nil {
			return err
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

// filterByChangesets removes stale charts that have a pending changeset file.
func filterByChangesets(root string, stale []*chart.Chart) ([]*chart.Chart, error) {
	files, err := changeset.Discover(root)
	if err != nil {
		return nil, fmt.Errorf("reading changesets: %w", err)
	}

	covered := changeset.ChartNames(files)
	var remaining []*chart.Chart
	for _, c := range stale {
		if covered[c.Name] {
			fmt.Printf("  %-30s %s  (has changeset)\n", c.Name, c.Version)
		} else {
			remaining = append(remaining, c)
		}
	}
	return remaining, nil
}
