package enrich

import "log"

// EnricherType identifies a registered enricher by name.
type EnricherType string

const (
	EnricherTypeOpenGraph      EnricherType = "opengraph"
	EnricherTypeYouTubeChannel EnricherType = "youtube-channel"
)

// Enricher fetches external metadata for a single key and caches it.
type Enricher interface {
	LoadCache() error
	SaveCache() error
	// Enrich returns a map of fields to merge into item data.
	// force bypasses the cache.
	Enrich(key string, force bool) (map[string]any, error)
}

// Registry maps EnricherType values to their Enricher and global force flag.
type Registry struct {
	enrichers map[EnricherType]Enricher
	force     map[EnricherType]bool
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		enrichers: make(map[EnricherType]Enricher),
		force:     make(map[EnricherType]bool),
	}
}

// Register adds an enricher for t with an optional global force flag.
func (r *Registry) Register(t EnricherType, e Enricher, force bool) {
	r.enrichers[t] = e
	r.force[t] = force
}

// Enrich calls the enricher registered for t.
// itemForce is an additional per-item override (enrich_refresh in item data).
// Returns nil, nil when no enricher is registered for t.
func (r *Registry) Enrich(t EnricherType, key string, itemForce bool) (map[string]any, error) {
	e, ok := r.enrichers[t]
	if !ok {
		return nil, nil
	}
	return e.Enrich(key, r.force[t] || itemForce)
}

// LoadAll calls LoadCache on every registered enricher.
func (r *Registry) LoadAll() {
	for name, e := range r.enrichers {
		if err := e.LoadCache(); err != nil {
			log.Printf("warning: loading %s cache: %v", name, err)
		}
	}
}

// SaveAll calls SaveCache on every registered enricher.
func (r *Registry) SaveAll() {
	for name, e := range r.enrichers {
		if err := e.SaveCache(); err != nil {
			log.Printf("warning: saving %s cache: %v", name, err)
		}
	}
}
