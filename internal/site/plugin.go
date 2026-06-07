package site

import "github.com/peacefixation/ssg/internal/config"

// ChildScanner produces ItemConfigs by scanning a directory in a non-standard
// way. Implement this when a node's children are not content files — e.g. a
// photos node whose children are image files.
type ChildScanner interface {
	ScanChildren(dir, outputPrefix string, cfg *config.SiteConfig, parent nodeMeta) ([]config.NodeConfig, error)
}

// AssetCopier copies static assets from a source directory to the output
// directory. Implement this when a node scanner emits non-HTML files alongside
// its rendered output.
type AssetCopier interface {
	CopyAssets(srcDir, destDir string) error
}

// nodePlugins maps scanner names (the "scanner" field in node.yaml) to their
// plugin implementation. Plugins satisfy ChildScanner and/or AssetCopier.
var nodePlugins = map[string]any{
	"photos": photoPlugin{},
}
