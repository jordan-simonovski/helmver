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
	git(t, dir, "branch", "base")

	out, code := helmver(t, dir, "check", "--base", "base")
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
	git(t, dir, "branch", "base")

	// Modify a file without bumping version
	writeFile(t, filepath.Join(dir, "values.yaml"), "key: changed\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "change values")

	out, code := helmver(t, dir, "check", "--base", "base")
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
	// Need at least one commit for the base ref
	writeFile(t, filepath.Join(dir, ".gitkeep"), "")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init")
	git(t, dir, "branch", "base")

	out, code := helmver(t, dir, "check", "--base", "base")
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
	git(t, dir, "branch", "base")

	out, code := helmver(t, dir, "check", "--base", "base", "--dir", filepath.Join(dir, "charts"))
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
	git(t, dir, "branch", "base")

	// Modify only the api chart
	writeFile(t, filepath.Join(dir, "charts", "api", "values.yaml"), "key: changed\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "update api values")

	out, code := helmver(t, dir, "check", "--base", "base", "--dir", filepath.Join(dir, "charts"))
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
	git(t, dir, "branch", "base")

	// Modify both
	writeFile(t, filepath.Join(dir, "charts", "api", "values.yaml"), "key: new\n")
	writeFile(t, filepath.Join(dir, "charts", "web", "values.yaml"), "key: new\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "update both")

	out, code := helmver(t, dir, "check", "--base", "base", "--dir", filepath.Join(dir, "charts"))
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
	if code != 0 {
		t.Errorf("expected exit 0 for non-git directory, got %d", code)
	}
	if !strings.Contains(out, "staleness cannot be determined") {
		t.Errorf("expected staleness warning, got:\n%s", out)
	}
	if !strings.Contains(out, "x") {
		t.Errorf("expected chart listing, got:\n%s", out)
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
	git(t, dir, "branch", "base")

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
	out, code := helmver(t, dir, "check", "--base", "base", "--dir", filepath.Join(dir, "charts"))
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

// --- apply tests ---

func TestE2E_Apply_NoPendingChangesets(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: x\nversion: 0.1.0\n")

	out, code := helmver(t, dir, "apply")
	if code != 0 {
		t.Errorf("expected exit 0, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "no pending changesets") {
		t.Errorf("expected no pending message, got:\n%s", out)
	}
}

func TestE2E_Apply_SingleChangeset(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: myapp\nversion: 1.0.0\n")
	writeFile(t, filepath.Join(dir, ".helmver", "001.md"),
		"---\n\"myapp\": patch\n---\n\nFixed a bug in the auth handler\n")

	out, code := helmver(t, dir, "apply")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "myapp: 1.0.0 -> 1.0.1") {
		t.Errorf("expected version bump in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1 chart(s) updated") {
		t.Errorf("expected updated count, got:\n%s", out)
	}

	// Chart.yaml should be bumped
	raw := readFileE2E(t, filepath.Join(dir, "Chart.yaml"))
	if !strings.Contains(raw, "version: 1.0.1") {
		t.Errorf("Chart.yaml not updated:\n%s", raw)
	}

	// CHANGELOG.md should exist
	cl := readFileE2E(t, filepath.Join(dir, "CHANGELOG.md"))
	if !strings.Contains(cl, "1.0.1") {
		t.Errorf("changelog missing version:\n%s", cl)
	}
	if !strings.Contains(cl, "Fixed a bug in the auth handler") {
		t.Errorf("changelog missing message:\n%s", cl)
	}

	// Changeset file should be deleted
	if _, err := os.Stat(filepath.Join(dir, ".helmver", "001.md")); !os.IsNotExist(err) {
		t.Error("changeset file should have been deleted after apply")
	}
}

func TestE2E_Apply_MultipleChangesets_SameChart_HighestBumpWins(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: api\nversion: 2.0.0\n")
	writeFile(t, filepath.Join(dir, ".helmver", "aaa.md"),
		"---\n\"api\": patch\n---\n\nFixed typo\n")
	writeFile(t, filepath.Join(dir, ".helmver", "bbb.md"),
		"---\n\"api\": minor\n---\n\nAdded new endpoint\n")
	writeFile(t, filepath.Join(dir, ".helmver", "ccc.md"),
		"---\n\"api\": patch\n---\n\nFixed another typo\n")

	out, code := helmver(t, dir, "apply")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. output:\n%s", code, out)
	}
	// minor wins over patch
	if !strings.Contains(out, "api: 2.0.0 -> 2.1.0 (minor)") {
		t.Errorf("expected minor bump, got:\n%s", out)
	}
	if !strings.Contains(out, "3 changeset(s) consumed") {
		t.Errorf("expected 3 consumed, got:\n%s", out)
	}

	cl := readFileE2E(t, filepath.Join(dir, "CHANGELOG.md"))
	if !strings.Contains(cl, "Fixed typo") || !strings.Contains(cl, "Added new endpoint") || !strings.Contains(cl, "Fixed another typo") {
		t.Errorf("changelog should contain all messages:\n%s", cl)
	}
}

func TestE2E_Apply_MultipleCharts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "charts", "api", "Chart.yaml"),
		"apiVersion: v2\nname: api\nversion: 1.0.0\n")
	writeFile(t, filepath.Join(dir, "charts", "web", "Chart.yaml"),
		"apiVersion: v2\nname: web\nversion: 2.0.0\n")
	writeFile(t, filepath.Join(dir, ".helmver", "001.md"),
		"---\n\"api\": minor\n---\n\nNew feature\n")
	writeFile(t, filepath.Join(dir, ".helmver", "002.md"),
		"---\n\"web\": major\n---\n\nBreaking change\n")

	out, code := helmver(t, dir, "apply", "--dir", dir)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "2 chart(s) updated") {
		t.Errorf("expected 2 charts updated, got:\n%s", out)
	}

	api := readFileE2E(t, filepath.Join(dir, "charts", "api", "Chart.yaml"))
	if !strings.Contains(api, "version: 1.1.0") {
		t.Errorf("api not bumped:\n%s", api)
	}
	web := readFileE2E(t, filepath.Join(dir, "charts", "web", "Chart.yaml"))
	if !strings.Contains(web, "version: 3.0.0") {
		t.Errorf("web not bumped:\n%s", web)
	}
}

