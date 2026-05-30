package site

import (
	"log"
	"maps"
	"os"
	"path/filepath"

	"github.com/peacefixation/ssg/internal/enricher"
	"gopkg.in/yaml.v3"
)

// itemTypeConfig is the in-memory representation of an items/{type}.yaml file.
type itemTypeConfig struct {
	Name     string         `yaml:"name"`
	Defaults map[string]any `yaml:"defaults"`
}

// loadItemTypeDefaults reads items/{typeName}.yaml and returns its defaults map.
// Returns nil if the file does not exist or cannot be parsed.
func loadItemTypeDefaults(itemsDir, typeName string) map[string]any {
	data, err := os.ReadFile(filepath.Join(itemsDir, typeName+".yaml"))
	if err != nil {
		return nil
	}
	var cfg itemTypeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return cfg.Defaults
}

// applyTypeDefaults merges type defaults into data, skipping keys already set.
func applyTypeDefaults(data map[string]any, defaults map[string]any) {
	for k, v := range defaults {
		if _, exists := data[k]; !exists {
			data[k] = v
		}
	}
}

// forceRefreshItem reports whether item data contains og_refresh: true.
func forceRefreshItem(data map[string]any) bool {
	b, ok := data["og_refresh"].(bool)
	return ok && b
}

// forceRefreshYouTube reports whether item data contains yt_refresh: true.
func forceRefreshYouTube(data map[string]any) bool {
	b, ok := data["yt_refresh"].(bool)
	return ok && b
}

// enrichTree walks the LoadedItem tree and enriches each item in place.
func enrichTree(items []LoadedItem, og *enricher.OGEnricher, yt *enricher.YouTubeEnricher, forceOG, forceYT bool) {
	for i := range items {
		enrichItem(&items[i], og, yt, forceOG, forceYT)
		enrichTree(items[i].Children, og, yt, forceOG, forceYT)
	}
}

// enrichItem applies OG or YouTube enrichment to a single LoadedItem in place.
func enrichItem(item *LoadedItem, og *enricher.OGEnricher, yt *enricher.YouTubeEnricher, forceOG, forceYT bool) {
	enrichType, _ := item.Data["enrich"].(string)
	switch enrichType {
	case "opengraph":
		if og == nil {
			return
		}
		if url, _ := item.Data["url"].(string); url != "" {
			force := forceOG || forceRefreshItem(item.Data)
			if ogData, err := og.Enrich(url, force); err != nil {
				log.Printf("warning: OG enrichment failed for %s: %v", url, err)
			} else {
				maps.Copy(item.Data, ogData)
			}
		}
		delete(item.Data, "enrich")
	case "youtube-channel":
		if yt == nil {
			return
		}
		if channelID, _ := item.Data["channelId"].(string); channelID != "" {
			force := forceYT || forceRefreshYouTube(item.Data)
			if ytData, err := yt.Enrich(channelID, force); err != nil {
				log.Printf("warning: YouTube enrichment failed for %s: %v", channelID, err)
			} else {
				maps.Copy(item.Data, ytData)
			}
		}
		delete(item.Data, "enrich")
	}
}
