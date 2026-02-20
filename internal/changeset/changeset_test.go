package changeset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndParse(t *testing.T) {
	dir := t.TempDir()
	entries := []Entry{
		{Chart: "api", Bump: "minor"},
		{Chart: "worker", Bump: "patch"},
	}

	path, err := Write(dir, entries, "added autoscaling")
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	f, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(f.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(f.Entries))
	}
	if f.Entries[0].Chart != "api" || f.Entries[0].Bump != "minor" {
		t.Errorf("entry 0: got %+v", f.Entries[0])
	}
	if f.Entries[1].Chart != "worker" || f.Entries[1].Bump != "patch" {
		t.Errorf("entry 1: got %+v", f.Entries[1])
	}
	if f.Message != "added autoscaling" {
		t.Errorf("message: got %q", f.Message)
	}
}

func TestDiscover_Empty(t *testing.T) {
	dir := t.TempDir()
	files, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestDiscover_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	if _, err := Write(dir, []Entry{{Chart: "api", Bump: "minor"}}, "msg1"); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(dir, []Entry{{Chart: "web", Bump: "major"}}, "msg2"); err != nil {
		t.Fatal(err)
	}

	files, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestAggregate_HighestBumpWins(t *testing.T) {
	files := []*File{
		{Entries: []Entry{{Chart: "api", Bump: "patch"}}, Message: "fix1"},
		{Entries: []Entry{{Chart: "api", Bump: "minor"}}, Message: "feat1"},
		{Entries: []Entry{{Chart: "api", Bump: "patch"}}, Message: "fix2"},
	}

	resolved := Aggregate(files)
	r := resolved["api"]
	if r == nil {
		t.Fatal("expected resolved entry for api")
	}
	if r.Bump != "minor" {
		t.Errorf("expected minor, got %q", r.Bump)
	}
	if len(r.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(r.Messages))
	}
}

func TestAggregate_MultipleCharts(t *testing.T) {
	files := []*File{
		{Entries: []Entry{
			{Chart: "api", Bump: "minor"},
			{Chart: "web", Bump: "patch"},
		}, Message: "shared change"},
		{Entries: []Entry{{Chart: "web", Bump: "major"}}, Message: "breaking"},
	}

	resolved := Aggregate(files)
	if resolved["api"].Bump != "minor" {
		t.Errorf("api: expected minor, got %q", resolved["api"].Bump)
	}
	if resolved["web"].Bump != "major" {
		t.Errorf("web: expected major, got %q", resolved["web"].Bump)
	}
}

func TestChartNames(t *testing.T) {
	files := []*File{
		{Entries: []Entry{{Chart: "a", Bump: "patch"}}},
		{Entries: []Entry{{Chart: "b", Bump: "minor"}, {Chart: "a", Bump: "minor"}}},
	}
	names := ChartNames(files)
	if !names["a"] || !names["b"] {
		t.Errorf("expected a and b, got %v", names)
	}
	if names["c"] {
		t.Error("unexpected chart c")
	}
}

func TestParse_InvalidFrontMatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.md")
	if err := os.WriteFile(path, []byte("no front matter here"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for missing front matter")
	}
}

func TestParse_InvalidBumpType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.md")
	if err := os.WriteFile(path, []byte("---\n\"api\": huge\n---\n\nmsg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for invalid bump type")
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	path, _ := Write(dir, []Entry{{Chart: "x", Bump: "patch"}}, "msg")
	if err := Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}
