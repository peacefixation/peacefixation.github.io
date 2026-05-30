package site

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

// writeTree walks the Page tree, merges each page's content data with the
// standard template context, renders the template, and writes the HTML file.
// Returns the total number of pages written.
func writeTree(pages []Page, ctx AssemblyContext, outputDir string) (int, error) {
	count := 0
	for _, page := range pages {
		if err := writePage(page, ctx, outputDir); err != nil {
			return count, err
		}
		count++
		n, err := writeTree(page.Children, ctx, outputDir)
		if err != nil {
			return count + n, err
		}
		count += n
	}
	return count, nil
}

// writePage merges a page's content data with the standard template context
// and renders the HTML file to disk.
func writePage(page Page, ctx AssemblyContext, outputDir string) error {
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

	renderData := maps.Clone(page.Data)
	renderData["Site"] = ctx.Cfg
	renderData["OutputPath"] = page.Config.OutputPath
	renderData["RootItems"] = ctx.RootNavItems
	renderData["List"] = page.Cards
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

	return ctx.Renderer.RenderItem(f, page.Config.Template, renderData)
}
