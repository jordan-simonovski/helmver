package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a fresh git repo in a temp dir and returns its path.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")

	return dir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v failed: %v", name, args, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestIsRepo(t *testing.T) {
	dir := initGitRepo(t)
	if !IsRepo(dir) {
		t.Error("expected IsRepo=true for git repo")
	}

	tmp := t.TempDir()
	if IsRepo(tmp) {
		t.Error("expected IsRepo=false for non-git dir")
	}
}

func TestIsStaleNewChart(t *testing.T) {
	dir := initGitRepo(t)

	// Create a chart that has never been committed
	chartDir := filepath.Join(dir, "mychart")
	chartFile := filepath.Join(chartDir, "Chart.yaml")
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.1.0\n")

	// Unstaged new file should be stale
	stale, err := IsStale(chartDir, chartFile)
	if err != nil {
		t.Fatal(err)
	}
	if !stale {
		t.Error("new uncommitted chart should be stale")
	}
}

func TestIsStaleAfterVersionBump(t *testing.T) {
	dir := initGitRepo(t)

	chartDir := filepath.Join(dir, "mychart")
	chartFile := filepath.Join(chartDir, "Chart.yaml")

	// Initial commit with chart
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "initial")

	// At this point, the version was just committed and nothing changed since.
	// The chart should NOT be stale.
	stale, err := IsStale(chartDir, chartFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("chart should not be stale right after version commit")
	}
}

func TestIsStaleAfterFileChange(t *testing.T) {
	dir := initGitRepo(t)

	chartDir := filepath.Join(dir, "mychart")
	chartFile := filepath.Join(chartDir, "Chart.yaml")

	// Initial commit with chart
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "initial")

	// Now modify a file in the chart directory
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: newval\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "update values")

	// Now there are changes since the last version bump => stale
	stale, err := IsStale(chartDir, chartFile)
	if err != nil {
		t.Fatal(err)
	}
	if !stale {
		t.Error("chart should be stale after file changes without version bump")
	}
}

func TestIsStaleAfterVersionBumpFollowedByClean(t *testing.T) {
	dir := initGitRepo(t)

	chartDir := filepath.Join(dir, "mychart")
	chartFile := filepath.Join(chartDir, "Chart.yaml")

	// Initial commit
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "initial")

	// Modify values
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: newval\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "update values")

	// Now bump the version (simulating what helmver changeset does)
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.2.0\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "bump to 0.2.0")

	// After version bump, chart should be clean
	stale, err := IsStale(chartDir, chartFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("chart should not be stale after version bump")
	}
}