func TestE2E_Apply_UnknownChart_Warning(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: real\nversion: 1.0.0\n")
	writeFile(t, filepath.Join(dir, ".helmver", "001.md"),
		"---\n\"ghost\": patch\n---\n\nThis chart does not exist\n")

	out, code := helmver(t, dir, "apply")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "ghost") {
		t.Errorf("expected warning about ghost chart, got:\n%s", out)
	}
}

// --- check --require-changeset tests ---

func TestE2E_Check_RequireChangeset_CoveredByChangeset(t *testing.T) {
	dir := initGitRepo(t)

	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: myapp\nversion: 1.0.0\n")
	writeFile(t, filepath.Join(dir, "values.yaml"), "key: val\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init")
	git(t, dir, "branch", "base")

	// Make it stale
	writeFile(t, filepath.Join(dir, "values.yaml"), "key: changed\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "change values")

	// Without --require-changeset: stale
	_, code := helmver(t, dir, "check", "--base", "base")
	if code != 1 {
		t.Fatal("should be stale without changeset flag")
	}

	// Add a changeset file
	writeFile(t, filepath.Join(dir, ".helmver", "001.md"),
		"---\n\"myapp\": patch\n---\n\nWill bump later\n")

	// With --require-changeset: covered
	out, code := helmver(t, dir, "check", "--base", "base", "--require-changeset")
	if code != 0 {
		t.Errorf("expected exit 0 with changeset, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "has changeset") {
		t.Errorf("expected 'has changeset' note, got:\n%s", out)
	}
	if !strings.Contains(out, "up to date") {
		t.Errorf("expected up to date, got:\n%s", out)
	}
}

func TestE2E_Check_RequireChangeset_NotCovered(t *testing.T) {
	dir := initGitRepo(t)

	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: myapp\nversion: 1.0.0\n")
	writeFile(t, filepath.Join(dir, "values.yaml"), "key: val\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init")
	git(t, dir, "branch", "base")

	writeFile(t, filepath.Join(dir, "values.yaml"), "key: changed\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "change values")

	// No changeset file -- still fails
	out, code := helmver(t, dir, "check", "--base", "base", "--require-changeset")
	if code != 1 {
		t.Errorf("expected exit 1 without changeset, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "myapp") {
		t.Errorf("expected myapp in stale list, got:\n%s", out)
	}
}

func TestE2E_Check_RequireChangeset_PartialCoverage(t *testing.T) {
	dir := initGitRepo(t)

	for _, name := range []string{"api", "web"} {
		chartDir := filepath.Join(dir, "charts", name)
		writeFile(t, filepath.Join(chartDir, "Chart.yaml"),
			"apiVersion: v2\nname: "+name+"\nversion: 0.1.0\n")
		writeFile(t, filepath.Join(chartDir, "values.yaml"), "key: val\n")
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init")
	git(t, dir, "branch", "base")

	// Make both stale
	writeFile(t, filepath.Join(dir, "charts", "api", "values.yaml"), "key: new\n")
	writeFile(t, filepath.Join(dir, "charts", "web", "values.yaml"), "key: new\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "update both")

	// Changeset only for api
	writeFile(t, filepath.Join(dir, ".helmver", "001.md"),
		"---\n\"api\": minor\n---\n\nNew feature\n")

	out, code := helmver(t, dir, "check", "--base", "base", "--require-changeset",
		"--dir", filepath.Join(dir, "charts"))
	if code != 1 {
		t.Errorf("expected exit 1 (web uncovered), got %d. output:\n%s", code, out)
	}
	// api should be covered
	if !strings.Contains(out, "has changeset") {
		t.Errorf("expected api 'has changeset', got:\n%s", out)
	}
	// web should still be flagged
	if !strings.Contains(out, "1 chart(s) need a version bump") {
		t.Errorf("expected 1 stale chart (web), got:\n%s", out)
	}
}

func TestE2E_Check_RequireChangeset_AllClean(t *testing.T) {
	dir := initGitRepo(t)

	writeFile(t, filepath.Join(dir, "Chart.yaml"),
		"apiVersion: v2\nname: x\nversion: 1.0.0\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "init")
	git(t, dir, "branch", "base")

	out, code := helmver(t, dir, "check", "--base", "base", "--require-changeset")
	if code != 0 {
		t.Errorf("expected exit 0, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Errorf("expected up to date, got:\n%s", out)
	}
}

// --- helpers ---

func readFileE2E(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}
