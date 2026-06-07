package site

import (
	"github.com/peacefixation/ssg/internal/config"
)

// buildSiteMap recursively builds the full site map tree from the loaded item
// tree, skipping the homepage and items with ExcludeFromSiteMap set.
// Titles and icons are read from item.Data, which is already loaded.
func buildSiteMap(items []LoadedNode, itemsDir string) []config.SiteMapNode {
	nodes := make([]config.SiteMapNode, 0, len(items))
	for _, item := range items {
		if item.Config.OutputPath == "index.html" {
			continue
		}
		if item.Config.ExcludeFromSiteMap {
			continue
		}
		if tmpl, _ := item.Data["template"].(string); tmpl == "sitemap.html" {
			continue
		}

		title, _ := item.Data["title"].(string)
		if title == "" {
			title = item.Config.Name
		}
		externalURL, _ := item.Data["url"].(string)
		icon, _ := item.Data["icon"].(string)
		if icon == "" && len(item.Children) > 0 {
			icon = "list"
		}

		nodes = append(nodes, config.SiteMapNode{
			Title:      title,
			OutputPath: item.Config.OutputPath,
			URL:        externalURL,
			Icon:       icon,
			Children:   buildSiteMap(item.Children, itemsDir),
		})
	}
	return nodes
}

// buildNavItems returns lightweight nav records (title, outputPath, count) for
// each item. Data is read from item.Data, which is already loaded.
func buildNavItems(items []LoadedNode) []map[string]any {
	nav := make([]map[string]any, 0, len(items))
	for _, item := range items {
		record := make(map[string]any, len(item.Data)+3)
		for k, v := range item.Data {
			record[k] = v
		}
		record["outputPath"] = item.Config.OutputPath
		record["name"] = item.Config.Name
		record["count"] = len(item.Children)
		nav = append(nav, record)
	}
	return nav
}
