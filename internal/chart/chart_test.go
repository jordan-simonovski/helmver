package chart

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBumpVersion(t *testing.T) {
	tests := []struct {
		current string
		bump    string
		want    string
		wantErr bool
	}{
		{"0.1.0", "patch", "0.1.1", false},
		{"0.1.0", "minor", "0.2.0", false},
		{"0.1.0", "major", "1.0.0", false},
		{"1.2.3", "patch", "1.2.4", false},
		{"1.2.3", "minor", "1.3.0", false},
		{"1.2.3", "major", "2.0.0", false},
		// v-prefix preserved
		{"v1.0.0", "patch", "v1.0.1", false},
		{"v0.0.0", "major", "v1.0.0", false},
		// errors
		{"1.2", "patch", "", true},
		{"abc", "patch", "", true},
		{"1.2.3", "bogus", "", true},
		{"1.x.3", "patch", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+tt.bump, func(t *testing.T) {
			got, err := BumpVersion(tt.current, tt.bump)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("BumpVersion(%q, %q) = %q, want %q", tt.current, tt.bump, got, tt.want)
			}
		})
	}
}

func TestLoadAndSetVersion(t *testing.T) {
	dir := t.TempDir()
	chartPath := filepath.Join(dir, "Chart.yaml")

	content := `apiVersion: v2
name: test-chart
description: A test chart
version: 1.0.0
appVersion: "1.0"
`
	if err := os.WriteFile(chartPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := Load(chartPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if c.Name != "test-chart" {
		t.Errorf("Name = %q, want %q", c.Name, "test-chart")
	}
	if c.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", c.Version, "1.0.0")
	}
	if c.Dir != dir {
		t.Errorf("Dir = %q, want %q", c.Dir, dir)
	}

	// SetVersion and re-read
	if err := c.SetVersion("2.0.0"); err != nil {
		t.Fatalf("SetVersion failed: %v", err)
	}
	if c.Version != "2.0.0" {
		t.Errorf("after SetVersion, Version = %q, want %q", c.Version, "2.0.0")
	}

	// Re-load from disk to confirm persistence
	c2, err := Load(chartPath)
	if err != nil {
		t.Fatalf("re-Load failed: %v", err)
	}
	if c2.Version != "2.0.0" {
		t.Errorf("re-loaded Version = %q, want %q", c2.Version, "2.0.0")
	}
	// Name and other fields should be preserved
	if c2.Name != "test-chart" {
		t.Errorf("re-loaded Name = %q, want %q", c2.Name, "test-chart")
	}
}

func TestLoadMissingVersion(t *testing.T) {
	dir := t.TempDir()
	chartPath := filepath.Join(dir, "Chart.yaml")

	content := `apiVersion: v2
name: no-version-chart
`
	if err := os.WriteFile(chartPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(chartPath)
	if err == nil {
		t.Fatal("expected error for missing version field")
	}
}

func TestLoadNonexistent(t *testing.T) {
	_, err := Load("/nonexistent/Chart.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSetVersionPreservesComments(t *testing.T) {
	dir := t.TempDir()
	chartPath := filepath.Join(dir, "Chart.yaml")

	content := `apiVersion: v2
name: commented-chart
# This is the chart version
version: 0.5.0
# App version tracks upstream
appVersion: "2.0"
`
	if err := os.WriteFile(chartPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := Load(chartPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if err := c.SetVersion("0.6.0"); err != nil {
		t.Fatalf("SetVersion failed: %v", err)
	}

	data, err := os.ReadFile(chartPath)
	if err != nil {
		t.Fatal(err)
	}

	s := string(data)
	// The comment should still be there
	if !contains(s, "# This is the chart version") {
		t.Error("comment before version was lost")
	}
	if !contains(s, "# App version tracks upstream") {
		t.Error("comment before appVersion was lost")
	}
	if !contains(s, "version: 0.6.0") {
		t.Error("version not updated to 0.6.0")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsBytes(s, substr))
}

func containsBytes(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
