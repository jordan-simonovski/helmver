package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jsimonovski/helmver/internal/changelog"
	"github.com/jsimonovski/helmver/internal/chart"
	"github.com/jsimonovski/helmver/internal/git"
)

var binary string

func TestMain(m *testing.M) {
	// Build the binary once for all acceptance tests.
	// We build from the repo root (two levels up from test/acceptance/).
	tmp, err := os.MkdirTemp("", "helmver-acceptance-bin-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	binary = filepath.Join(tmp, "helmver")
	repoRoot := projectRoot()
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = repoRoot
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build helmver binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// --- helpers ---

func projectRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func testdataDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "testdata")
}

// copyDir recursively copies src to dst.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatalf("copying %s -> %s: %v", src, dst, err)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Acceptance Test")
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed in %s: %v", args, dir, err)
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
			t.Fatalf("helmver %v failed: %v\noutput: %s", args, err, out)
		}
	}
	return string(out), exitCode
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
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

// setupFixture copies a testdata fixture into a temp git repo, commits it, and returns the repo path.
func setupFixture(t *testing.T, fixtureName string) string {
	t.Helper()
	repo := t.TempDir()
	gitInit(t, repo)
	copyDir(t, filepath.Join(testdataDir(), fixtureName), repo)
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "initial commit")
	return repo
}

// ===================================================================
// Single chart acceptance tests
// ===================================================================

func TestAcceptance_SingleChart_CheckClean(t *testing.T) {
	repo := setupFixture(t, "single-chart")

	out, code := helmver(t, repo, "check")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Errorf("expected up to date, got:\n%s", out)
	}
}

func TestAcceptance_SingleChart_CheckStaleAfterEdit(t *testing.T) {
	repo := setupFixture(t, "single-chart")

	// Modify values.yaml without bumping version
	writeFile(t, filepath.Join(repo, "values.yaml"), "replicaCount: 5\nimage:\n  repository: nginx\n")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "scale up replicas")

	out, code := helmver(t, repo, "check")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "myapp") {
		t.Errorf("expected 'myapp' in output, got:\n%s", out)
	}
}

func TestAcceptance_SingleChart_CheckStaleAfterTemplateEdit(t *testing.T) {
	repo := setupFixture(t, "single-chart")

	// Modify a template file
	writeFile(t, filepath.Join(repo, "templates", "deployment.yaml"),
		"# modified template\napiVersion: apps/v1\nkind: Deployment\n")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "update template")

	out, code := helmver(t, repo, "check")
	if code != 1 {
		t.Fatalf("expected exit 1 after template edit, got %d. output:\n%s", code, out)
	}
}

func TestAcceptance_SingleChart_ApplyPatchBump(t *testing.T) {
	repo := setupFixture(t, "single-chart")

	// Load, bump, and write changelog programmatically (same as changeset apply path)
	chartPath := filepath.Join(repo, "Chart.yaml")
	c, err := chart.Load(chartPath)
	if err != nil {
		t.Fatal(err)
	}

	if c.Name != "myapp" || c.Version != "0.1.0" {
		t.Fatalf("unexpected chart: name=%q version=%q", c.Name, c.Version)
	}

	newVer, err := chart.BumpVersion(c.Version, "patch")
	if err != nil {
		t.Fatal(err)
	}
	if newVer != "0.1.1" {
		t.Fatalf("expected 0.1.1, got %q", newVer)
	}

	if err := c.SetVersion(newVer); err != nil {
		t.Fatal(err)
	}
	if err := changelog.Prepend(c.Dir, newVer, "Fixed a minor bug in deployment template."); err != nil {
		t.Fatal(err)
	}

	// Verify Chart.yaml
	reloaded, err := chart.Load(chartPath)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version != "0.1.1" {
		t.Errorf("Chart.yaml version should be 0.1.1, got %q", reloaded.Version)
	}
	// Name and other fields preserved
	if reloaded.Name != "myapp" {
		t.Errorf("name should be preserved, got %q", reloaded.Name)
	}

	// Verify CHANGELOG.md
	cl := readFile(t, filepath.Join(repo, "CHANGELOG.md"))
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(cl, "## 0.1.1 ("+today+")") {
		t.Errorf("changelog missing version heading:\n%s", cl)
	}
	if !strings.Contains(cl, "Fixed a minor bug") {
		t.Errorf("changelog missing message:\n%s", cl)
	}
}

