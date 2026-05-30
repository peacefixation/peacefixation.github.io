package site

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
)

// LoadedItem is the output of the Load phase: an ItemConfig with its data
// fetched from the datasource, type defaults applied, and sub-list
// declarations in the data resolved into children.
type LoadedItem struct {
	Config   config.ItemConfig
	Data     map[string]any
	Children []LoadedItem
}

// loadTree fetches data for every item in the tree and returns a parallel tree
// of LoadedItems. For each item: datasource is created, FetchOne is called,
// item-type defaults are applied, the icon default is set, any template
// override in the data is applied, and sub-lists declared via a "lists" field
// are scanned and appended as children.
func loadTree(items []config.ItemConfig, registry *datasource.Registry, itemsDir string, cfg *config.SiteConfig) ([]LoadedItem, error) {
	loaded := make([]LoadedItem, 0, len(items))
	for _, itemCfg := range items {
		ds, err := registry.New(itemCfg.DataSource)
		if err != nil {
			return nil, fmt.Errorf("creating datasource for %q: %w", itemCfg.Name, err)
		}
		data, err := ds.FetchOne()
		if err != nil {
			return nil, fmt.Errorf("fetching data for %q: %w", itemCfg.Name, err)
		}

		// Apply item-type defaults.
		if typeName, ok := data["type"].(string); ok && typeName != "" {
			if defaults := loadItemTypeDefaults(itemsDir, typeName); defaults != nil {
				applyTypeDefaults(data, defaults)
			}
		}

		// Allow item data to override the template.
		if tmpl, ok := data["template"].(string); ok && tmpl != "" {
			itemCfg.Template = tmpl
		}

		// Set a default icon if the item does not supply one.
		if _, ok := data["icon"]; !ok {
			if strings.HasSuffix(itemCfg.DataSource.Path, "list.yaml") {
				data["icon"] = "list"
			} else {
				ext := strings.ToLower(filepath.Ext(itemCfg.DataSource.Path))
				if ext == ".md" || ext == ".markdown" {
					data["icon"] = "post"
				}
			}
		}

		// Expand sub-lists declared in item data (file items with sibling list dirs).
		allChildren := itemCfg.Children
		if rawLists, ok := data["lists"].([]any); ok {
			extra, err := scanSubLists(rawLists, itemCfg, cfg)
			if err != nil {
				return nil, err
			}
			allChildren = append(allChildren, extra...)
			delete(data, "lists")
		}

		children, err := loadTree(allChildren, registry, itemsDir, cfg)
		if err != nil {
			return nil, err
		}

		loaded = append(loaded, LoadedItem{
			Config:   itemCfg,
			Data:     data,
			Children: children,
		})
	}
	return loaded, nil
}

// scanSubLists scans sibling directories named in rawLists and returns them
// as additional ItemConfigs to append to the parent item's children.
func scanSubLists(rawLists []any, itemCfg config.ItemConfig, cfg *config.SiteConfig) ([]config.ItemConfig, error) {
	stem := stemOf(itemCfg.DataSource.Path)
	siblingDir := filepath.Join(filepath.Dir(itemCfg.DataSource.Path), stem)
	outputPrefix := strings.TrimSuffix(itemCfg.OutputPath, "index.html")
	var extra []config.ItemConfig
	for _, raw := range rawLists {
		name, _ := raw.(string)
		if name == "" {
			continue
		}
		sub, ok, err := scanDirItem(siblingDir, name, outputPrefix, cfg, listMeta{
			CardTemplate: itemCfg.CardTemplate,
			SortBy:       itemCfg.SortBy,
			SortOrder:    itemCfg.SortOrder,
			Limit:        itemCfg.Limit,
		})
		if err != nil {
			return nil, fmt.Errorf("scanning sub-list %q of %q: %w", name, itemCfg.Name, err)
		}
		if ok {
			extra = append(extra, sub)
		}
	}
	return extra, nil
}
