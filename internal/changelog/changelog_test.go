package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrependNewFile(t *testing.T) {
	dir := t.TempDir()
	today := time.Now().Format("2006-01-02")

	if err := Prepend(dir, "1.0.0", "Initial release"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(data)
	if !strings.Contains(s, "# Changelog") {
		t.Error("missing top-level heading")
	}
	if !strings.Contains(s, "## 1.0.0 ("+today+")") {
		t.Errorf("missing version heading, got:\n%s", s)
	}
	if !strings.Contains(s, "Initial release") {
		t.Error("missing message")
	}
}

func TestPrependExistingFile(t *testing.T) {
	dir := t.TempDir()
	today := time.Now().Format("2006-01-02")

	existing := "# Changelog\n\n## 0.1.0 (2025-01-01)\n\nOld entry\n"
	if err := os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Prepend(dir, "0.2.0", "New feature"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(data)

	// New entry should come before old entry
	newIdx := strings.Index(s, "## 0.2.0 ("+today+")")
	oldIdx := strings.Index(s, "## 0.1.0 (2025-01-01)")

	if newIdx == -1 {
		t.Fatalf("new version heading not found in:\n%s", s)
	}
	if oldIdx == -1 {
		t.Fatalf("old version heading not found in:\n%s", s)
	}
	if newIdx >= oldIdx {
		t.Errorf("new entry (pos %d) should be before old entry (pos %d) in:\n%s", newIdx, oldIdx, s)
	}

	if !strings.Contains(s, "New feature") {
		t.Error("missing new message")
	}
	if !strings.Contains(s, "Old entry") {
		t.Error("old message was lost")
	}
}

func TestPrependMultilineMessage(t *testing.T) {
	dir := t.TempDir()

	msg := "Line one\nLine two\nLine three"
	if err := Prepend(dir, "1.0.0", msg); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(data)
	if !strings.Contains(s, "Line one\nLine two\nLine three") {
		t.Errorf("multiline message not preserved in:\n%s", s)
	}
}
