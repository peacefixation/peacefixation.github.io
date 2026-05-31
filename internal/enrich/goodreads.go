package enrich

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// GRCacheEntry holds cached Goodreads data for a single book URL.
type GRCacheEntry struct {
	FetchedAt   time.Time `json:"fetchedAt"`
	Title       string    `json:"title,omitempty"`
	Author      string    `json:"author,omitempty"`
	Cover       string    `json:"cover,omitempty"`
	Description string    `json:"description,omitempty"`
	ISBN        string    `json:"isbn,omitempty"`
	Rating      string    `json:"rating,omitempty"`
}

// GREnricher fetches and caches book metadata from Goodreads pages.
type GREnricher struct {
	cacheFile  string
	cache      map[string]GRCacheEntry
	httpClient *http.Client
}

// NewGoodreads returns a GREnricher that persists its cache to cacheFile.
func NewGoodreads(cacheFile string) *GREnricher {
	return &GREnricher{
		cacheFile:  cacheFile,
		cache:      make(map[string]GRCacheEntry),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// LoadCache reads the cache file into memory. Missing file is not an error.
func (e *GREnricher) LoadCache() error {
	data, err := os.ReadFile(e.cacheFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading Goodreads cache %s: %w", e.cacheFile, err)
	}
	return json.Unmarshal(data, &e.cache)
}

// SaveCache writes the in-memory cache to disk.
func (e *GREnricher) SaveCache() error {
	data, err := json.MarshalIndent(e.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding Goodreads cache: %w", err)
	}
	return os.WriteFile(e.cacheFile, data, 0644)
}

// Enrich returns book fields for url, using the cache unless force is true.
// Returns a map with keys: gr_title, gr_author, gr_cover, gr_description,
// gr_isbn, gr_rating. Only keys with non-empty values are included.
func (e *GREnricher) Enrich(url string, force bool) (map[string]any, error) {
	if !force {
		if entry, ok := e.cache[url]; ok {
			return grEntryToMap(entry), nil
		}
	}

	entry, err := e.fetch(url)
	if err != nil {
		return nil, err
	}

	entry.FetchedAt = time.Now().UTC()
	e.cache[url] = entry
	return grEntryToMap(entry), nil
}

func (e *GREnricher) fetch(url string) (GRCacheEntry, error) {
	// Polite rate limit between network requests.
	time.Sleep(time.Second)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return GRCacheEntry{}, fmt.Errorf("creating request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ssg-goodreads-enricher/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return GRCacheEntry{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GRCacheEntry{}, fmt.Errorf("fetching %s: status %d", url, resp.StatusCode)
	}

	return parseGoodreadsPage(resp)
}

// grBookLD is the schema.org Book JSON-LD type as embedded by Goodreads.
type grBookLD struct {
	Type   string `json:"@type"`
	Name   string `json:"name"`
	Author json.RawMessage `json:"author"`
	Image  string `json:"image"`
	Description string `json:"description"`
	ISBN   string `json:"isbn"`
	AggregateRating struct {
		RatingValue json.RawMessage `json:"ratingValue"`
	} `json:"aggregateRating"`
}

func parseGoodreadsPage(resp *http.Response) (GRCacheEntry, error) {
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return GRCacheEntry{}, fmt.Errorf("parsing HTML: %w", err)
	}

	// Try JSON-LD first — more structured and stable than scraping markup.
	for _, src := range collectJSONLDScripts(doc) {
		var ld grBookLD
		if err := json.Unmarshal([]byte(src), &ld); err != nil {
			continue
		}
		if !strings.EqualFold(ld.Type, "Book") {
			continue
		}
		return GRCacheEntry{
			Title:       ld.Name,
			Author:      extractAuthorName(ld.Author),
			Cover:       ld.Image,
			Description: ld.Description,
			ISBN:        ld.ISBN,
			Rating:      extractRatingValue(ld.AggregateRating.RatingValue),
		}, nil
	}

	// Fallback: OG meta tags (title, description, cover image).
	return extractGROGFallback(doc), nil
}

// collectJSONLDScripts returns the text content of every
// <script type="application/ld+json"> element in doc.
func collectJSONLDScripts(doc *html.Node) []string {
	var out []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			if attrVal(n, "type") == "application/ld+json" && n.FirstChild != nil {
				out = append(out, n.FirstChild.Data)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// extractAuthorName handles author as either a single Person object or an array.
func extractAuthorName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var arr []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr[0].Name
	}
	var obj struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Name
	}
	return ""
}

// extractRatingValue handles ratingValue as either a JSON string or number.
func extractRatingValue(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil && s != "" {
		return s
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return fmt.Sprintf("%.2f", f)
	}
	return ""
}

// extractGROGFallback reads OG meta tags when no JSON-LD Book block is found.
func extractGROGFallback(doc *html.Node) GRCacheEntry {
	var entry GRCacheEntry
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			content := attrVal(n, "content")
			switch strings.ToLower(attrVal(n, "property")) {
			case "og:title":
				if entry.Title == "" {
					entry.Title = content
				}
			case "og:description":
				if entry.Description == "" {
					entry.Description = content
				}
			case "og:image":
				if entry.Cover == "" {
					entry.Cover = content
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return entry
}

func grEntryToMap(e GRCacheEntry) map[string]any {
	m := make(map[string]any, 6)
	if e.Title != "" {
		m["gr_title"] = e.Title
	}
	if e.Author != "" {
		m["gr_author"] = e.Author
	}
	if e.Cover != "" {
		m["gr_cover"] = e.Cover
	}
	if e.Description != "" {
		m["gr_description"] = e.Description
	}
	if e.ISBN != "" {
		m["gr_isbn"] = e.ISBN
	}
	if e.Rating != "" {
		m["gr_rating"] = e.Rating
	}
	return m
}
