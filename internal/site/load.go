package site

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
)

// LoadedNode is the output of the Load phase: a NodeConfig with its data
// fetched from the datasource, type defaults applied, and any template
// override in the data resolved.
type LoadedNode struct {
	Config   config.NodeConfig
	Data     map[string]any
	Children []LoadedNode
}

// loadTree fetches data for every node in the tree and returns a parallel tree
// of LoadedNodes. For each node: datasource is created, FetchOne is called,
// node-type defaults are applied, the icon default is set, and any template
// override in the data is applied.
func loadTree(items []config.NodeConfig, registry *datasource.Registry, typesDir string, cfg *config.SiteConfig) ([]LoadedNode, error) {
	loaded := make([]LoadedNode, 0, len(items))
	for _, nodeCfg := range items {
		ds, err := registry.New(nodeCfg.DataSource)
		if err != nil {
			return nil, fmt.Errorf("creating datasource for %q: %w", nodeCfg.Name, err)
		}
		data, err := ds.FetchOne()
		if err != nil {
			return nil, fmt.Errorf("fetching data for %q: %w", nodeCfg.Name, err)
		}

		// Apply node-type defaults.
		if typeName, ok := data["type"].(string); ok && typeName != "" {
			if defaults := loadTypeDefaults(typesDir, typeName); defaults != nil {
				applyTypeDefaults(data, defaults)
			}
		}

		// Allow node data to override the template.
		if tmpl, ok := data["template"].(string); ok && tmpl != "" {
			nodeCfg.Template = tmpl
		}

		// Set a default icon if the node does not supply one.
		if _, ok := data["icon"]; !ok {
			if strings.HasSuffix(nodeCfg.DataSource.Path, "node.yaml") {
				data["icon"] = "list"
			} else {
				ext := strings.ToLower(filepath.Ext(nodeCfg.DataSource.Path))
				if ext == ".md" || ext == ".markdown" {
					data["icon"] = "post"
				}
			}
		}

		children, err := loadTree(nodeCfg.Children, registry, typesDir, cfg)
		if err != nil {
			return nil, err
		}

		loaded = append(loaded, LoadedNode{
			Config:   nodeCfg,
			Data:     data,
			Children: children,
		})
	}
	return loaded, nil
}

