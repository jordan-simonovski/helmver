package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jordan-simonovski/helmver/internal/changelog"
	"github.com/jordan-simonovski/helmver/internal/changeset"
	"github.com/jordan-simonovski/helmver/internal/chart"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply pending changeset files",
	Long:  "Reads all .helmver/*.md changeset files, computes version bumps (highest bump wins per chart), applies them to Chart.yaml, writes changelogs, and removes consumed changeset files.",
	RunE:  runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	files, err := changeset.Discover(cwd)
	if err != nil {
		return fmt.Errorf("reading changesets: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("no pending changesets")
		return nil
	}

	resolved := changeset.Aggregate(files)

	chartPaths, err := chart.Discover(absDir)
	if err != nil {
		return fmt.Errorf("discovering charts: %w", err)
	}

	chartsByName := make(map[string]*chart.Chart)
	for _, p := range chartPaths {
		c, err := chart.Load(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s\n", err)
			continue
		}
		chartsByName[c.Name] = c
	}

	var applied int
	for name, r := range resolved {
		c, ok := chartsByName[name]
		if !ok {
			fmt.Fprintf(os.Stderr, "warning: changeset references chart %q but no Chart.yaml found\n", name)
			continue
		}

		newVer, err := chart.BumpVersion(c.Version, r.Bump)
		if err != nil {
			return fmt.Errorf("bumping %s: %w", name, err)
		}

		oldVer := c.Version
		if err := c.SetVersion(newVer); err != nil {
			return fmt.Errorf("updating %s: %w", c.Path, err)
		}

		message := strings.Join(r.Messages, "\n\n")
		if err := changelog.Prepend(c.Dir, newVer, message); err != nil {
			return fmt.Errorf("updating changelog for %s: %w", name, err)
		}

		fmt.Printf("  %s: %s -> %s (%s)\n", name, oldVer, newVer, r.Bump)
		applied++
	}

	for _, f := range files {
		if err := changeset.Remove(f.Path); err != nil {
			fmt.Fprintf(os.Stderr, "warning: removing %s: %s\n", f.Path, err)
		}
	}

	fmt.Printf("\n%d chart(s) updated, %d changeset(s) consumed\n", applied, len(files))
	return nil
}
