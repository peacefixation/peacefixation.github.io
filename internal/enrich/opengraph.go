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

// OGCacheEntry holds cached Open Graph data for a single URL.
type OGCacheEntry struct {
	FetchedAt   time.Time `json:"fetchedAt"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Image       string    `json:"image,omitempty"`
	SiteName    string    `json:"siteName,omitempty"`
}

// OGEnricher fetches and caches Open Graph metadata for URLs.
type OGEnricher struct {
	cacheFile  string
	siteURL    string
	cache      map[string]OGCacheEntry
	httpClient *http.Client
}

// New returns an OGEnricher that persists its cache to cacheFile.
// siteURL is sent as the Referer header when probing image URLs, so hotlink
// protection is detected under the same conditions as a real browser load.
func New(cacheFile, siteURL string) *OGEnricher {
	return &OGEnricher{
		cacheFile:  cacheFile,
		siteURL:    siteURL,
		cache:      make(map[string]OGCacheEntry),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// LoadCache reads the cache file into memory. Missing file is not an error.
func (e *OGEnricher) LoadCache() error {
	data, err := os.ReadFile(e.cacheFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading OG cache %s: %w", e.cacheFile, err)
	}
	return json.Unmarshal(data, &e.cache)
}

// SaveCache writes the in-memory cache to disk.
func (e *OGEnricher) SaveCache() error {
	data, err := json.MarshalIndent(e.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding OG cache: %w", err)
	}
	return os.WriteFile(e.cacheFile, data, 0644)
}

// Enrich returns OG fields for url, using the cache unless force is true.
// Returns a map with keys og_title, og_description, og_image, og_site_name.
// Only keys with non-empty values are included.
func (e *OGEnricher) Enrich(url string, force bool) (map[string]any, error) {
	if !force {
		if entry, ok := e.cache[url]; ok {
			return entryToMap(entry), nil
		}
	}

	entry, err := e.fetch(url)
	if err != nil {
		return nil, err
	}

	entry.FetchedAt = time.Now().UTC()
	e.cache[url] = entry
	return entryToMap(entry), nil
}

func (e *OGEnricher) fetch(url string) (OGCacheEntry, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return OGCacheEntry{}, fmt.Errorf("creating request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", "ssg-opengraph-enricher/1.0 (static site builder)")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return OGCacheEntry{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return OGCacheEntry{}, fmt.Errorf("fetching %s: status %d", url, resp.StatusCode)
	}

	entry, err := parseOGTags(resp)
	if err != nil {
		return OGCacheEntry{}, err
	}
	if entry.Image != "" {
		if resolved, ok := e.resolveImageURL(entry.Image); ok {
			entry.Image = resolved
		} else {
			entry.Image = ""
		}
	}
	return entry, nil
}

// resolveImageURL verifies the image URL is reachable and returns the final
// URL after any redirects. Falls back from HEAD to GET for servers that don't
// support HEAD.
func (e *OGEnricher) resolveImageURL(url string) (string, bool) {
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return "", false
		}
		req.Header.Set("User-Agent", "ssg-opengraph-enricher/1.0 (static site builder)")
		if e.siteURL != "" {
			req.Header.Set("Referer", e.siteURL)
		}
		resp, err := e.httpClient.Do(req)
		if err != nil {
			return "", false
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp.Request.URL.String(), true
		}
		// HEAD not supported by this server — retry with GET
		if method == http.MethodHead && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusForbidden) {
			continue
		}
		return "", false
	}
	return "", false
}

func parseOGTags(resp *http.Response) (OGCacheEntry, error) {
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return OGCacheEntry{}, fmt.Errorf("parsing HTML: %w", err)
	}

	var entry OGCacheEntry
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			property := attrVal(n, "property")
			name := attrVal(n, "name")
			content := attrVal(n, "content")

			switch strings.ToLower(property) {
			case "og:title":
				entry.Title = content
			case "og:description":
				entry.Description = content
			case "og:image":
				entry.Image = content
			case "og:site_name":
				entry.SiteName = content
			}

			// Fall back to <meta name="description"> when og:description is absent.
			if strings.ToLower(name) == "description" && entry.Description == "" {
				entry.Description = content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return entry, nil
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func entryToMap(e OGCacheEntry) map[string]any {
	m := make(map[string]any, 4)
	if e.Title != "" {
		m["og_title"] = e.Title
	}
	if e.Description != "" {
		m["og_description"] = e.Description
	}
	if e.Image != "" {
		m["og_image"] = e.Image
	}
	if e.SiteName != "" {
		m["og_site_name"] = e.SiteName
	}
	return m
}