func TestAcceptance_SingleChart_CleanAfterBumpCommit(t *testing.T) {
	repo := setupFixture(t, "single-chart")

	// Edit values, commit
	writeFile(t, filepath.Join(repo, "values.yaml"), "replicaCount: 10\n")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "change values")

	// Verify stale
	out, code := helmver(t, repo, "check")
	if code != 1 {
		t.Fatalf("should be stale, got exit %d: %s", code, out)
	}

	// Bump version and commit
	c, _ := chart.Load(filepath.Join(repo, "Chart.yaml"))
	newVer, _ := chart.BumpVersion(c.Version, "patch")
	c.SetVersion(newVer)
	changelog.Prepend(c.Dir, newVer, "Bumped replicas.")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "bump to "+newVer)

	// Verify clean
	out, code = helmver(t, repo, "check")
	if code != 0 {
		t.Errorf("should be clean after bump, got exit %d: %s", code, out)
	}
}

// ===================================================================
// Monorepo acceptance tests
// ===================================================================

func TestAcceptance_Monorepo_CheckClean(t *testing.T) {
	repo := setupFixture(t, "monorepo")

	out, code := helmver(t, repo, "check", "--dir", filepath.Join(repo, "charts"))
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. output:\n%s", code, out)
	}
}

func TestAcceptance_Monorepo_OneChartStale(t *testing.T) {
	repo := setupFixture(t, "monorepo")

	// Modify only the worker chart
	writeFile(t, filepath.Join(repo, "charts", "worker", "values.yaml"),
		"replicaCount: 3\nimage:\n  repository: worker\n  tag: \"1.0.0\"\n")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "scale worker")

	out, code := helmver(t, repo, "check", "--dir", filepath.Join(repo, "charts"))
	if code != 1 {
		t.Fatalf("expected exit 1, got %d. output:\n%s", code, out)
	}
	if !strings.Contains(out, "worker") {
		t.Errorf("expected worker in stale list:\n%s", out)
	}
	if strings.Contains(out, "api") {
		t.Errorf("api should not be stale:\n%s", out)
	}
	if strings.Contains(out, "web") {
		t.Errorf("web should not be stale:\n%s", out)
	}
}

func TestAcceptance_Monorepo_AllChartsStale(t *testing.T) {
	repo := setupFixture(t, "monorepo")

	for _, name := range []string{"api", "web", "worker"} {
		writeFile(t, filepath.Join(repo, "charts", name, "values.yaml"), "replicaCount: 99\n")
	}
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "update all")

	out, code := helmver(t, repo, "check", "--dir", filepath.Join(repo, "charts"))
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(out, "3 chart(s)") {
		t.Errorf("expected 3 stale charts:\n%s", out)
	}
}

func TestAcceptance_Monorepo_BumpOneLeaveOthersClean(t *testing.T) {
	repo := setupFixture(t, "monorepo")

	// Edit all three
	for _, name := range []string{"api", "web", "worker"} {
		writeFile(t, filepath.Join(repo, "charts", name, "values.yaml"), "replicaCount: 99\n")
	}
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "update all")

	// Bump only api
	c, _ := chart.Load(filepath.Join(repo, "charts", "api", "Chart.yaml"))
	newVer, _ := chart.BumpVersion(c.Version, "minor")
	c.SetVersion(newVer)
	changelog.Prepend(c.Dir, newVer, "Added new endpoint.")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "bump api to "+newVer)

	// api should be clean, web and worker still stale
	out, code := helmver(t, repo, "check", "--dir", filepath.Join(repo, "charts"))
	if code != 1 {
		t.Fatalf("expected exit 1 (web+worker stale), got %d", code)
	}
	if !strings.Contains(out, "2 chart(s)") {
		t.Errorf("expected 2 stale charts, got:\n%s", out)
	}
	if strings.Contains(out, "api") {
		t.Errorf("api should not be stale after bump:\n%s", out)
	}
}

