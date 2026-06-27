package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jordan-simonovski/helmver/internal/changeset"
	"github.com/jordan-simonovski/helmver/internal/chart"
	"github.com/jordan-simonovski/helmver/internal/git"
)

// ChartResult holds the check outcome for one chart.
type ChartResult struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Dir          string `json:"dir"`
	HasChangeset bool   `json:"hasChangeset,omitempty"`
}

// Result is the outcome of a helmver check run.
type Result struct {
	StaleCharts   []ChartResult
	CoveredCharts []ChartResult
	Changesets    []*changeset.File
	AllUpToDate   bool
}

// Options configures a check run.
type Options struct {
	Dir              string
	Base             string
	Exclude          []string
	RequireChangeset bool
	ChangesetRoot    string
}

// Run discovers charts, detects staleness, and optionally filters by changesets.
func Run(opts Options) (*Result, error) {
	absDir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, err
	}

	charts, err := chart.Discover(absDir, opts.Exclude)
	if err != nil {
		return nil, fmt.Errorf("discovering charts: %w", err)
	}

	result := &Result{AllUpToDate: true}
	if len(charts) == 0 {
		return result, nil
	}

	if !git.IsRepo(absDir) {
		return result, nil
	}

	repoRoot, err := git.RepoRoot(absDir)
	if err != nil {
		return nil, err
	}

	baseRef := opts.Base
	if baseRef == "" {
		baseRef = git.ResolveBase(repoRoot)
	}

	var stale []*chart.Chart
	for _, path := range charts {
		c, err := chart.Load(path)
		if err != nil {
			continue
		}

		isStale, err := git.IsStale(repoRoot, c.Dir, c.Path, baseRef, c.Version)
		if err != nil {
			continue
		}
		if isStale {
			stale = append(stale, c)
		}
	}

	changesetRoot := opts.ChangesetRoot
	if changesetRoot == "" {
		changesetRoot, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	var covered map[string]bool
	if opts.RequireChangeset && len(stale) > 0 {
		files, err := changeset.Discover(changesetRoot)
		if err != nil {
			return nil, fmt.Errorf("reading changesets: %w", err)
		}
		result.Changesets = files
		covered = changeset.ChartNames(files)
	}

	for _, c := range stale {
		cr := ChartResult{
			Name:    c.Name,
			Version: c.Version,
			Dir:     c.Dir,
		}
		if covered != nil && covered[c.Name] {
			cr.HasChangeset = true
			result.CoveredCharts = append(result.CoveredCharts, cr)
		} else {
			result.StaleCharts = append(result.StaleCharts, cr)
		}
	}

	result.AllUpToDate = len(result.StaleCharts) == 0
	return result, nil
}
