package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRepo returns true if dir is inside a git repository.
func IsRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// RepoRoot returns the root of the git repository containing dir.
func RepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %s", dir)
	}
	return strings.TrimSpace(string(out)), nil
}

// IsStale returns true if the chart directory has changes since the last
// commit that bumped the version field in Chart.yaml.
//
// Logic:
//  1. Find the last commit that changed the "version:" line in Chart.yaml.
//  2. Check if any files in the chart directory have changed since that commit.
//  3. If there are changes (or no version commit exists), the chart is stale.
func IsStale(chartDir string, chartFile string) (bool, error) {
	repoRoot, err := RepoRoot(chartDir)
	if err != nil {
		return false, err
	}

	// Resolve symlinks so paths are comparable with git's resolved toplevel.
	// On macOS, /var -> /private/var breaks filepath.Rel without this.
	chartFile, err = filepath.EvalSymlinks(chartFile)
	if err != nil {
		return false, err
	}
	chartDir, err = filepath.EvalSymlinks(chartDir)
	if err != nil {
		return false, err
	}

	// Get relative paths from repo root for git commands
	relChart, err := filepath.Rel(repoRoot, chartFile)
	if err != nil {
		return false, err
	}
	relDir, err := filepath.Rel(repoRoot, chartDir)
	if err != nil {
		return false, err
	}

	// Find the last commit that touched the version line in Chart.yaml
	// -S finds commits where the number of occurrences of the string changed
	// We look for changes to lines matching "version:" in the Chart.yaml
	lastVersionCommit, err := lastCommitChangingVersion(repoRoot, relChart)
	if err != nil || lastVersionCommit == "" {
		// No commit ever touched version: treat as stale
		return true, nil
	}

	// Check for any changes in the chart directory since that commit
	// (both committed and uncommitted)
	changes, err := changedFilesSince(repoRoot, lastVersionCommit, relDir)
	if err != nil {
		return false, err
	}

	return len(changes) > 0, nil
}

// lastCommitChangingVersion finds the most recent commit that modified the
// "version:" field in the given Chart.yaml file.
func lastCommitChangingVersion(repoRoot, relChartFile string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot,
		"log", "-1", "--format=%H",
		"-G", `^version:`,
		"--", relChartFile,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// changedFilesSince returns files in relDir that have changed since the given commit.
// This includes both committed changes and uncommitted working tree changes.
func changedFilesSince(repoRoot, commit, relDir string) ([]string, error) {
	// Committed changes since the version bump
	cmd := exec.Command("git", "-C", repoRoot,
		"diff", "--name-only", commit, "HEAD", "--", relDir,
	)
	out, err := cmd.Output()
	if err != nil {
		// If HEAD doesn't exist (fresh repo), fall back to checking staged/unstaged
		out = nil
	}

	files := splitLines(string(out))

	// Also check for uncommitted changes (staged + unstaged)
	cmd2 := exec.Command("git", "-C", repoRoot,
		"diff", "--name-only", commit, "--", relDir,
	)
	out2, err := cmd2.Output()
	if err == nil {
		files = append(files, splitLines(string(out2))...)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, f := range files {
		if f != "" && !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}
	return unique, nil
}

func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
