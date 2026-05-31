package site

import (
	"maps"
	"sort"
	"time"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/theme"
)

// CardSpec describes a card to be rendered in the Write phase.
// Data["List"] may itself be []CardSpec for nested list children.
type CardSpec struct {
	Template string
	Data     map[string]any
}

// Page is the output of the Assemble phase: a fully assembled page with its
// child card specs and ancestor chain recorded for breadcrumbs.
// Page.Data contains content fields only — Site, Theme, RootItems,
// and other template context are not present until the Write phase.
type Page struct {
	Config    config.ItemConfig
	Data      map[string]any
	CardSpecs []CardSpec
	Ancestors []map[string]any
	Children  []Page
}

// AssemblyContext carries the invariants shared across the whole tree during
// assembly. It has no IO methods, no mutable state, and no renderer dependency,
// so assembleTree is a pure function.
type AssemblyContext struct {
	RootNavItems []map[string]any
	ThemeData    theme.Data
	SiteMap      []config.SiteMapNode
	Cfg          *config.SiteConfig
}

// assembleTree assembles a slice of LoadedItems into a Page tree.
// assembleTree is a pure function — no IO, no error path.
func assembleTree(items []LoadedItem, ctx AssemblyContext, ancestors []map[string]any) []Page {
	pages := make([]Page, 0, len(items))
	for _, item := range items {
		page, ok := assembleItem(item, ctx, ancestors)
		if !ok {
			continue // draft
		}
		pages = append(pages, page)
	}
	return pages
}

// assembleItem assembles a single LoadedItem and its descendants into a Page.
// Returns ok=false when the item is a draft and drafts are not enabled.
func assembleItem(item LoadedItem, ctx AssemblyContext, ancestors []map[string]any) (Page, bool) {
	if isDraft(item.Data) && !ctx.Cfg.Drafts {
		return Page{}, false
	}

	title, _ := item.Data["title"].(string)
	childAncestors := appendAncestor(ancestors, title, item.Config.OutputPath)
	childPages := assembleTree(item.Children, ctx, childAncestors)

	// Build card entries from assembled child pages. Each child's CardSpecs field
	// holds its own grandchild specs, which are deferred into List for list children.
	entries := make([]cardEntry, 0, len(childPages))
	for _, cp := range childPages {
		e := cardEntry{
			cfg:  cp.Config,
			data: maps.Clone(cp.Data),
		}
		e.data["outputPath"] = cp.Config.OutputPath
		e.data["count"] = len(cp.Children)
		if len(cp.Children) > 0 {
			e.data["List"] = cp.CardSpecs
		}
		entries = append(entries, e)
	}

	// Sort and apply limit.
	if item.Config.SortBy != "" {
		sortEntries(entries, item.Config.SortBy, item.Config.SortOrder)
	}
	if item.Config.PinnedField != "" {
		entries = pinEntries(entries, item.Config.PinnedField, item.Config.PinnedValue)
	}
	if item.Config.Limit > 0 && len(entries) > item.Config.Limit {
		entries = entries[:item.Config.Limit]
	}

	return Page{
		Config:    item.Config,
		Data:      item.Data,
		CardSpecs: buildCardSpecs(item.Config, entries),
		Ancestors: ancestors,
		Children:  childPages,
	}, true
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

// buildCardSpecs builds a CardSpec for each entry using the parent's card
// template, which individual entries may override via a "cardTemplate" field.
func buildCardSpecs(parent config.ItemConfig, entries []cardEntry) []CardSpec {
	specs := make([]CardSpec, 0, len(entries))
	for _, e := range entries {
		cardTemplate := parent.CardTemplate
		if t, ok := e.data["cardTemplate"].(string); ok && t != "" {
			cardTemplate = t
		}
		specs = append(specs, CardSpec{Template: cardTemplate, Data: e.data})
	}
	return specs
}

// pinEntries moves entries where data[field] == value to the front, preserving relative order within each group.
func pinEntries(entries []cardEntry, field, value string) []cardEntry {
	var pinned []cardEntry
	var rest []cardEntry
	for _, e := range entries {
		if s, _ := e.data[field].(string); s == value {
			pinned = append(pinned, e)
		} else {
			rest = append(rest, e)
		}
	}
	return append(pinned, rest...)
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