func TestAcceptance_Monorepo_ApplyMultipleBumps(t *testing.T) {
	repo := setupFixture(t, "monorepo")

	// Programmatically bump api (minor) and worker (patch)
	apiChart, err := chart.Load(filepath.Join(repo, "charts", "api", "Chart.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	workerChart, err := chart.Load(filepath.Join(repo, "charts", "worker", "Chart.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	apiNew, _ := chart.BumpVersion(apiChart.Version, "minor")
	workerNew, _ := chart.BumpVersion(workerChart.Version, "patch")

	if apiNew != "1.3.0" {
		t.Fatalf("expected api 1.3.0, got %q", apiNew)
	}
	if workerNew != "0.5.1" {
		t.Fatalf("expected worker 0.5.1, got %q", workerNew)
	}

	apiChart.SetVersion(apiNew)
	changelog.Prepend(apiChart.Dir, apiNew, "New list endpoint for paginated results.")
	workerChart.SetVersion(workerNew)
	changelog.Prepend(workerChart.Dir, workerNew, "Fixed retry backoff logic.")

	// Verify both Chart.yaml files
	a, _ := chart.Load(filepath.Join(repo, "charts", "api", "Chart.yaml"))
	w, _ := chart.Load(filepath.Join(repo, "charts", "worker", "Chart.yaml"))
	if a.Version != "1.3.0" {
		t.Errorf("api version should be 1.3.0, got %q", a.Version)
	}
	if w.Version != "0.5.1" {
		t.Errorf("worker version should be 0.5.1, got %q", w.Version)
	}

	// Verify changelogs
	today := time.Now().Format("2006-01-02")
	apiCL := readFile(t, filepath.Join(repo, "charts", "api", "CHANGELOG.md"))
	if !strings.Contains(apiCL, "## 1.3.0 ("+today+")") {
		t.Errorf("api changelog wrong:\n%s", apiCL)
	}
	workerCL := readFile(t, filepath.Join(repo, "charts", "worker", "CHANGELOG.md"))
	if !strings.Contains(workerCL, "## 0.5.1 ("+today+")") {
		t.Errorf("worker changelog wrong:\n%s", workerCL)
	}

	// web should be untouched
	web, _ := chart.Load(filepath.Join(repo, "charts", "web", "Chart.yaml"))
	if web.Version != "2.0.0" {
		t.Errorf("web should be untouched at 2.0.0, got %q", web.Version)
	}
}

// ===================================================================
// Complex chart acceptance tests (comments, v-prefix, existing changelog)
// ===================================================================

func TestAcceptance_ComplexChart_PreservesComments(t *testing.T) {
	repo := setupFixture(t, "complex-chart")

	c, err := chart.Load(filepath.Join(repo, "Chart.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if c.Version != "v1.5.2" {
		t.Fatalf("expected v1.5.2, got %q", c.Version)
	}

	newVer, _ := chart.BumpVersion(c.Version, "minor")
	if newVer != "v1.6.0" {
		t.Fatalf("expected v1.6.0, got %q", newVer)
	}

	c.SetVersion(newVer)

	// Re-read raw file and verify comments survived
	raw := readFile(t, filepath.Join(repo, "Chart.yaml"))
	if !strings.Contains(raw, "# This is the chart version") {
		t.Error("comment before version was lost")
	}
	if !strings.Contains(raw, "# App version tracks the upstream") {
		t.Error("comment before appVersion was lost")
	}
	if !strings.Contains(raw, "version: v1.6.0") {
		t.Errorf("version not updated in raw file:\n%s", raw)
	}
	// Dependencies should still be there
	if !strings.Contains(raw, "postgresql") {
		t.Error("dependencies section was lost")
	}
}

func TestAcceptance_ComplexChart_PrependsExistingChangelog(t *testing.T) {
	repo := setupFixture(t, "complex-chart")
	today := time.Now().Format("2006-01-02")

	c, _ := chart.Load(filepath.Join(repo, "Chart.yaml"))
	newVer, _ := chart.BumpVersion(c.Version, "patch")
	c.SetVersion(newVer)
	changelog.Prepend(c.Dir, newVer, "Fixed edge case in query parser.")

	cl := readFile(t, filepath.Join(repo, "CHANGELOG.md"))

	// New entry should be before existing entries
	newIdx := strings.Index(cl, "## v1.5.3 ("+today+")")
	oldIdx := strings.Index(cl, "## v1.5.2 (2025-06-15)")

	if newIdx == -1 {
		t.Fatalf("new version heading not found:\n%s", cl)
	}
	if oldIdx == -1 {
		t.Fatalf("old version heading lost:\n%s", cl)
	}
	if newIdx >= oldIdx {
		t.Errorf("new entry (pos %d) should precede old entry (pos %d):\n%s", newIdx, oldIdx, cl)
	}

	// All old entries should still be there
	if !strings.Contains(cl, "v1.5.1") || !strings.Contains(cl, "v1.5.0") {
		t.Errorf("older changelog entries lost:\n%s", cl)
	}
}

// ===================================================================
// Subchart acceptance tests
// ===================================================================

func TestAcceptance_Subchart_DiscoversAll(t *testing.T) {
	repo := setupFixture(t, "subchart-parent")

	paths, err := chart.Discover(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 charts (parent + redis subchart), got %d: %v", len(paths), paths)
	}

	names := map[string]bool{}
	for _, p := range paths {
		c, err := chart.Load(p)
		if err != nil {
			t.Fatal(err)
		}
		names[c.Name] = true
	}
	if !names["parent-app"] {
		t.Error("parent-app not found")
	}
	if !names["redis"] {
		t.Error("redis subchart not found")
	}
}

func TestAcceptance_Subchart_CheckIndependent(t *testing.T) {
	repo := setupFixture(t, "subchart-parent")

	// Modify only the redis subchart
	writeFile(t, filepath.Join(repo, "charts", "redis", "values.yaml"),
		"port: 6380\npersistence:\n  enabled: true\n")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "update redis port")

	out, code := helmver(t, repo, "check")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d. output:\n%s", code, out)
	}
	// redis should be stale (its chart dir changed)
	if !strings.Contains(out, "redis") {
		t.Errorf("expected redis in stale list:\n%s", out)
	}
}

func TestAcceptance_Subchart_BumpSubchartOnly(t *testing.T) {
	repo := setupFixture(t, "subchart-parent")

	redis, err := chart.Load(filepath.Join(repo, "charts", "redis", "Chart.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	newVer, _ := chart.BumpVersion(redis.Version, "minor")
	if newVer != "0.2.0" {
		t.Fatalf("expected 0.2.0, got %q", newVer)
	}

	redis.SetVersion(newVer)
	changelog.Prepend(redis.Dir, newVer, "Added persistence support.")

	// Verify redis bumped
	r, _ := chart.Load(filepath.Join(repo, "charts", "redis", "Chart.yaml"))
	if r.Version != "0.2.0" {
		t.Errorf("redis version should be 0.2.0, got %q", r.Version)
	}

	// Parent should be untouched
	parent, _ := chart.Load(filepath.Join(repo, "Chart.yaml"))
	if parent.Version != "3.0.0" {
		t.Errorf("parent should be untouched at 3.0.0, got %q", parent.Version)
	}
}

// ===================================================================
// Non-git acceptance tests (changeset path)
// ===================================================================

func TestAcceptance_NonGit_DiscoverAndLoad(t *testing.T) {
	// Copy fixture without git -- simulates running helmver changeset outside a repo
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir(), "monorepo"), dir)

	paths, err := chart.Discover(filepath.Join(dir, "charts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 charts, got %d", len(paths))
	}

	hasGit := git.IsRepo(dir)
	if hasGit {
		t.Fatal("should not be a git repo")
	}

	for _, p := range paths {
		c, err := chart.Load(p)
		if err != nil {
			t.Fatalf("failed to load %s: %v", p, err)
		}
		if c.Stale {
			t.Errorf("%s should not be stale without git", c.Name)
		}
	}
}

func TestAcceptance_NonGit_ApplyBumpAndChangelog(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir(), "single-chart"), dir)
	today := time.Now().Format("2006-01-02")

	c, err := chart.Load(filepath.Join(dir, "Chart.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	newVer, _ := chart.BumpVersion(c.Version, "major")
	if newVer != "1.0.0" {
		t.Fatalf("expected 1.0.0, got %q", newVer)
	}

	c.SetVersion(newVer)
	changelog.Prepend(c.Dir, newVer, "Breaking: restructured values schema.")

	// Verify
	reloaded, _ := chart.Load(filepath.Join(dir, "Chart.yaml"))
	if reloaded.Version != "1.0.0" {
		t.Errorf("expected 1.0.0, got %q", reloaded.Version)
	}

	cl := readFile(t, filepath.Join(dir, "CHANGELOG.md"))
	if !strings.Contains(cl, "## 1.0.0 ("+today+")") {
		t.Errorf("changelog missing entry:\n%s", cl)
	}
	if !strings.Contains(cl, "Breaking: restructured values schema") {
		t.Errorf("changelog missing message:\n%s", cl)
	}
}

// ===================================================================
// helmver check via binary against fixtures
// ===================================================================

func TestAcceptance_Binary_CheckMonorepoWithDir(t *testing.T) {
	repo := setupFixture(t, "monorepo")

	// Clean state
	out, code := helmver(t, repo, "check", "--dir", filepath.Join(repo, "charts"))
	if code != 0 {
		t.Fatalf("expected clean, got %d:\n%s", code, out)
	}

	// Edit one chart, verify stale
	writeFile(t, filepath.Join(repo, "charts", "web", "values.yaml"), "replicaCount: 100\n")
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-m", "scale web")

	out, code = helmver(t, repo, "check", "--dir", filepath.Join(repo, "charts"))
	if code != 1 {
		t.Fatalf("expected stale, got %d:\n%s", code, out)
	}
	if !strings.Contains(out, "web") {
		t.Errorf("expected web in output:\n%s", out)
	}
	if !strings.Contains(out, "1 chart(s)") {
		t.Errorf("expected 1 stale chart:\n%s", out)
	}
}

func TestAcceptance_Binary_CheckSubchart(t *testing.T) {
	repo := setupFixture(t, "subchart-parent")

	out, code := helmver(t, repo, "check")
	if code != 0 {
		t.Fatalf("expected clean, got %d:\n%s", code, out)
	}
}
