package chart_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jsimonovski/helmver/internal/chart"
	"github.com/jsimonovski/helmver/internal/git"
)

// Integration tests that validate the discover -> load -> staleness pipeline.

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
}

func mkFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_MonorepoStaleness(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")

	// Create 3 charts
	for _, name := range []string{"api", "web", "worker"} {
		cd := filepath.Join(dir, "charts", name)
		mkFile(t, filepath.Join(cd, "Chart.yaml"),
			"apiVersion: v2\nname: "+name+"\nversion: 0.1.0\n")
		mkFile(t, filepath.Join(cd, "values.yaml"), "key: val\n")
	}
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", "init")

	// Modify only api
	mkFile(t, filepath.Join(dir, "charts", "api", "values.yaml"), "key: changed\n")
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", "update api")

	// Discover and annotate
	paths, err := chart.Discover(filepath.Join(dir, "charts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 charts, got %d", len(paths))
	}

	staleCount := 0
	for _, p := range paths {
		c, err := chart.Load(p)
		if err != nil {
			t.Fatal(err)
		}
		isStale, err := git.IsStale(c.Dir, c.Path)
		if err != nil {
			t.Fatal(err)
		}
		c.Stale = isStale
		if c.Stale {
			staleCount++
			if c.Name != "api" {
				t.Errorf("only api should be stale, got %s", c.Name)
			}
		}
	}
	if staleCount != 1 {
		t.Errorf("expected 1 stale chart, got %d", staleCount)
	}
}

func TestIntegration_NonGitDiscovery(t *testing.T) {
	// Changeset should work without git: discover charts, all marked not stale.
	dir := t.TempDir()

	for _, name := range []string{"svc-a", "svc-b"} {
		cd := filepath.Join(dir, "charts", name)
		mkFile(t, filepath.Join(cd, "Chart.yaml"),
			"apiVersion: v2\nname: "+name+"\nversion: 1.0.0\n")
	}

	paths, err := chart.Discover(filepath.Join(dir, "charts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 charts, got %d", len(paths))
	}

	hasGit := git.IsRepo(dir)
	if hasGit {
		t.Fatal("temp dir should not be a git repo")
	}

	for _, p := range paths {
		c, err := chart.Load(p)
		if err != nil {
			t.Fatal(err)
		}
		// Without git, Stale defaults to false
		if c.Stale {
			t.Errorf("chart %s should not be stale without git", c.Name)
		}
	}
}
