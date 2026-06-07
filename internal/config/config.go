package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// SiteConfig is the top-level configuration for an SSG site.
type SiteConfig struct {
	Title            string       `mapstructure:"title"`
	BaseURL          string       `mapstructure:"baseURL"`
	CanonicalURL     string       `mapstructure:"canonicalURL"`
	ContentDir       string       `mapstructure:"contentDir"`
	OutputDir        string       `mapstructure:"outputDir"`
	StaticDir        string       `mapstructure:"staticDir"`
	StaticJS         []string     `mapstructure:"staticJS"`
	TemplateDir      string       `mapstructure:"templateDir"`
	ThemesDir        string       `mapstructure:"themesDir"`
	TypesDir         string       `mapstructure:"typesDir"`
	Theme            string       `mapstructure:"theme"`
	Defaults         Defaults     `mapstructure:"defaults"`
	Server           ServerConfig `mapstructure:"server"`
	Drafts           bool         `mapstructure:"-"`
	SiteMap          bool         `mapstructure:"sitemap"`
	OGCacheFile      string       `mapstructure:"ogCacheFile"`
	RefreshOG        bool         `mapstructure:"-"`
	YouTubeCacheFile  string       `mapstructure:"youtubeCacheFile"`
	RefreshYouTube    bool         `mapstructure:"-"`
	YouTubeAPIKey     string       `mapstructure:"-"`
	GoodreadsCacheFile string      `mapstructure:"goodreadsCacheFile"`
	RefreshGoodreads  bool         `mapstructure:"-"`
	Tags             TagsConfig   `mapstructure:"tags"`
}

// SiteMapNode is one node in the site map tree.
// Branch nodes carry Children; leaf nodes have an empty Children slice.
type SiteMapNode struct {
	Title      string
	OutputPath string
	URL        string
	Icon       string
	Children   []SiteMapNode
}

// Defaults holds fallback build config used when a node does not specify its own.
// A single template is used for all nodes; the template self-selects via {{if .List}}.
type Defaults struct {
	Template     string `mapstructure:"template"`
	CardTemplate string `mapstructure:"cardTemplate"`
	SortBy       string `mapstructure:"sortBy"`
	SortOrder    string `mapstructure:"sortOrder"`
	Limit        int    `mapstructure:"limit"`
}

// NodeConfig configures a single node (leaf or branch).
// Branch nodes carry Children discovered by scanning; leaf nodes do not.
type NodeConfig struct {
	Name               string           `mapstructure:"name"`
	Template           string           `mapstructure:"template"`
	CardTemplate       string           `mapstructure:"cardTemplate"`
	OutputPath         string           `mapstructure:"outputPath"`
	DataSource         DataSourceConfig `mapstructure:"dataSource"`
	Children           []NodeConfig
	SortBy             string `mapstructure:"sortBy"`
	SortOrder          string `mapstructure:"sortOrder"`
	Limit              int    `mapstructure:"limit"`
	PinnedField        string `mapstructure:"pinnedField"`
	PinnedValue        string `mapstructure:"pinnedValue"`
	Scanner            string `mapstructure:"-"` // plugin name, e.g. "photos"
	ExcludeFromSiteMap bool   `mapstructure:"-"`
}

// TagsConfig controls the synthesized tags section of the site.
type TagsConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	Style            string `mapstructure:"style"`            // "list" | "cloud" | "heatmap"; sets default template/cardTemplate
	Template         string `mapstructure:"template"`         // explicit override; takes priority over Style
	CardTemplate     string `mapstructure:"cardTemplate"`     // explicit override; takes priority over Style
	TagTemplate      string `mapstructure:"tagTemplate"`      // default: "tag.html"
	ItemCardTemplate string `mapstructure:"itemCardTemplate"` // default: "tag-item-card.html"
}

// DataSourceType identifies the kind of datasource driver to use.
type DataSourceType string

const (
	FileType DataSourceType = "file"
	APIType  DataSourceType = "api"
	MapType  DataSourceType = "map"
)

// DataSourceConfig describes where and how to load data.
type DataSourceConfig struct {
	Type    DataSourceType    `mapstructure:"type"`
	Path    string            `mapstructure:"path"`
	Glob    string            `mapstructure:"glob"`
	Headers map[string]string `mapstructure:"headers"`
	Params  map[string]string `mapstructure:"params"`
	Data    map[string]any    `mapstructure:"-"` // in-memory only; used by MapType
}

// ServerConfig holds development server settings.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// Load reads the site config via Viper (which must already be initialised by
// the root command). If path is non-empty, it overrides the Viper config file.
func Load(path string) (*SiteConfig, error) {
	if path != "" {
		viper.SetConfigFile(path)
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	viper.SetDefault("contentDir", "content")
	viper.SetDefault("outputDir", "public")
	viper.SetDefault("staticDir", "static")
	viper.SetDefault("templateDir", "templates")
	viper.SetDefault("themesDir", "themes")
	viper.SetDefault("typesDir", "types")
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("ogCacheFile", "cache/opengraph.json")
	viper.SetDefault("youtubeCacheFile", "cache/youtube-channel.json")
	viper.SetDefault("goodreadsCacheFile", "cache/goodreads.json")

	var cfg SiteConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return &cfg, nil
}

// Validate checks cfg for obvious errors before a build starts.
func Validate(cfg *SiteConfig) error {
	if cfg.Title == "" {
		return fmt.Errorf("title is required")
	}
	if cfg.Defaults.Template == "" {
		return fmt.Errorf("defaults.template is required")
	}
	return nil
}
