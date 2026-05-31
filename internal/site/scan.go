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

// listMeta is the in-memory representation of a list.yaml file.
// It carries browser extension metadata (title, fields) plus optional build
// overrides (template, cardTemplate, sortBy, sortOrder, limit).
type listMeta struct {
	Title        string `yaml:"title"`
	Type         string `yaml:"type"` // "photos" triggers image-file scanning
	Template     string `yaml:"template"`
	CardTemplate string `yaml:"cardTemplate"`
	SortBy       string `yaml:"sortBy"`
	SortOrder    string `yaml:"sortOrder"`
	Limit        int    `yaml:"limit"`
}

// scanDir recursively walks dir and returns an ItemConfig for every discovered item.
// If the parent list's type is registered in listPlugins as a ChildScanner, that
// scanner is used instead of the normal content scan.
func scanDir(dir, outputPrefix string, cfg *config.SiteConfig, parent listMeta) ([]config.ItemConfig, error) {
	if p, ok := listPlugins[parent.Type]; ok {
		if scanner, ok := p.(ChildScanner); ok {
			return scanner.ScanChildren(dir, outputPrefix, cfg, parent)
		}
	}
	return scanContentDir(dir, outputPrefix, cfg, parent)
}

// scanContentDir scans a directory for content files and subdirectories:
//   - Files with a supported extension become page items.
//   - Subdirectories containing a list.yaml become directory items whose
//     Children are the result of recursively scanning that subdirectory.
func scanContentDir(dir, outputPrefix string, cfg *config.SiteConfig, parent listMeta) ([]config.ItemConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}

	var items []config.ItemConfig
	for _, entry := range entries {
		if entry.IsDir() {
			item, ok, err := scanDirItem(dir, entry.Name(), outputPrefix, cfg, parent)
			if err != nil {
				return nil, err
			}
			if ok {
				items = append(items, item)
			}
		} else {
			item, ok := scanFileItem(dir, entry.Name(), outputPrefix, cfg)
			if ok {
				items = append(items, item)
			}
		}
	}
	return items, nil
}

// scanDirItem checks whether name is a directory item (contains list.yaml).
// Returns ok=false for directories without list.yaml (they are ignored).
func scanDirItem(parentDir, name, outputPrefix string, cfg *config.SiteConfig, parent listMeta) (config.ItemConfig, bool, error) {
	dir := filepath.Join(parentDir, name)
	listFile := filepath.Join(dir, "list.yaml")
	if _, err := os.Stat(listFile); err != nil {
		return config.ItemConfig{}, false, nil
	}

	meta := readListMeta(listFile)

	// Resolve each setting: list.yaml → parent list → site.yaml defaults.
	tmpl := first(meta.Template, parent.Template, cfg.Defaults.List.Template)
	cardTemplate := first(meta.CardTemplate, parent.CardTemplate, cfg.Defaults.List.CardTemplate)
	sortBy := first(meta.SortBy, parent.SortBy, cfg.Defaults.List.SortBy)
	sortOrder := first(meta.SortOrder, parent.SortOrder, cfg.Defaults.List.SortOrder)
	limit := meta.Limit
	if limit == 0 {
		limit = parent.Limit
	}
	if limit == 0 {
		limit = cfg.Defaults.List.Limit
	}

	resolved := listMeta{
		Type:         meta.Type,
		Template:     tmpl,
		CardTemplate: cardTemplate,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		Limit:        limit,
	}

	children, err := scanDir(dir, outputPrefix+name+"/", cfg, resolved)
	if err != nil {
		return config.ItemConfig{}, false, err
	}

	return config.ItemConfig{
		Name:         name,
		Template:     tmpl,
		CardTemplate: cardTemplate,
		OutputPath:   outputPrefix + name + "/index.html",
		DataSource:   config.DataSourceConfig{Type: config.FileType, Path: listFile},
		Children:     children,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		Limit:        limit,
		ListType:     meta.Type,
	}, true, nil
}

// scanFileItem checks whether name is a supported content file.
// Returns ok=false for files with unsupported extensions and for list.yaml.
func scanFileItem(parentDir, name, outputPrefix string, cfg *config.SiteConfig) (config.ItemConfig, bool) {
	if name == "list.yaml" {
		return config.ItemConfig{}, false
	}
	ext := strings.ToLower(filepath.Ext(name))
	if !contentExts[ext] {
		return config.ItemConfig{}, false
	}

	stem := strings.TrimSuffix(name, filepath.Ext(name))
	outputPath := outputPrefix + stem + "/index.html"
	if outputPrefix == "" && stem == "index" {
		outputPath = "index.html"
	}

	return config.ItemConfig{
		Name:         stem,
		Template:     cfg.Defaults.Page.Template,
		CardTemplate: cfg.Defaults.List.CardTemplate,
		OutputPath:   outputPath,
		DataSource:   config.DataSourceConfig{Type: config.FileType, Path: filepath.Join(parentDir, name)},
	}, true
}

// readListMeta reads build and browser-extension metadata from a list.yaml file.
func readListMeta(path string) listMeta {
	data, err := os.ReadFile(path)
	if err != nil {
		return listMeta{}
	}
	var m listMeta
	_ = yaml.Unmarshal(data, &m)
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
