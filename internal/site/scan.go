package site

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peacefixation/ssg/internal/config"
	"gopkg.in/yaml.v3"
)

var contentExts = map[string]bool{
	".md": true, ".markdown": true, ".json": true, ".yaml": true, ".yml": true,
}

// nodeMeta is the in-memory representation of a node.yaml file.
// It carries build overrides (template, cardTemplate, sortBy, sortOrder, limit)
// and an optional scanner name that selects a ChildScanner plugin.
type nodeMeta struct {
	Title        string `yaml:"title"`
	Scanner      string `yaml:"scanner"`   // e.g. "photos" — triggers plugin scanning
	Template     string `yaml:"template"`
	CardTemplate string `yaml:"cardTemplate"`
	SortBy       string `yaml:"sortBy"`
	SortOrder    string `yaml:"sortOrder"`
	Limit        int    `yaml:"limit"`
	PinnedField  string `yaml:"pinnedField"`
	PinnedValue  string `yaml:"pinnedValue"`
}

// nodeConfigFile returns the path to node.yaml in dir and true if found.
func nodeConfigFile(dir string) (string, bool) {
	p := filepath.Join(dir, "node.yaml")
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return "", false
}

// scanDir recursively walks dir and returns an ItemConfig for every discovered node.
// If the parent node's scanner is registered in nodePlugins as a ChildScanner,
// that scanner is used instead of the normal content scan.
func scanDir(dir, outputPrefix string, cfg *config.SiteConfig, parent nodeMeta) ([]config.NodeConfig, error) {
	if p, ok := nodePlugins[parent.Scanner]; ok {
		if scanner, ok := p.(ChildScanner); ok {
			return scanner.ScanChildren(dir, outputPrefix, cfg, parent)
		}
	}
	return scanContentDir(dir, outputPrefix, cfg, parent)
}

// scanContentDir scans a directory for content files and subdirectories:
//   - Files with a supported extension become leaf nodes.
//   - Subdirectories containing a node.yaml become branch nodes
//     whose Children are the result of recursively scanning that subdirectory.
func scanContentDir(dir, outputPrefix string, cfg *config.SiteConfig, parent nodeMeta) ([]config.NodeConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}

	var items []config.NodeConfig
	for _, entry := range entries {
		item, ok, err := scanNode(dir, entry.Name(), outputPrefix, cfg, parent)
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, item)
		}
	}
	return items, nil
}

// scanNode dispatches to scanDirNode or scanFileNode based on filesystem type.
func scanNode(parentDir, name, outputPrefix string, cfg *config.SiteConfig, parent nodeMeta) (config.NodeConfig, bool, error) {
	info, err := os.Stat(filepath.Join(parentDir, name))
	if err != nil {
		return config.NodeConfig{}, false, nil
	}
	if info.IsDir() {
		return scanDirNode(parentDir, name, outputPrefix, cfg, parent)
	}
	item, ok := scanFileNode(parentDir, name, outputPrefix, cfg)
	return item, ok, nil
}

// scanDirNode checks whether name is a directory node (contains node.yaml).
// Returns ok=false for directories without a node config (they are ignored).
func scanDirNode(parentDir, name, outputPrefix string, cfg *config.SiteConfig, parent nodeMeta) (config.NodeConfig, bool, error) {
	dir := filepath.Join(parentDir, name)
	cfgFile, found := nodeConfigFile(dir)
	if !found {
		return config.NodeConfig{}, false, nil
	}

	meta := readNodeMeta(cfgFile)

	// Resolve each setting: node.yaml → parent node → site.yaml defaults.
	tmpl := first(meta.Template, parent.Template, cfg.Defaults.Template)
	cardTemplate := first(meta.CardTemplate, parent.CardTemplate, cfg.Defaults.CardTemplate)
	sortBy := first(meta.SortBy, parent.SortBy, cfg.Defaults.SortBy)
	sortOrder := first(meta.SortOrder, parent.SortOrder, cfg.Defaults.SortOrder)
	limit := meta.Limit
	if limit == 0 {
		limit = parent.Limit
	}
	if limit == 0 {
		limit = cfg.Defaults.Limit
	}

	resolved := nodeMeta{
		Scanner:      meta.Scanner,
		Template:     tmpl,
		CardTemplate: cardTemplate,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		Limit:        limit,
	}

	children, err := scanDir(dir, outputPrefix+name+"/", cfg, resolved)
	if err != nil {
		return config.NodeConfig{}, false, err
	}

	return config.NodeConfig{
		Name:         name,
		Template:     tmpl,
		CardTemplate: cardTemplate,
		OutputPath:   outputPrefix + name + "/index.html",
		DataSource:   config.DataSourceConfig{Type: config.FileType, Path: cfgFile},
		Children:     children,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		Limit:        limit,
		PinnedField:  meta.PinnedField,
		PinnedValue:  meta.PinnedValue,
		Scanner:      meta.Scanner,
	}, true, nil
}

