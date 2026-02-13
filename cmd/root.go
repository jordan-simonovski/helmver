package cmd

import (
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	dir     string
)

var rootCmd = &cobra.Command{
	Use:   "helmver",
	Short: "Helm chart versioning and changelog management",
	Long:  "helmver detects stale Helm chart versions and provides an interactive TUI for bumping versions and writing changelogs.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dir, "dir", ".", "root directory to scan for Chart.yaml files")
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(changesetCmd)
	rootCmd.Version = version
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
