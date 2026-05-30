package site

import (
	"log"
	"maps"
	"os"
	"path/filepath"

	"github.com/peacefixation/ssg/internal/enrich"
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

// enrichTree walks the LoadedItem tree and enriches each item in place.
func enrichTree(items []LoadedItem, r *enrich.Registry) {
	for i := range items {
		enrichItem(&items[i], r)
		enrichTree(items[i].Children, r)
	}
}

// enrichItem applies enrichment to a single LoadedItem in place.
func enrichItem(item *LoadedItem, r *enrich.Registry) {
	rawType, _ := item.Data["enrich"].(string)
	enrichType := enrich.EnricherType(rawType)
	if enrichType == "" {
		return
	}

	key := enrichKeyForType(enrichType, item.Data)
	if key == "" {
		return
	}

	itemForce, _ := item.Data["enrich_refresh"].(bool)
	data, err := r.Enrich(enrichType, key, itemForce)
	if err != nil {
		log.Printf("warning: %s enrichment failed for %q: %v", enrichType, key, err)
	} else if data != nil {
		maps.Copy(item.Data, data)
	}
	delete(item.Data, "enrich")
	delete(item.Data, "enrich_refresh")
}

// enrichKeyForType returns the lookup key for the given enricher type from item data.
func enrichKeyForType(enricherType enrich.EnricherType, data map[string]any) string {
	switch enricherType {
	case enrich.EnricherTypeOpenGraph:
		v, _ := data["url"].(string)
		return v
	case enrich.EnricherTypeYouTubeChannel:
		v, _ := data["channelId"].(string)
		return v
	default:
		return ""
	}
}
