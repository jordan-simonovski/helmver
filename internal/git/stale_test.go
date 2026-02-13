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

func TestRefExists(t *testing.T) {
	dir := initGitRepo(t)
	writeFile(t, filepath.Join(dir, ".gitkeep"), "")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "initial")

	if !RefExists(dir, "HEAD") {
		t.Error("HEAD should exist")
	}
	if RefExists(dir, "nonexistent-branch") {
		t.Error("nonexistent-branch should not exist")
	}
}

func TestIsStaleNoChanges(t *testing.T) {
	dir := initGitRepo(t)

	chartDir := filepath.Join(dir, "mychart")
	chartFile := filepath.Join(chartDir, "Chart.yaml")
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "initial")
	run(t, dir, "git", "branch", "base")

	// No changes since base → not stale
	stale, err := IsStale(dir, chartDir, chartFile, "base", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("chart with no changes since base should not be stale")
	}
}

func TestIsStaleFileChangedNoVersionBump(t *testing.T) {
	dir := initGitRepo(t)

	chartDir := filepath.Join(dir, "mychart")
	chartFile := filepath.Join(chartDir, "Chart.yaml")
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "initial")
	run(t, dir, "git", "branch", "base")

	// Modify values without bumping version
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: newval\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "update values")

	stale, err := IsStale(dir, chartDir, chartFile, "base", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !stale {
		t.Error("chart with file changes and no version bump should be stale")
	}
}

func TestIsStaleVersionBumped(t *testing.T) {
	dir := initGitRepo(t)

	chartDir := filepath.Join(dir, "mychart")
	chartFile := filepath.Join(chartDir, "Chart.yaml")
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "initial")
	run(t, dir, "git", "branch", "base")

	// Modify values AND bump version
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: newval\n")
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.2.0\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "bump version")

	stale, err := IsStale(dir, chartDir, chartFile, "base", "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("chart with version bump should not be stale")
	}
}

func TestIsStaleNewChart(t *testing.T) {
	dir := initGitRepo(t)

	// Empty initial commit as base
	writeFile(t, filepath.Join(dir, ".gitkeep"), "")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "initial")
	run(t, dir, "git", "branch", "base")

	// Add a chart that does not exist in base
	chartDir := filepath.Join(dir, "mychart")
	chartFile := filepath.Join(chartDir, "Chart.yaml")
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.1.0\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "add chart")

	stale, err := IsStale(dir, chartDir, chartFile, "base", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("new chart not present in base should not be stale")
	}
}

func TestIsStaleMultipleChangesOneBump(t *testing.T) {
	dir := initGitRepo(t)

	chartDir := filepath.Join(dir, "mychart")
	chartFile := filepath.Join(chartDir, "Chart.yaml")
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	writeFile(t, filepath.Join(chartDir, "templates", "deploy.yaml"), "kind: Deployment\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "initial")
	run(t, dir, "git", "branch", "base")

	// Change values
	writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: v2\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "update values")

	// Change template
	writeFile(t, filepath.Join(chartDir, "templates", "deploy.yaml"), "kind: Deployment\nmetadata:\n  name: x\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "update template")

	// Bump version
	writeFile(t, chartFile, "apiVersion: v2\nname: mychart\nversion: 0.2.0\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "bump")

	stale, err := IsStale(dir, chartDir, chartFile, "base", "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("chart should not be stale after version bump covers all changes")
	}
}
