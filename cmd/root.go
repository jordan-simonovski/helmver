package cmd

import (
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	dir     string
	base    string
	exclude []string
)

var rootCmd = &cobra.Command{
	Use:   "helmver",
	Short: "Helm chart versioning and changelog management",
	Long:  "helmver detects stale Helm chart versions and provides an interactive TUI for bumping versions and writing changelogs.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dir, "dir", ".", "root directory to scan for Chart.yaml files")
	rootCmd.PersistentFlags().StringVar(&base, "base", "", "base git ref to compare against; auto-detected from CI env (GITHUB_BASE_REF, BITBUCKET_PR_DESTINATION_BRANCH, CI_MERGE_REQUEST_TARGET_BRANCH_NAME, CI_DEFAULT_BRANCH), then remote HEAD, falls back to origin/main")
	rootCmd.PersistentFlags().StringSliceVar(&exclude, "exclude", nil, "glob patterns to exclude from chart discovery (repeatable, matched against path relative to --dir)")
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(changesetCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.Version = version
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
