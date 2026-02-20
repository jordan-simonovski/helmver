package git

import "testing"

// clearCIEnv ensures no CI env vars leak into a test from the host shell.
func clearCIEnv(t *testing.T) {
	t.Helper()
	for _, env := range ciBaseEnvVars {
		t.Setenv(env, "")
	}
}

func TestResolveBase_Default(t *testing.T) {
	clearCIEnv(t)
	// Without a real remote, DefaultBranch falls back to origin/main.
	got := ResolveBase("/nonexistent")
	if got != "origin/main" {
		t.Errorf("expected origin/main, got %q", got)
	}
}

func TestResolveBase_GitHub(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("GITHUB_BASE_REF", "develop")
	got := ResolveBase("")
	if got != "origin/develop" {
		t.Errorf("expected origin/develop, got %q", got)
	}
}

func TestResolveBase_Bitbucket(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("BITBUCKET_PR_DESTINATION_BRANCH", "master")
	got := ResolveBase("")
	if got != "origin/master" {
		t.Errorf("expected origin/master, got %q", got)
	}
}

func TestResolveBase_GitLab(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME", "main")
	got := ResolveBase("")
	if got != "origin/main" {
		t.Errorf("expected origin/main, got %q", got)
	}
}

func TestResolveBase_GitLabDefault(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("CI_DEFAULT_BRANCH", "trunk")
	got := ResolveBase("")
	if got != "origin/trunk" {
		t.Errorf("expected origin/trunk, got %q", got)
	}
}

func TestResolveBase_GitHubTakesPrecedence(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("GITHUB_BASE_REF", "gh-base")
	t.Setenv("CI_DEFAULT_BRANCH", "should-not-win")
	got := ResolveBase("")
	if got != "origin/gh-base" {
		t.Errorf("expected origin/gh-base, got %q", got)
	}
}
