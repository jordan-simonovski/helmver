package chart

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDiscoverSingleChart(t *testing.T) {
	dir := t.TempDir()
	chartPath := filepath.Join(dir, "Chart.yaml")
	if err := os.WriteFile(chartPath, []byte("apiVersion: v2\nname: x\nversion: 0.1.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Discover(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 chart, got %d", len(got))
	}
	if got[0] != chartPath {
		t.Errorf("got %q, want %q", got[0], chartPath)
	}
}

func TestDiscoverMonorepo(t *testing.T) {
	dir := t.TempDir()

	charts := []string{
		filepath.Join(dir, "charts", "api", "Chart.yaml"),
		filepath.Join(dir, "charts", "web", "Chart.yaml"),
		filepath.Join(dir, "charts", "worker", "Chart.yml"),
	}
	for _, p := range charts {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("apiVersion: v2\nname: x\nversion: 0.1.0\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Also create a non-chart yaml to make sure it's excluded
	other := filepath.Join(dir, "charts", "api", "values.yaml")
	if err := os.WriteFile(other, []byte("key: val\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Discover(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	sort.Strings(got)
	sort.Strings(charts)

	if len(got) != len(charts) {
		t.Fatalf("expected %d charts, got %d: %v", len(charts), len(got), got)
	}
	for i := range got {
		if got[i] != charts[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], charts[i])
		}
	}
}

func TestDiscoverEmpty(t *testing.T) {
	dir := t.TempDir()
	got, err := Discover(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 charts, got %d", len(got))
	}
}

func TestDiscoverExcludePattern(t *testing.T) {
	dir := t.TempDir()

	// Simulate vendored upstream charts alongside a wrapper chart
	mkDir := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("apiVersion: v2\nname: x\nversion: 0.1.0\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Their own chart
	mkDir(filepath.Join(dir, "ingress-nginx", "Chart.yaml"))
	// Vendored upstream under version directories
	mkDir(filepath.Join(dir, "ingress-nginx", "4.13.3", "ingress-nginx", "Chart.yaml"))
	mkDir(filepath.Join(dir, "ingress-nginx", "4.12.0", "ingress-nginx", "Chart.yaml"))

	// Without exclude: finds all 3
	all, err := Discover(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 charts without exclude, got %d", len(all))
	}

	// Exclude version directories (digit-prefixed names)
	filtered, err := Discover(dir, []string{"[0-9]*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 chart with exclude, got %d: %v", len(filtered), filtered)
	}
}

func TestDiscoverExcludeByDirectoryName(t *testing.T) {
	dir := t.TempDir()

	mkDir := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("apiVersion: v2\nname: x\nversion: 0.1.0\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	mkDir(filepath.Join(dir, "api", "Chart.yaml"))
	mkDir(filepath.Join(dir, "web", "Chart.yaml"))
	mkDir(filepath.Join(dir, "vendor", "upstream", "Chart.yaml"))

	filtered, err := Discover(dir, []string{"vendor"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 charts, got %d: %v", len(filtered), filtered)
	}
}
