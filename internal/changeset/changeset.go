package changeset

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Entry represents one chart's intended change within a changeset file.
type Entry struct {
	Chart string // chart name
	Bump  string // "patch", "minor", or "major"
}

// File represents a parsed .helmver changeset file.
type File struct {
	Path    string
	Entries []Entry
	Message string
}

// Resolved holds the aggregated bump and collected messages for one chart
// across all changeset files.
type Resolved struct {
	Chart    string
	Bump     string
	Messages []string
}

// Dir returns the .helmver directory path under root.
func Dir(root string) string {
	return filepath.Join(root, ".helmver")
}

// Write creates a new changeset file in .helmver/ with a random ID.
// Returns the absolute path of the created file.
func Write(root string, entries []Entry, message string) (string, error) {
	dir := Dir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating .helmver directory: %w", err)
	}

	id, err := randomID()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, id+".md")

	var b strings.Builder
	b.WriteString("---\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "%q: %s\n", e.Chart, e.Bump)
	}
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimRight(message, "\n"))
	b.WriteString("\n")

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("writing changeset: %w", err)
	}
	return path, nil
}

// Discover finds and parses all changeset files in .helmver/.
func Discover(root string) ([]*File, error) {
	dir := Dir(root)
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading .helmver: %w", err)
	}

	var files []*File
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := Parse(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		files = append(files, f)
	}
	return files, nil
}

// Parse reads and validates a single changeset file.
func Parse(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("missing front matter opening in %s", path)
	}

	rest := content[4:]
	idx := strings.Index(rest, "---\n")
	if idx < 0 {
		return nil, fmt.Errorf("missing front matter closing in %s", path)
	}

	frontMatter := rest[:idx]
	message := strings.TrimSpace(rest[idx+4:])

	var entries []Entry
	for _, line := range strings.Split(frontMatter, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.LastIndex(line, ":")
		if colonIdx < 0 {
			return nil, fmt.Errorf("invalid entry %q in %s", line, path)
		}
		name := strings.TrimSpace(line[:colonIdx])
		name = strings.Trim(name, "\"")
		bump := strings.TrimSpace(line[colonIdx+1:])

		if bump != "patch" && bump != "minor" && bump != "major" {
			return nil, fmt.Errorf("invalid bump type %q for chart %q in %s", bump, name, path)
		}
		entries = append(entries, Entry{Chart: name, Bump: bump})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries in %s", path)
	}

	return &File{
		Path:    path,
		Entries: entries,
		Message: message,
	}, nil
}

// ChartNames returns the set of chart names referenced across all files.
func ChartNames(files []*File) map[string]bool {
	names := make(map[string]bool)
	for _, f := range files {
		for _, e := range f.Entries {
			names[e.Chart] = true
		}
	}
	return names
}

// Aggregate groups entries by chart name across all files.
// For each chart, the highest bump wins and messages are collected in order.
func Aggregate(files []*File) map[string]*Resolved {
	m := make(map[string]*Resolved)
	for _, f := range files {
		for _, e := range f.Entries {
			r, ok := m[e.Chart]
			if !ok {
				r = &Resolved{Chart: e.Chart, Bump: e.Bump}
				m[e.Chart] = r
			} else if bumpRank(e.Bump) > bumpRank(r.Bump) {
				r.Bump = e.Bump
			}
			if f.Message != "" {
				r.Messages = append(r.Messages, f.Message)
			}
		}
	}
	return m
}

// Remove deletes a consumed changeset file.
func Remove(path string) error {
	return os.Remove(path)
}

func bumpRank(bump string) int {
	switch bump {
	case "major":
		return 3
	case "minor":
		return 2
	case "patch":
		return 1
	default:
		return 0
	}
}

func randomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}
