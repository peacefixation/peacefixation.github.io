package site

import (
	"log"
	"maps"
	"os"
	"path/filepath"

	"github.com/peacefixation/ssg/internal/enrich"
	"gopkg.in/yaml.v3"
)

// nodeTypeConfig is the in-memory representation of a types/{type}.yaml file.
type nodeTypeConfig struct {
	Name     string         `yaml:"name"`
	Defaults map[string]any `yaml:"defaults"`
}

// loadTypeDefaults reads types/{typeName}.yaml and returns its defaults map.
// Returns nil if the file does not exist or cannot be parsed.
func loadTypeDefaults(typesDir, typeName string) map[string]any {
	data, err := os.ReadFile(filepath.Join(typesDir, typeName+".yaml"))
	if err != nil {
		return nil
	}
	var cfg nodeTypeConfig
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

// enrichTree walks the LoadedNode tree and enriches each item in place.
func enrichTree(items []LoadedNode, r *enrich.Registry) {
	for i := range items {
		enrichNode(&items[i], r)
		enrichTree(items[i].Children, r)
	}
}

// enrichNode applies enrichment to a single LoadedNode in place.
func enrichNode(item *LoadedNode, r *enrich.Registry) {
	rawType, _ := item.Data["enrich"].(string)
	enrichType := enrich.EnricherType(rawType)
	if enrichType == "" {
		return
	}

	keyField, _ := item.Data["enrich_key"].(string)
	key, _ := item.Data[keyField].(string)
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
	delete(item.Data, "enrich_key")
	delete(item.Data, "enrich_refresh")
}
