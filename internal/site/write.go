package site

import (
	"fmt"
	"html/template"
	"maps"
	"os"
	"path/filepath"

	"github.com/peacefixation/ssg/internal/renderer"
)

// writeTree walks the Page tree, merges each page's content data with the
// standard template context, renders the template, and writes the HTML file.
// Returns the total number of pages written.
func writeTree(pages []Page, ctx AssemblyContext, r *renderer.Renderer, outputDir string) (int, error) {
	count := 0
	for _, page := range pages {
		if err := writePage(page, ctx, r, outputDir); err != nil {
			return count, err
		}
		count++
		n, err := writeTree(page.Children, ctx, r, outputDir)
		if err != nil {
			return count + n, err
		}
		count += n
	}
	return count, nil
}

// writePage merges a page's content data with the standard template context
// and renders the HTML file to disk.
func writePage(page Page, ctx AssemblyContext, r *renderer.Renderer, outputDir string) error {
	staticJS := make([]string, len(ctx.Cfg.StaticJS))
	for i, f := range ctx.Cfg.StaticJS {
		staticJS[i] = "/static/" + f
	}

	// Tags items carry sourcePath for breadcrumbs; all other items use ancestors.
	breadcrumbLinks := page.Ancestors
	if sp, ok := page.Data["sourcePath"].([]map[string]any); ok {
		breadcrumbLinks = sp
	}

	title, _ := page.Data["title"].(string)

	cards, err := renderCardSpecs(r, page.CardSpecs)
	if err != nil {
		return fmt.Errorf("rendering cards for %q: %w", page.Config.OutputPath, err)
	}

	renderData := maps.Clone(page.Data)
	renderData["Site"] = ctx.Cfg
	renderData["OutputPath"] = page.Config.OutputPath
	renderData["RootItems"] = ctx.RootNavItems
	renderData["List"] = cards
	renderData["Theme"] = ctx.ThemeData
	renderData["StaticJS"] = staticJS
	renderData["BreadcrumbLinks"] = breadcrumbLinks
	renderData["BreadcrumbCurrent"] = title
	renderData["SiteMap"] = ctx.SiteMap
	renderData["PageTemplate"] = page.Config.Template

	outPath := filepath.Join(outputDir, filepath.FromSlash(page.Config.OutputPath))
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("creating output directory for %s: %w", outPath, err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating output file %s: %w", outPath, err)
	}
	defer f.Close()

	return r.RenderItem(f, page.Config.Template, renderData)
}

// renderCardSpecs resolves a slice of CardSpecs into rendered HTML fragments.
// Nested []CardSpec values under the "List" key are resolved recursively before
// each card is rendered, preserving the bottom-up order required by list cards
// that themselves render children.
func renderCardSpecs(r *renderer.Renderer, specs []CardSpec) ([]template.HTML, error) {
	fragments := make([]template.HTML, 0, len(specs))
	for _, spec := range specs {
		data := maps.Clone(spec.Data)
		if nested, ok := data["List"].([]CardSpec); ok {
			resolved, err := renderCardSpecs(r, nested)
			if err != nil {
				return nil, err
			}
			data["List"] = resolved
		}
		html, err := r.RenderCard(spec.Template, data)
		if err != nil {
			return nil, fmt.Errorf("rendering card with template %q: %w", spec.Template, err)
		}
		fragments = append(fragments, html)
	}
	return fragments, nil
}
