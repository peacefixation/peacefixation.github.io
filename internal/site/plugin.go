package site

import "github.com/peacefixation/ssg/internal/config"

// ChildScanner produces ItemConfigs by scanning a directory in a non-standard
// way. Implement this when a list's children are not content files — e.g. a
// photos list whose children are image files.
type ChildScanner interface {
	ScanChildren(dir, outputPrefix string, cfg *config.SiteConfig, parent listMeta) ([]config.ItemConfig, error)
}

// AssetCopier copies static assets from a source directory to the output
// directory. Implement this when a list type emits non-HTML files alongside
// its rendered output.
type AssetCopier interface {
	CopyAssets(srcDir, destDir string) error
}

// listPlugins maps list type names (the "type" field in list.yaml) to their
// plugin implementation. Plugins satisfy ChildScanner and/or AssetCopier.
var listPlugins = map[string]any{
	"photos": photoPlugin{},
}
