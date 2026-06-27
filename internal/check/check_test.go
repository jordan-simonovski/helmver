package check_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jordan-simonovski/helmver/internal/changeset"
	"github.com/jordan-simonovski/helmver/internal/check"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
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

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	return dir
}

func TestRun_staleChart(t *testing.T) {
	dir := initRepo(t)
	mkFile(t, filepath.Join(dir, "Chart.yaml"), "apiVersion: v2\nname: myapp\nversion: 1.0.0\n")
	mkFile(t, filepath.Join(dir, "values.yaml"), "key: val\n")
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", "init")
	gitRun(t, dir, "branch", "base")

	mkFile(t, filepath.Join(dir, "values.yaml"), "key: changed\n")
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", "change values")

	result, err := check.Run(check.Options{Dir: dir, Base: "base"})
	if err != nil {
		t.Fatal(err)
	}
	if result.AllUpToDate || len(result.StaleCharts) != 1 || result.StaleCharts[0].Name != "myapp" {
		t.Fatalf("expected one stale chart, got %+v", result)
	}
}

func TestRun_changesetCoverage(t *testing.T) {
	dir := initRepo(t)
	mkFile(t, filepath.Join(dir, "Chart.yaml"), "apiVersion: v2\nname: myapp\nversion: 1.0.0\n")
	mkFile(t, filepath.Join(dir, "values.yaml"), "key: val\n")
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", "init")
	gitRun(t, dir, "branch", "base")

	mkFile(t, filepath.Join(dir, "values.yaml"), "key: changed\n")
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", "change values")

	mkFile(t, filepath.Join(dir, ".helmver", "001.md"), "---\n\"myapp\": patch\n---\n\nWill bump\n")

	result, err := check.Run(check.Options{
		Dir:              dir,
		Base:             "base",
		RequireChangeset: true,
		ChangesetRoot:    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.AllUpToDate || len(result.CoveredCharts) != 1 {
		t.Fatalf("expected covered chart, got %+v", result)
	}
}

func TestRun_missingBaseRef(t *testing.T) {
	dir := initRepo(t)
	mkFile(t, filepath.Join(dir, "Chart.yaml"), "apiVersion: v2\nname: myapp\nversion: 1.0.0\n")
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", "init")

	_, err := check.Run(check.Options{Dir: dir, Base: "missing-ref"})
	if err == nil {
		t.Fatal("expected error for missing base ref")
	}
	if !strings.Contains(err.Error(), "base ref") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_notGitRepo(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "Chart.yaml"), "apiVersion: v2\nname: myapp\nversion: 1.0.0\n")

	_, err := check.Run(check.Options{Dir: dir})
	if !errors.Is(err, check.ErrNotGitRepo) {
		t.Fatalf("expected ErrNotGitRepo, got %v", err)
	}
}

func TestRun_invalidChart(t *testing.T) {
	dir := initRepo(t)
	mkFile(t, filepath.Join(dir, "Chart.yaml"), "not: valid: yaml: [\n")
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-m", "init")
	gitRun(t, dir, "branch", "base")

	_, err := check.Run(check.Options{Dir: dir, Base: "base"})
	if err == nil {
		t.Fatal("expected error for invalid Chart.yaml")
	}
	if !strings.Contains(err.Error(), "loading") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatMarkdown_stableChangesetOrder(t *testing.T) {
	result := &check.Result{
		AllUpToDate: true,
		CoveredCharts: []check.ChartResult{
			{Name: "api", Version: "1.0.0", Dir: "/charts/api", HasChangeset: true},
		},
		Changesets: []*changeset.File{
			{Entries: []changeset.Entry{{Chart: "zebra", Bump: "patch"}, {Chart: "api", Bump: "minor"}}},
		},
	}
	out, err := check.Format(result, "markdown", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	zebraIdx := strings.Index(out, "| zebra |")
	apiIdx := strings.Index(out, "| api |")
	if zebraIdx < 0 || apiIdx < 0 {
		t.Fatalf("expected both charts in table, got:\n%s", out)
	}
	if zebraIdx > apiIdx {
		t.Fatalf("expected stable first-seen order (zebra before api), got:\n%s", out)
	}
}

func TestFormatMarkdown_noCommitSHA(t *testing.T) {
	result := &check.Result{AllUpToDate: true}
	out, err := check.Format(result, "markdown", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Latest commit") {
		t.Fatalf("expected no commit line when SHA is empty, got:\n%s", out)
	}
}
