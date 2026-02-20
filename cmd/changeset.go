package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jordan-simonovski/helmver/internal/changelog"
	"github.com/jordan-simonovski/helmver/internal/changeset"
	"github.com/jordan-simonovski/helmver/internal/chart"
	"github.com/jordan-simonovski/helmver/internal/git"
	"github.com/jordan-simonovski/helmver/internal/tui"
)

var writeChangesetFlag bool

var changesetCmd = &cobra.Command{
	Use:   "changeset",
	Short: "Interactively create version bumps and changelogs",
	Long:  "Scans for Chart.yaml files and launches an interactive TUI to select charts, choose bump types, and write changelog messages. By default applies immediately; use --write to create .helmver/ changeset files for later application with 'helmver apply'.",
	RunE:  runChangeset,
}

func init() {
	changesetCmd.Flags().BoolVar(&writeChangesetFlag, "write", false, "write .helmver/ changeset files instead of applying immediately")
}

func runChangeset(cmd *cobra.Command, args []string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	chartPaths, err := chart.Discover(absDir, exclude)
	if err != nil {
		return fmt.Errorf("discovering charts: %w", err)
	}

	if len(chartPaths) == 0 {
		fmt.Println("no Chart.yaml files found")
		return nil
	}

	hasGit := git.IsRepo(absDir)
	var repoRoot, baseRef string
	if hasGit {
		repoRoot, err = git.RepoRoot(absDir)
		if err != nil {
			hasGit = false
		} else {
			baseRef = base
			if baseRef == "" {
				baseRef = git.ResolveBase(repoRoot)
			}
		}
	}

	var all []*chart.Chart
	for _, path := range chartPaths {
		c, err := chart.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s\n", err)
			continue
		}

		if hasGit {
			isStale, err := git.IsStale(repoRoot, c.Dir, c.Path, baseRef, c.Version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: checking %s: %s\n", c.Name, err)
			} else {
				c.Stale = isStale
			}
		}

		all = append(all, c)
	}

	if len(all) == 0 {
		fmt.Println("no valid charts found")
		return nil
	}

	changesets, err := tui.Run(all)
	if err != nil {
		return err
	}

	if changesets == nil {
		fmt.Println("aborted")
		return nil
	}

	if writeChangesetFlag {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return cwdErr
		}
		return writeChangesetFiles(cwd, changesets)
	}
	return applyChangesets(changesets)
}

func writeChangesetFiles(root string, changesets []tui.Changeset) error {
	for _, cs := range changesets {
		entries := []changeset.Entry{{Chart: cs.Chart.Name, Bump: cs.Bump}}
		path, err := changeset.Write(root, entries, cs.Message)
		if err != nil {
			return fmt.Errorf("writing changeset: %w", err)
		}
		fmt.Printf("  %s: %s changeset -> %s\n", cs.Chart.Name, cs.Bump, filepath.Base(path))
	}
	fmt.Printf("\n%d changeset(s) written to .helmver/\n", len(changesets))
	return nil
}

func applyChangesets(changesets []tui.Changeset) error {
	for _, cs := range changesets {
		oldVer := cs.Chart.Version
		if err := cs.Chart.SetVersion(cs.NewVer); err != nil {
			return fmt.Errorf("updating %s: %w", cs.Chart.Path, err)
		}

		if err := changelog.Prepend(cs.Chart.Dir, cs.NewVer, cs.Message); err != nil {
			return fmt.Errorf("updating changelog for %s: %w", cs.Chart.Name, err)
		}

		fmt.Printf("  %s: %s -> %s\n", cs.Chart.Name, oldVer, cs.NewVer)
	}

	fmt.Printf("\n%d chart(s) updated\n", len(changesets))
	return nil
}
