package site

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/peacefixation/ssg/internal/config"
)

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".gif": true, ".webp": true, ".avif": true,
}

type photoPlugin struct{}

// ScanChildren scans dir for image files and returns a synthetic ItemConfig
// for each one. Sidecar YAML files (same stem, .yaml extension) are merged
// into the item data when present.
func (photoPlugin) ScanChildren(dir, outputPrefix string, cfg *config.SiteConfig, parent listMeta) ([]config.ItemConfig, error) {
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
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !imageExts[ext] {
			continue
		}
		stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		data := map[string]any{
			"type":     "photo",
			"title":    stem,
			"filename": entry.Name(),
			"src":      "/" + outputPrefix + entry.Name(),
		}
		if sidecar := readSidecar(filepath.Join(dir, stem+".yaml")); sidecar != nil {
			maps.Copy(data, sidecar)
		}
		items = append(items, config.ItemConfig{
			Name:         stem,
			Template:     cfg.Defaults.Page.Template,
			CardTemplate: parent.CardTemplate,
			OutputPath:   outputPrefix + stem + "/index.html",
			DataSource: config.DataSourceConfig{
				Type: config.MapType,
				Data: data,
			},
		})
	}
	return items, nil
}

// CopyAssets copies image files from srcDir to destDir.
func (photoPlugin) CopyAssets(srcDir, destDir string) error {
	return copyFiles(srcDir, destDir, imageExts)
}
