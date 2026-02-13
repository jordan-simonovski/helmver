package chart

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Chart holds the parsed metadata we care about plus the raw YAML tree.
type Chart struct {
	Name    string
	Version string
	Path    string // absolute path to Chart.yaml
	Dir     string // directory containing Chart.yaml
	Stale   bool   // true if chart has changes since last version bump
	doc     yaml.Node
}

// Load reads and parses a Chart.yaml, preserving the full YAML tree.
func Load(path string) (*Chart, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	c := &Chart{
		Path: path,
		Dir:  filepath.Dir(path),
		doc:  doc,
	}

	// doc.Content[0] is the mapping node
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s: unexpected YAML structure", path)
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: expected mapping at top level", path)
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		key := mapping.Content[i]
		val := mapping.Content[i+1]
		switch key.Value {
		case "name":
			c.Name = val.Value
		case "version":
			c.Version = val.Value
		}
	}

	if c.Version == "" {
		return nil, fmt.Errorf("%s: missing version field", path)
	}

	return c, nil
}

// SetVersion updates the version field in the YAML tree and writes it back to disk.
func (c *Chart) SetVersion(newVersion string) error {
	mapping := c.doc.Content[0]
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		key := mapping.Content[i]
		val := mapping.Content[i+1]
		if key.Value == "version" {
			val.Value = newVersion
			c.Version = newVersion
			break
		}
	}
	return c.write()
}

func (c *Chart) write() error {
	f, err := os.Create(c.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(&c.doc); err != nil {
		return err
	}
	return enc.Close()
}

// BumpVersion computes the next semver given a bump type.
func BumpVersion(current string, bump string) (string, error) {
	// Strip leading 'v' if present
	trimmed := strings.TrimPrefix(current, "v")
	parts := strings.SplitN(trimmed, ".", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("version %q is not valid semver (expected X.Y.Z)", current)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid patch version %q: %w", parts[2], err)
	}

	switch bump {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	default:
		return "", fmt.Errorf("unknown bump type %q", bump)
	}

	prefix := ""
	if strings.HasPrefix(current, "v") {
		prefix = "v"
	}
	return fmt.Sprintf("%s%d.%d.%d", prefix, major, minor, patch), nil
}
