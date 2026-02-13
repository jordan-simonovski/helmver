package git

import "os"

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
//  2. Fallback to origin/main
func ResolveBase() string {
	for _, env := range ciBaseEnvVars {
		if val := os.Getenv(env); val != "" {
			return "origin/" + val
		}
	}
	return "origin/main"
}
