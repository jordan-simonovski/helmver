package chart

import (
	"os"
	"path/filepath"
)

// Discover recursively walks dir and returns paths to all Chart.yaml files found.
// Paths matching any of the exclude glob patterns (matched against the path
// relative to dir) are skipped. Directories that match an exclude pattern are
// not descended into.
func Discover(dir string, exclude []string) ([]string, error) {
	var charts []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}

		if excluded(rel, exclude) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}
		if d.Name() == "Chart.yaml" || d.Name() == "Chart.yml" {
			charts = append(charts, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return charts, nil
}

func excluded(rel string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, rel); matched {
			return true
		}
		// Also match against each path component so "*/4.*/*" style
		// patterns work, and match against just the base name.
		if matched, _ := filepath.Match(p, filepath.Base(rel)); matched {
			return true
		}
	}
	return false
}
