package site

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
	"github.com/peacefixation/ssg/internal/enrich"
	"github.com/peacefixation/ssg/internal/renderer"
	"github.com/peacefixation/ssg/internal/theme"
)

// Build runs the full build pipeline for cfg, writing pages to cfg.OutputDir.
// It returns the total number of pages written across all items.
//
// Pipeline:
//
//	Scan  (IO)  → []ItemConfig
//	Load  (IO)  → []LoadedItem
//	Enrich(IO)  → []LoadedItem (enriched in place)
//	Assemble    → []Page        (pure — no IO)
//	Write (IO)  → HTML files on disk
func Build(cfg *config.SiteConfig, registry *datasource.Registry, clean bool) (int, error) {
	if err := setupOutput(cfg, clean); err != nil {
		return 0, err
	}

	themeData, themeTemplateDir, err := loadTheme(cfg)
	if err != nil {
		return 0, err
	}
	if cfg.Theme != "" {
		if err := theme.CopyAssets(filepath.Join(cfg.ThemesDir, cfg.Theme), cfg.OutputDir); err != nil {
			return 0, fmt.Errorf("copying theme assets: %w", err)
		}
	}
	if cfg.StaticDir != "" {
		if err := copyStaticDir(cfg.StaticDir, cfg.OutputDir); err != nil {
			return 0, fmt.Errorf("copying static assets: %w", err)
		}
	}

	r, err := renderer.New(cfg.TemplateDir, themeTemplateDir)
	if err != nil {
		return 0, fmt.Errorf("initializing renderer: %w", err)
	}

	enrichers, cleanup := initEnrichers(cfg)
	defer cleanup()

	// Phase 1: Scan — walk the content directory and build an ItemConfig tree.
	rootConfigs, err := scanDir(cfg.ContentDir, "", cfg, listMeta{})
	if err != nil {
		return 0, err
	}

	// Phase 2: Load — fetch data for every item; apply type defaults and sub-lists.
	rootItems, err := loadTree(rootConfigs, registry, cfg.ItemsDir, cfg)
	if err != nil {
		return 0, err
	}

	// Phase 3: Enrich — OG and YouTube metadata, in place.
	enrichTree(rootItems, enrichers)

	// Tags are collected from the already-loaded tree — no re-fetching.
	if cfg.Tags.Enabled {
		tagMap := collectTags(rootItems, nil)
		rootItems = append(rootItems, buildTagsItem(tagMap, cfg))
	}

	// Copy photo assets to the output directory before rendering.
	if err := copyPhotosFromTree(rootItems, cfg.OutputDir); err != nil {
		return 0, fmt.Errorf("copying photo assets: %w", err)
	}

	rootNavItems := buildRootNav(rootItems)
	var siteMap []config.SiteMapNode
	if cfg.SiteMap {
		siteMap = buildSiteMap(rootItems, cfg.ItemsDir)
	}

	// Phase 4: Assemble — pure; no IO, no renderer dependency.
	ctx := AssemblyContext{
		RootNavItems: rootNavItems,
		ThemeData:    themeData,
		SiteMap:      siteMap,
		Cfg:          cfg,
	}
	pages := assembleTree(rootItems, ctx, nil)

	// Phase 5: Write — render card specs and page templates, write HTML files.
	return writeTree(pages, ctx, r, cfg.OutputDir)
}

// setupOutput cleans (if requested) and creates the output directory.
func setupOutput(cfg *config.SiteConfig, clean bool) error {
	if clean {
		if err := os.RemoveAll(cfg.OutputDir); err != nil {
			return fmt.Errorf("cleaning output dir: %w", err)
		}
	}
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}
	return nil
}

// buildRootNav returns nav metadata for all root items, filtering out the
// homepage — the site title serves as the home link in the global nav.
func buildRootNav(items []LoadedItem) []map[string]any {
	all := buildNavItems(items)
	nav := make([]map[string]any, 0, len(all))
	for _, item := range all {
		if item["outputPath"] != "index.html" {
			nav = append(nav, item)
		}
	}
	return nav
}

// initEnrichers builds and warms the enricher registry from cfg.
// The returned cleanup func saves all caches and should be deferred by the caller.
func initEnrichers(cfg *config.SiteConfig) (*enrich.Registry, func()) {
	r := enrich.NewRegistry()

	if cfg.OGCacheFile != "" {
		referer := cfg.CanonicalURL
		if referer == "" {
			referer = cfg.BaseURL
		}
		r.Register(enrich.EnricherTypeOpenGraph, enrich.New(cfg.OGCacheFile, referer), cfg.RefreshOG)
	}

	if cfg.YouTubeCacheFile != "" && cfg.YouTubeAPIKey != "" {
		r.Register(enrich.EnricherTypeYouTubeChannel, enrich.NewYouTube(cfg.YouTubeCacheFile, cfg.YouTubeAPIKey), cfg.RefreshYouTube)
	}

	r.LoadAll()

	return r, r.SaveAll
}

// loadTheme reads the theme config (if a theme is set) and returns the theme
// data to inject into templates and the path to the theme's template partials.
func loadTheme(cfg *config.SiteConfig) (theme.Data, string, error) {
	if cfg.Theme == "" {
		return theme.Data{}, "", nil
	}
	themeDir := filepath.Join(cfg.ThemesDir, cfg.Theme)
	themeCfg, err := theme.Load(themeDir)
	if err != nil {
		return theme.Data{}, "", fmt.Errorf("loading theme %q: %w", cfg.Theme, err)
	}
	return theme.BuildData(themeCfg), theme.TemplateDir(themeDir), nil
}

// copyPhotosFromTree walks the LoadedItem tree and copies image files to the
// output directory for any list with ListType "photos".
func copyPhotosFromTree(items []LoadedItem, outputDir string) error {
	for _, item := range items {
		if item.Config.ListType == "photos" {
			srcDir := filepath.Dir(item.Config.DataSource.Path)
			destDir := filepath.Join(outputDir, filepath.Dir(item.Config.OutputPath))
			if err := copyImages(srcDir, destDir); err != nil {
				return fmt.Errorf("copying photos for %q: %w", item.Config.Name, err)
			}
		}
		if err := copyPhotosFromTree(item.Children, outputDir); err != nil {
			return err
		}
	}
	return nil
}
