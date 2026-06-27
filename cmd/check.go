package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jordan-simonovski/helmver/internal/check"
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
	result, err := check.Run(check.Options{
		Dir:              dir,
		Base:             base,
		Exclude:          exclude,
		RequireChangeset: requireChangeset,
	})
	if err != nil {
		return err
	}

	if result.AllUpToDate {
		fmt.Println("all charts up to date")
		return nil
	}

	fmt.Printf("%d chart(s) need a version bump:\n\n", len(result.StaleCharts))
	for _, c := range result.StaleCharts {
		fmt.Printf("  %-30s %s  (%s)\n", c.Name, c.Version, c.Dir)
	}
	for _, c := range result.CoveredCharts {
		fmt.Printf("  %-30s %s  (has changeset)\n", c.Name, c.Version)
	}
	fmt.Println()

	os.Exit(1)
	return nil
}
