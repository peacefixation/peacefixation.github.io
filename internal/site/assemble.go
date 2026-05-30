package site

import (
	"fmt"
	"html/template"
	"maps"
	"sort"
	"time"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/renderer"
	"github.com/peacefixation/ssg/internal/theme"
)

// Page is the output of the Assemble phase: a fully assembled page with its
// child card fragments pre-rendered and its ancestor chain recorded for
// breadcrumbs. Page.Data contains content fields only — Site, Theme, RootItems,
// and other template context are not present until the Write phase.
type Page struct {
	Config    config.ItemConfig
	Data      map[string]any
	Cards     []template.HTML
	Ancestors []map[string]any
	Children  []Page
}

// AssemblyContext carries the invariants shared across the whole tree during
// assembly. It has no IO methods and no mutable state, so assembleTree is a
// pure function given this context.
type AssemblyContext struct {
	Renderer     *renderer.Renderer
	RootNavItems []map[string]any
	ThemeData    theme.Data
	SiteMap      []config.SiteMapNode
	Cfg          *config.SiteConfig
}

// assembleTree assembles a slice of LoadedItems into a Page tree.
// Each item's child cards are rendered from its assembled children, so card
// rendering is bottom-up with no re-fetching and no snapshot needed.
// assembleTree has no IO and can be tested in isolation.
func assembleTree(items []LoadedItem, ctx AssemblyContext, ancestors []map[string]any) ([]Page, error) {
	pages := make([]Page, 0, len(items))
	for _, item := range items {
		page, ok, err := assembleItem(item, ctx, ancestors)
		if err != nil {
			return nil, fmt.Errorf("assembling %q: %w", item.Config.Name, err)
		}
		if !ok {
			continue // draft
		}
		pages = append(pages, page)
	}
	return pages, nil
}

// assembleItem assembles a single LoadedItem and its descendants into a Page.
// Returns ok=false when the item is a draft and drafts are not enabled.
func assembleItem(item LoadedItem, ctx AssemblyContext, ancestors []map[string]any) (Page, bool, error) {
	if isDraft(item.Data) && !ctx.Cfg.Drafts {
		return Page{}, false, nil
	}

	title, _ := item.Data["title"].(string)
	childAncestors := appendAncestor(ancestors, title, item.Config.OutputPath)

	childPages, err := assembleTree(item.Children, ctx, childAncestors)
	if err != nil {
		return Page{}, false, err
	}

	// Build card entries from assembled child pages. Each child's Cards field
	// holds its own grandchild fragments, which are injected as List for list children.
	entries := make([]cardEntry, 0, len(childPages))
	for _, cp := range childPages {
		e := cardEntry{
			cfg:  cp.Config,
			data: maps.Clone(cp.Data),
		}
		e.data["outputPath"] = cp.Config.OutputPath
		e.data["count"] = len(cp.Children)
		if len(cp.Children) > 0 {
			e.data["List"] = cp.Cards
		}
		entries = append(entries, e)
	}

	// Sort and apply limit.
	if item.Config.SortBy != "" {
		sortEntries(entries, item.Config.SortBy, item.Config.SortOrder)
	}
	if item.Config.Limit > 0 && len(entries) > item.Config.Limit {
		entries = entries[:item.Config.Limit]
	}

	cards, err := renderCards(ctx.Renderer, item.Config, entries)
	if err != nil {
		return Page{}, false, err
	}

	return Page{
		Config:    item.Config,
		Data:      item.Data,
		Cards:     cards,
		Ancestors: ancestors,
		Children:  childPages,
	}, true, nil
}

// cardEntry pairs an ItemConfig with the data needed for card rendering.
type cardEntry struct {
	cfg  config.ItemConfig
	data map[string]any
}

// appendAncestor returns a new ancestors slice with {title, outputPath} appended.
func appendAncestor(ancestors []map[string]any, title, outputPath string) []map[string]any {
	next := make([]map[string]any, len(ancestors)+1)
	copy(next, ancestors)
	next[len(ancestors)] = map[string]any{"title": title, "outputPath": outputPath}
	return next
}

// renderCards renders a card fragment for each entry using the parent's card
// template, which individual entries may override via a "cardTemplate" field.
func renderCards(r *renderer.Renderer, parent config.ItemConfig, entries []cardEntry) ([]template.HTML, error) {
	fragments := make([]template.HTML, 0, len(entries))
	for _, e := range entries {
		cardTemplate := parent.CardTemplate
		if t, ok := e.data["cardTemplate"].(string); ok && t != "" {
			cardTemplate = t
		}
		fragment, err := r.RenderCard(cardTemplate, e.data)
		if err != nil {
			return nil, fmt.Errorf("rendering card for %q: %w", e.cfg.Name, err)
		}
		fragments = append(fragments, fragment)
	}
	return fragments, nil
}

// sortEntries sorts entries in-place by the given field.
// time.Time values are compared chronologically; all other values as strings.
func sortEntries(entries []cardEntry, field, order string) {
	sort.SliceStable(entries, func(i, j int) bool {
		ai := entries[i].data[field]
		bi := entries[j].data[field]

		at, aIsTime := ai.(time.Time)
		bt, bIsTime := bi.(time.Time)
		if aIsTime && bIsTime {
			if order == "desc" {
				return at.After(bt)
			}
			return at.Before(bt)
		}

		a, _ := ai.(string)
		b, _ := bi.(string)
		if order == "desc" {
			return a > b
		}
		return a < b
	})
}
