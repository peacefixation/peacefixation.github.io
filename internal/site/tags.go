package site

import (
	"maps"
	"regexp"
	"sort"
	"strings"

	"github.com/peacefixation/ssg/internal/config"
)

// taggedItem pairs a LoadedNode with the ordered ancestor chain that leads to
// it, captured at collection time for use as breadcrumbs on the tag page.
type taggedItem struct {
	item      LoadedNode
	ancestors []map[string]any
}

// collectTags recursively walks the LoadedNode tree and returns a map from
// normalised tag name to the items carrying that tag.
// ancestors tracks the list chain at the current level and should be nil at
// the top level.
func collectTags(items []LoadedNode, ancestors []map[string]any) map[string][]taggedItem {
	result := make(map[string][]taggedItem)
	for _, item := range items {
		if len(item.Children) > 0 {
			// Branch node: extend the ancestor chain and recurse.
			title, _ := item.Data["title"].(string)
			if title == "" {
				title = item.Config.Name
			}
			childAncestors := appendAncestor(ancestors, title, item.Config.OutputPath)
			for tag, tagged := range collectTags(item.Children, childAncestors) {
				result[tag] = append(result[tag], tagged...)
			}
		} else {
			// Leaf node: extract tags from already-loaded data.
			if isDraft(item.Data) {
				continue
			}
			for _, tag := range extractTags(item.Data) {
				result[tag] = append(result[tag], taggedItem{
					item:      item,
					ancestors: ancestors,
				})
			}
		}
	}
	return result
}

// extractTags reads the "tags" field from data and returns normalised tag strings.
func extractTags(data map[string]any) []string {
	raw, ok := data["tags"]
	if !ok {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	var tags []string
	for _, item := range items {
		if s, ok := item.(string); ok {
			s = strings.TrimSpace(strings.ToLower(s))
			if s != "" {
				tags = append(tags, s)
			}
		}
	}
	return tags
}

// styleTemplates maps a style name to its default [template, cardTemplate] pair.
var styleTemplates = map[string][2]string{
	"list":    {"tags.html", "tag-card.html"},
	"cloud":   {"tags-cloud.html", "tag-cloud-card.html"},
	"heatmap": {"tags-heatmap.html", "tag-heatmap-card.html"},
}

// buildTagsNode constructs the virtual tags/ root LoadedNode from the collected
// tag map. The root lists all tags as children; each tag is itself a list whose
// children are copies of the real items carrying that tag, with sourcePath
// injected directly into their data for breadcrumb rendering.
func buildTagsNode(tagMap map[string][]taggedItem, cfg *config.SiteConfig) LoadedNode {
	style := cfg.Tags.Style
	var styleTemplate, styleCardTemplate string
	if d, ok := styleTemplates[style]; ok {
		styleTemplate, styleCardTemplate = d[0], d[1]
	}
	tagsTemplate := first(cfg.Tags.Template, styleTemplate, "tags.html")
	tagCardTemplate := first(cfg.Tags.CardTemplate, styleCardTemplate, "tag-card.html")
	tagTemplate := first(cfg.Tags.TagTemplate, "tag.html")
	itemCardTemplate := first(cfg.Tags.ItemCardTemplate, "tag-item-card.html")

	sortedTags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		sortedTags = append(sortedTags, tag)
	}
	sort.Strings(sortedTags)

	// Compute min/max counts for weight normalisation (used by heatmap style).
	var minCount, maxCount int
	if len(sortedTags) > 0 {
		minCount = len(tagMap[sortedTags[0]])
		maxCount = minCount
		for _, tag := range sortedTags {
			c := len(tagMap[tag])
			if c < minCount {
				minCount = c
			}
			if c > maxCount {
				maxCount = c
			}
		}
	}

	tagChildren := make([]LoadedNode, 0, len(sortedTags))
	for _, tag := range sortedTags {
		items := tagMap[tag]
		slug := tagSlug(tag)

		weight := 1.0
		if maxCount > minCount {
			weight = float64(len(items)-minCount) / float64(maxCount-minCount)
		}

		// Each tagged item becomes a child LoadedNode. sourcePath is injected
		// directly into a cloned data map for breadcrumb rendering on the tag page.
		children := make([]LoadedNode, 0, len(items))
		for _, ti := range items {
			itemData := maps.Clone(ti.item.Data)
			itemData["sourcePath"] = ti.ancestors
			children = append(children, LoadedNode{
				Config:   ti.item.Config,
				Data:     itemData,
				Children: ti.item.Children,
			})
		}

		tagChildren = append(tagChildren, LoadedNode{
			Config: config.NodeConfig{
				Name:         slug,
				Template:     tagTemplate,
				CardTemplate: itemCardTemplate,
				OutputPath:   "tags/" + slug + "/index.html",
				SortBy:       cfg.Defaults.SortBy,
				SortOrder:    cfg.Defaults.SortOrder,
			},
			Data: map[string]any{
				"title":  tag,
				"count":  len(items),
				"weight": weight,
			},
			Children: children,
		})
	}

	return LoadedNode{
		Config: config.NodeConfig{
			Name:               "tags",
			Template:           tagsTemplate,
			CardTemplate:       tagCardTemplate,
			OutputPath:         "tags/index.html",
			SortBy:             "title",
			SortOrder:          "asc",
			ExcludeFromSiteMap: true,
		},
		Data: map[string]any{
			"title": "Tags",
			"style": first(style, "list"),
		},
		Children: tagChildren,
	}
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9-]+`)

// tagSlug converts a tag name to a URL-safe slug.
func tagSlug(tag string) string {
	slug := strings.ToLower(tag)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonAlphaNum.ReplaceAllString(slug, "")
	return strings.Trim(slug, "-")
}
