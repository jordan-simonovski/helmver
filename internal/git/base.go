package git

import (
	"os"
	"os/exec"
	"strings"
)

// ciBaseEnvVars maps CI provider environment variables to the branch name
// of the PR/MR target. Checked in order; first non-empty wins.
var ciBaseEnvVars = []string{
	"GITHUB_BASE_REF",                     // GitHub Actions (pull_request)
	"BITBUCKET_PR_DESTINATION_BRANCH",     // Bitbucket Pipelines
	"CI_MERGE_REQUEST_TARGET_BRANCH_NAME", // GitLab CI (merge request)
	"CI_DEFAULT_BRANCH",                   // GitLab CI (fallback: repo default branch)
}

// ResolveBase determines the base git ref for staleness comparison.
// Priority:
//  1. CI environment variable (prefixed with origin/)
//  2. Remote HEAD (git symbolic-ref refs/remotes/origin/HEAD)
//  3. Fallback to origin/main
func ResolveBase(repoRoot string) string {
	for _, env := range ciBaseEnvVars {
		if val := os.Getenv(env); val != "" {
			return "origin/" + val
		}
	}
	return DefaultBranch(repoRoot)
}

// DefaultBranch returns the origin's default branch as "origin/<branch>".
// Uses git symbolic-ref to read the remote HEAD; falls back to "origin/main".
func DefaultBranch(repoRoot string) string {
	cmd := exec.Command("git", "-C", repoRoot, "symbolic-ref", "refs/remotes/origin/HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "origin/main"
	}
	// Output: "refs/remotes/origin/<branch>\n"
	ref := strings.TrimSpace(string(out))
	const prefix = "refs/remotes/origin/"
	if strings.HasPrefix(ref, prefix) {
		return "origin/" + strings.TrimPrefix(ref, prefix)
	}
	return "origin/main"
}
