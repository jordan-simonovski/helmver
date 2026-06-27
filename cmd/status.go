package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jordan-simonovski/helmver/internal/check"
)

var (
	statusFormat string
	statusCommit string
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report chart version and changeset status",
	Long:  "Reports stale charts and pending .helmver/ changesets. Intended for CI and GitHub Actions.",
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&statusFormat, "format", "markdown", "output format: markdown or json")
	statusCmd.Flags().StringVar(&statusCommit, "commit", "", "commit SHA to include in markdown output")
}

func runStatus(cmd *cobra.Command, args []string) error {
	result, err := check.Run(check.Options{
		Dir:              dir,
		Base:             base,
		Exclude:          exclude,
		RequireChangeset: requireChangeset,
	})
	if err != nil {
		return err
	}

	out, err := check.Format(result, statusFormat, statusCommit)
	if err != nil {
		return err
	}

	fmt.Print(out)
	if !result.AllUpToDate {
		os.Exit(1)
	}
	return nil
}
