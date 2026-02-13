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

// RefExists reports whether a git ref (branch, tag, or SHA) resolves.
func RefExists(repoRoot, ref string) bool {
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "--verify", ref)
	return cmd.Run() == nil
}

// IsStale reports whether a chart has file changes relative to baseRef
// without a corresponding version bump in Chart.yaml. This mirrors the
// common CI pattern:
//
//	git diff --name-only <base>...HEAD -- <chartDir>
//	compare version in Chart.yaml vs git show <base>:Chart.yaml
//
// A chart is stale when files changed but the version field did not.
// A chart that does not exist in baseRef (new chart) is never stale.
func IsStale(repoRoot, chartDir, chartFile, baseRef, currentVersion string) (bool, error) {
	var err error

	// Resolve symlinks so paths are comparable with git's resolved toplevel.
	// On macOS, /var -> /private/var breaks filepath.Rel without this.
	repoRoot, err = filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return false, err
	}
	chartFile, err = filepath.EvalSymlinks(chartFile)
	if err != nil {
		return false, err
	}
	chartDir, err = filepath.EvalSymlinks(chartDir)
	if err != nil {
		return false, err
	}

	relDir, err := filepath.Rel(repoRoot, chartDir)
	if err != nil {
		return false, err
	}
	relFile, err := filepath.Rel(repoRoot, chartFile)
	if err != nil {
		return false, err
	}

	// 1. Any files changed between baseRef and HEAD?
	changed, err := hasChangedFiles(repoRoot, baseRef, relDir)
	if err != nil {
		return false, fmt.Errorf("diff %s...HEAD -- %s: %w", baseRef, relDir, err)
	}
	if !changed {
		return false, nil
	}

	// 2. Files changed; compare versions.
	baseVer, err := showVersion(repoRoot, baseRef, relFile)
	if err != nil {
		// Chart doesn't exist in base ref → new chart, not stale.
		return false, nil
	}

	return currentVersion == baseVer, nil
}

// hasChangedFiles returns true if any files under relDir differ between
// the merge-base of baseRef/HEAD and HEAD (three-dot diff).
func hasChangedFiles(repoRoot, baseRef, relDir string) (bool, error) {
	cmd := exec.Command("git", "-C", repoRoot,
		"diff", "--name-only", baseRef+"...HEAD", "--", relDir,
	)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// showVersion extracts the version field from a Chart.yaml at the given ref.
func showVersion(repoRoot, ref, relFile string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot,
		"show", ref+":"+relFile,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show %s:%s: %w", ref, relFile, err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "version:"))
			v = strings.Trim(v, "\"'")
			return v, nil
		}
	}
	return "", fmt.Errorf("no version field in %s:%s", ref, relFile)
}
