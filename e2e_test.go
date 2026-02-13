package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binary string

func TestMain(m *testing.M) {
	// Build the binary once for all e2e tests
	tmp, err := os.MkdirTemp("", "helmver-e2e-bin-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	binary = filepath.Join(tmp, "helmver")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build helmver binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// --- helpers ---

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init")
	git(t, dir, "config", "user.email", "test@test.com")
	git(t, dir, "config", "user.name", "Test")
	return dir
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed: %v", args, err)
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

func helmver(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("helmver %v failed: %v", args, err)
		}
	}
	return string(out), exitCode
}

// --- single chart tests ---

func TestE2E_Check_SingleChart_Clean(t *testing.T) {
	dir := initGitRepo(t)

	// Create and commit a chart
	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: myapp\nversion: 1.0.0\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init chart")

	out, code := helmver(t, dir, "check")
	if code != 0 {
		t.Errorf("expected exit 0, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Errorf("expected 'up to date' message, got:\n%s", out)
	}
}

func TestE2E_Check_SingleChart_Stale(t *testing.T) {
	dir := initGitRepo(t)

	// Create and commit a chart
	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: myapp\nversion: 1.0.0\n")
	writeFile(t, filepath.Join(dir, "values.yaml"), "key: val\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init chart")

	// Modify a file without bumping version
	writeFile(t, filepath.Join(dir, "values.yaml"), "key: changed\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "change values")

	out, code := helmver(t, dir, "check")
	if code != 1 {
		t.Errorf("expected exit 1, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "myapp") {
		t.Errorf("expected chart name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "need a version bump") {
		t.Errorf("expected stale message, got:\n%s", out)
	}
}

func TestE2E_Check_SingleChart_NoCharts(t *testing.T) {
	dir := initGitRepo(t)

	out, code := helmver(t, dir, "check")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out, "no Chart.yaml") {
		t.Errorf("expected no charts message, got:\n%s", out)
	}
}

// --- monorepo tests (--dir flag) ---

func TestE2E_Check_Monorepo_AllClean(t *testing.T) {
	dir := initGitRepo(t)

	// Create multiple charts
	for _, name := range []string{"api", "web", "worker"} {
		chartDir := filepath.Join(dir, "charts", name)
		writeFile(t, filepath.Join(chartDir, "Chart.yaml"),
			"apiVersion: v2\nname: "+name+"\nversion: 0.1.0\n")
		writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init all charts")

	out, code := helmver(t, dir, "check", "--dir", filepath.Join(dir, "charts"))
	if code != 0 {
		t.Errorf("expected exit 0, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Errorf("expected all up to date, got:\n%s", out)
	}
}

func TestE2E_Check_Monorepo_SomeStale(t *testing.T) {
	dir := initGitRepo(t)

	// Create multiple charts
	for _, name := range []string{"api", "web", "worker"} {
		chartDir := filepath.Join(dir, "charts", name)
		writeFile(t, filepath.Join(chartDir, "Chart.yaml"),
			"apiVersion: v2\nname: "+name+"\nversion: 0.1.0\n")
		writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init all charts")

	// Modify only the api chart
	writeFile(t, filepath.Join(dir, "charts", "api", "values.yaml"), "key: changed\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "update api values")

	out, code := helmver(t, dir, "check", "--dir", filepath.Join(dir, "charts"))
	if code != 1 {
		t.Errorf("expected exit 1, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "api") {
		t.Errorf("expected 'api' in stale list, got:\n%s", out)
	}
	// web and worker should NOT be in the stale list
	if strings.Contains(out, "web") {
		t.Errorf("'web' should not be stale, got:\n%s", out)
	}
	if strings.Contains(out, "worker") {
		t.Errorf("'worker' should not be stale, got:\n%s", out)
	}
}

func TestE2E_Check_Monorepo_AllStale(t *testing.T) {
	dir := initGitRepo(t)

	for _, name := range []string{"api", "web"} {
		chartDir := filepath.Join(dir, "charts", name)
		writeFile(t, filepath.Join(chartDir, "Chart.yaml"),
			"apiVersion: v2\nname: "+name+"\nversion: 0.1.0\n")
		writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init")

	// Modify both
	writeFile(t, filepath.Join(dir, "charts", "api", "values.yaml"), "key: new\n")
	writeFile(t, filepath.Join(dir, "charts", "web", "values.yaml"), "key: new\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "update both")

	out, code := helmver(t, dir, "check", "--dir", filepath.Join(dir, "charts"))
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(out, "2 chart(s)") {
		t.Errorf("expected 2 stale charts, got:\n%s", out)
	}
}

func TestE2E_Check_NotGitRepo(t *testing.T) {
	dir := t.TempDir() // Not a git repo

	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: x\nversion: 0.1.0\n")

	out, code := helmver(t, dir, "check")
	if code == 0 {
		t.Error("expected non-zero exit for non-git directory")
	}
	if !strings.Contains(out, "not inside a git repository") {
		t.Errorf("expected git error message, got:\n%s", out)
	}
}

func TestE2E_Check_Monorepo_AfterBump(t *testing.T) {
	dir := initGitRepo(t)

	// Two charts
	for _, name := range []string{"api", "web"} {
		chartDir := filepath.Join(dir, "charts", name)
		writeFile(t, filepath.Join(chartDir, "Chart.yaml"),
			"apiVersion: v2\nname: "+name+"\nversion: 0.1.0\n")
		writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init")

	// Modify api
	writeFile(t, filepath.Join(dir, "charts", "api", "values.yaml"), "key: changed\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "update api")

	// Bump api version (simulate helmver changeset)
	writeFile(t, filepath.Join(dir, "charts", "api", "Chart.yaml"),
		"apiVersion: v2\nname: api\nversion: 0.2.0\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "bump api to 0.2.0")

	// Now everything should be clean
	out, code := helmver(t, dir, "check", "--dir", filepath.Join(dir, "charts"))
	if code != 0 {
		t.Errorf("expected exit 0 after bump, got %d. output:\n%s", code, out)
	}
}

func TestE2E_Changeset_NotGitRepo_NoError(t *testing.T) {
	// changeset should NOT fail with "not a git repo" anymore.
	// It will fail because there's no TTY, but the error should NOT be about git.
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: nogit-chart\nversion: 0.1.0\n")

	out, code := helmver(t, dir, "changeset")
	// May fail due to no TTY, but must NOT mention "not inside a git repository"
	if strings.Contains(out, "not inside a git repository") {
		t.Errorf("changeset should not require git, got:\n%s", out)
	}
	_ = code // exit code may be non-zero due to TTY, that's fine
}

func TestE2E_Changeset_NoCharts(t *testing.T) {
	dir := initGitRepo(t)

	out, code := helmver(t, dir, "changeset")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out, "no Chart.yaml") {
		t.Errorf("expected no charts message, got:\n%s", out)
	}
}

func TestE2E_Version(t *testing.T) {
	dir := initGitRepo(t)
	out, code := helmver(t, dir, "--version")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out, "helmver version") {
		t.Errorf("expected version output, got:\n%s", out)
	}
}
