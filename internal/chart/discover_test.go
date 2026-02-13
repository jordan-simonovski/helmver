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

	got, err := Discover(dir)
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

	got, err := Discover(dir)
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
	got, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 charts, got %d", len(got))
	}
}