// scanFileNode checks whether name is a supported content file.
// Returns ok=false for node config files, sidecar files, and unsupported extensions.
// If a sidecar {stem}.node.yaml exists alongside the file, its config is applied
// and the sibling directory {stem}/ is scanned for children.
func scanFileNode(parentDir, name, outputPrefix string, cfg *config.SiteConfig) (config.NodeConfig, bool) {
	if name == "node.yaml" {
		return config.NodeConfig{}, false
	}
	// Skip sidecar files ({stem}.node.yaml).
	if strings.HasSuffix(name, ".node.yaml") {
		return config.NodeConfig{}, false
	}
	ext := strings.ToLower(filepath.Ext(name))
	if !contentExts[ext] {
		return config.NodeConfig{}, false
	}

	stem := strings.TrimSuffix(name, filepath.Ext(name))
	outputPath := outputPrefix + stem + "/index.html"
	if outputPrefix == "" && stem == "index" {
		outputPath = "index.html"
	}

	item := config.NodeConfig{
		Name:         stem,
		Template:     cfg.Defaults.Template,
		CardTemplate: cfg.Defaults.CardTemplate,
		OutputPath:   outputPath,
		DataSource:   config.DataSourceConfig{Type: config.FileType, Path: filepath.Join(parentDir, name)},
	}

	// Apply sidecar config if present: {stem}.node.yaml
	sidecarPath := filepath.Join(parentDir, stem+".node.yaml")
	if _, err := os.Stat(sidecarPath); err == nil {
		sidecar := readNodeMeta(sidecarPath)
		if sidecar.Template != "" {
			item.Template = sidecar.Template
		}
		if sidecar.CardTemplate != "" {
			item.CardTemplate = sidecar.CardTemplate
		}
		if sidecar.SortBy != "" {
			item.SortBy = sidecar.SortBy
		}
		if sidecar.SortOrder != "" {
			item.SortOrder = sidecar.SortOrder
		}
		if sidecar.Limit != 0 {
			item.Limit = sidecar.Limit
		}
		if sidecar.PinnedField != "" {
			item.PinnedField = sidecar.PinnedField
		}
		if sidecar.PinnedValue != "" {
			item.PinnedValue = sidecar.PinnedValue
		}
		// Scan the sibling directory for children.
		siblingDir := filepath.Join(parentDir, stem)
		if info, statErr := os.Stat(siblingDir); statErr == nil && info.IsDir() {
			children, err := scanDir(siblingDir, outputPrefix+stem+"/", cfg, sidecar)
			if err == nil {
				item.Children = children
			}
		}
	}

	return item, true
}

// readNodeMeta reads build metadata from a node.yaml file.
func readNodeMeta(path string) nodeMeta {
	data, err := os.ReadFile(path)
	if err != nil {
		return nodeMeta{}
	}
	// Decode into a raw map first so we can read the legacy "type" key.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nodeMeta{}
	}
	var m nodeMeta
	_ = yaml.Unmarshal(data, &m)
	// Fall back to legacy "type" key if "scanner" is not set.
	if m.Scanner == "" {
		if t, ok := raw["type"].(string); ok {
			m.Scanner = t
		}
	}
	return m
}

// readSidecar reads a YAML sidecar file and returns its contents as a map.
// Returns nil if the file does not exist or cannot be parsed.
func readSidecar(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}
