package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const header = "# Changelog\n"

// Prepend adds a new version entry to the top of CHANGELOG.md in the given directory.
// If CHANGELOG.md does not exist, it is created with a top-level heading.
func Prepend(dir, version, message string) error {
	path := filepath.Join(dir, "CHANGELOG.md")
	date := time.Now().Format("2006-01-02")

	entry := fmt.Sprintf("## %s (%s)\n\n%s\n", version, date, strings.TrimRight(message, "\n"))

	existing, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		// File does not exist; create with header
		content := header + "\n" + entry + "\n"
		return os.WriteFile(path, []byte(content), 0o644)
	}

	// Insert the new entry after the top-level heading
	content := string(existing)
	if idx := strings.Index(content, "\n"); idx != -1 && strings.HasPrefix(content, "# ") {
		// Insert after the first heading line
		before := content[:idx+1]
		after := content[idx+1:]
		content = before + "\n" + entry + "\n" + after
	} else {
		// No recognizable heading; just prepend
		content = header + "\n" + entry + "\n" + content
	}

	return os.WriteFile(path, []byte(content), 0o644)
}
