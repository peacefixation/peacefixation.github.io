package datasource

import (
	"encoding/json"
	"fmt"
	"html/template"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/renderer"
	"gopkg.in/yaml.v3"
)

// filenameDateRe matches the compact RFC3339 timestamp prefix used in item filenames,
// e.g. "20260418T150405Z-slug.json" or "20260418T150405+1000-slug.json".
var filenameDateRe = regexp.MustCompile(`^(\d{8}T\d{6}(?:Z|[+-]\d{4}))-`)

// FileSource reads data from files on disk.
// It handles type "file" (single file) and type "directory" (glob expansion).
type FileSource struct {
	cfg config.DataSourceConfig
}

// NewFileSource returns a FileSource for the given config.
func NewFileSource(cfg config.DataSourceConfig) (*FileSource, error) {
	return &FileSource{cfg: cfg}, nil
}

// FetchOne reads the single file at cfg.Path and returns its data.
func (f *FileSource) FetchOne() (map[string]any, error) {
	return readFile(f.cfg.Path)
}

// FetchMany expands cfg.Glob (or cfg.Path/*) and returns data for each file.
// node.yaml files are skipped — they are metadata, not content.
func (f *FileSource) FetchMany() ([]map[string]any, error) {
	pattern := f.cfg.Glob
	if pattern == "" {
		pattern = filepath.Join(f.cfg.Path, "*")
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("expanding glob %q: %w", pattern, err)
	}

	results := make([]map[string]any, 0, len(matches))
	for _, match := range matches {
		base := filepath.Base(match)
		if base == "node.yaml" {
			continue
		}
		data, err := readFile(match)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", match, err)
		}
		results = append(results, data)
	}
	return results, nil
}

func readFile(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}

	var data map[string]any
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		data, err = parseMarkdown(raw)
	case ".json":
		data, err = parseJSON(raw)
	case ".yaml", ".yml":
		data, err = parseYAML(raw)
	default:
		data = map[string]any{"content": string(raw)}
	}
	if err != nil {
		return nil, err
	}

	// Inject date from filename if not already present in the item data.
	// Expected filename prefix: 20260418T150405Z-slug.ext or 20260418T150405+1000-slug.ext
	if _, hasDate := data["date"]; !hasDate {
		if t, ok := parseDateFromFilename(filepath.Base(path)); ok {
			data["date"] = t
		}
	}

	return data, nil
}

// parseDateFromFilename extracts a time.Time from a compact RFC3339 filename prefix.
func parseDateFromFilename(name string) (time.Time, bool) {
	m := filenameDateRe.FindStringSubmatch(name)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("20060102T150405Z0700", m[1])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func parseMarkdown(data []byte) (map[string]any, error) {
	frontMatter, bodyHTML, err := renderer.ParseMarkdown(data)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any, len(frontMatter)+1)
	maps.Copy(result, frontMatter)
	result["body"] = template.HTML(bodyHTML) //nolint:gosec // rendered by our own Markdown parser
	return result, nil
}

func parseJSON(data []byte) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return result, nil
}

func parseYAML(data []byte) (map[string]any, error) {
	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	return result, nil
}
