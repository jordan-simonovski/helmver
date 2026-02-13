package chart

import (
	"os"
	"path/filepath"
)

// Discover recursively walks dir and returns paths to all Chart.yaml files found.
func Discover(dir string) ([]string, error) {
	var charts []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
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
