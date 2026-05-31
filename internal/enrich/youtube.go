package enrich

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"time"
)

// RecentVideo holds the ID and title of a single video.
type RecentVideo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Playlist holds the ID and title of a channel playlist.
type Playlist struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// YouTubeCacheEntry holds cached YouTube channel data.
type YouTubeCacheEntry struct {
	FetchedAt       time.Time     `json:"fetchedAt"`
	ChannelTitle    string        `json:"channelTitle,omitempty"`
	Description     string        `json:"description,omitempty"`
	Thumbnail       string        `json:"thumbnail,omitempty"`
	SubscriberCount string        `json:"subscriberCount,omitempty"`
	RecentVideos    []RecentVideo `json:"recentVideos,omitempty"`
	Playlists       []Playlist    `json:"playlists,omitempty"`
}

// YouTubeEnricher fetches and caches YouTube channel metadata via the Data API v3.
type YouTubeEnricher struct {
	cacheFile  string
	apiKey     string
	cache      map[string]YouTubeCacheEntry
	httpClient *http.Client
}

// NewYouTube returns a YouTubeEnricher that persists its cache to cacheFile.
func NewYouTube(cacheFile, apiKey string) *YouTubeEnricher {
	return &YouTubeEnricher{
		cacheFile:  cacheFile,
		apiKey:     apiKey,
		cache:      make(map[string]YouTubeCacheEntry),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// doRequest performs a GET to url, retrying up to 3 times on 429 with
// exponential backoff (1s, 2s). Returns an error when all attempts are exhausted.
func (e *YouTubeEnricher) doRequest(url string) (*http.Response, error) {
	for i, delay := 0, time.Second; i < 3; i, delay = i+1, delay*2 {
		resp, err := e.httpClient.Get(url)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}
		resp.Body.Close()
		if i < 2 {
			log.Printf("youtube: rate limited (429), retrying in %s", delay)
			time.Sleep(delay)
		}
	}
	return nil, fmt.Errorf("rate limited after 3 attempts")
}

// LoadCache reads the cache file into memory. A missing file is not an error.
func (e *YouTubeEnricher) LoadCache() error {
	data, err := os.ReadFile(e.cacheFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading YouTube cache %s: %w", e.cacheFile, err)
	}
	return json.Unmarshal(data, &e.cache)
}

// SaveCache writes the in-memory cache to disk.
func (e *YouTubeEnricher) SaveCache() error {
	data, err := json.MarshalIndent(e.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding YouTube cache: %w", err)
	}
	return os.WriteFile(e.cacheFile, data, 0644)
}

// Enrich returns YouTube channel fields for channelID, using the cache unless
// force is true. Returns a map with yt_-prefixed keys merged into item.Data.
func (e *YouTubeEnricher) Enrich(channelID string, force bool) (map[string]any, error) {
	if !force {
		if entry, ok := e.cache[channelID]; ok {
			return ytEntryToMap(entry), nil
		}
	}

	log.Printf("enriching youtube-channel: %s", channelID)
	entry, err := e.fetch(channelID)
	if err != nil {
		return nil, err
	}

	entry.FetchedAt = time.Now().UTC()
	e.cache[channelID] = entry
	return ytEntryToMap(entry), nil
}

func (e *YouTubeEnricher) fetch(channelID string) (YouTubeCacheEntry, error) {
	channelData, err := e.fetchChannelInfo(channelID)
	if err != nil {
		return YouTubeCacheEntry{}, err
	}

	videos, err := e.fetchRecentVideos(channelID)
	if err != nil {
		return YouTubeCacheEntry{}, err
	}
	channelData.RecentVideos = videos

	playlists, err := e.fetchPlaylists(channelID)
	if err != nil {
		return YouTubeCacheEntry{}, err
	}
	channelData.Playlists = playlists

	return channelData, nil
}

// channelsResponse models the fields we need from the channels.list API response.
type channelsResponse struct {
	Items []struct {
		Snippet struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Thumbnails  struct {
				High   *ytThumbnail `json:"high"`
				Medium *ytThumbnail `json:"medium"`
				Default *ytThumbnail `json:"default"`
			} `json:"thumbnails"`
		} `json:"snippet"`
		Statistics struct {
			SubscriberCount string `json:"subscriberCount"`
		} `json:"statistics"`
	} `json:"items"`
}

type ytThumbnail struct {
	URL string `json:"url"`
}

func (e *YouTubeEnricher) fetchChannelInfo(channelID string) (YouTubeCacheEntry, error) {
	log.Printf("youtube: fetching channel info for %s", channelID)
	url := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/channels?part=snippet,statistics&id=%s&key=%s",
		channelID, e.apiKey,
	)

	resp, err := e.doRequest(url)
	if err != nil {
		return YouTubeCacheEntry{}, fmt.Errorf("fetching channel info for %s: %w", channelID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("youtube: channels API error %d for %s", resp.StatusCode, channelID)
		return YouTubeCacheEntry{}, fmt.Errorf("channels API returned status %d for %s", resp.StatusCode, channelID)
	}

	var result channelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return YouTubeCacheEntry{}, fmt.Errorf("decoding channel response for %s: %w", channelID, err)
	}

	if len(result.Items) == 0 {
		return YouTubeCacheEntry{}, fmt.Errorf("no channel found for ID %s", channelID)
	}

	item := result.Items[0]
	thumbnail := ""
	switch {
	case item.Snippet.Thumbnails.High != nil:
		thumbnail = item.Snippet.Thumbnails.High.URL
	case item.Snippet.Thumbnails.Medium != nil:
		thumbnail = item.Snippet.Thumbnails.Medium.URL
	case item.Snippet.Thumbnails.Default != nil:
		thumbnail = item.Snippet.Thumbnails.Default.URL
	}

	return YouTubeCacheEntry{
		ChannelTitle:    html.UnescapeString(item.Snippet.Title),
		Description:     html.UnescapeString(item.Snippet.Description),
		Thumbnail:       thumbnail,
		SubscriberCount: item.Statistics.SubscriberCount,
	}, nil
}

// searchResponse models the fields we need from the search.list API response.
type searchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			Title string `json:"title"`
		} `json:"snippet"`
	} `json:"items"`
}

func (e *YouTubeEnricher) fetchRecentVideos(channelID string) ([]RecentVideo, error) {
	log.Printf("youtube: fetching recent videos for %s", channelID)
	url := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/search?channelId=%s&part=snippet&order=date&maxResults=5&type=video&key=%s",
		channelID, e.apiKey,
	)

	resp, err := e.doRequest(url)
	if err != nil {
		return nil, fmt.Errorf("fetching recent videos for %s: %w", channelID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("youtube: search API error %d for %s", resp.StatusCode, channelID)
		return nil, fmt.Errorf("search API returned status %d for %s", resp.StatusCode, channelID)
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding search response for %s: %w", channelID, err)
	}

	videos := make([]RecentVideo, 0, len(result.Items))
	for _, item := range result.Items {
		if item.ID.VideoID == "" {
			continue
		}
		videos = append(videos, RecentVideo{ID: item.ID.VideoID, Title: html.UnescapeString(item.Snippet.Title)})
	}
	return videos, nil
}

// playlistsResponse models the fields we need from the playlists.list API response.
type playlistsResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title string `json:"title"`
		} `json:"snippet"`
	} `json:"items"`
}

func (e *YouTubeEnricher) fetchPlaylists(channelID string) ([]Playlist, error) {
	log.Printf("youtube: fetching playlists for %s", channelID)
	url := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/playlists?channelId=%s&part=snippet&maxResults=5&key=%s",
		channelID, e.apiKey,
	)

	resp, err := e.doRequest(url)
	if err != nil {
		return nil, fmt.Errorf("fetching playlists for %s: %w", channelID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("youtube: playlists API error %d for %s", resp.StatusCode, channelID)
		return nil, fmt.Errorf("playlists API returned status %d for %s", resp.StatusCode, channelID)
	}

	var result playlistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding playlists response for %s: %w", channelID, err)
	}

	playlists := make([]Playlist, 0, len(result.Items))
	for _, item := range result.Items {
		if item.ID == "" {
			continue
		}
		playlists = append(playlists, Playlist{ID: item.ID, Title: html.UnescapeString(item.Snippet.Title)})
	}
	return playlists, nil
}

func ytEntryToMap(e YouTubeCacheEntry) map[string]any {
	m := make(map[string]any, 6)
	if e.ChannelTitle != "" {
		m["yt_channel_title"] = e.ChannelTitle
	}
	if e.Description != "" {
		m["yt_description"] = e.Description
	}
	if e.Thumbnail != "" {
		m["yt_thumbnail"] = e.Thumbnail
	}
	if e.SubscriberCount != "" {
		m["yt_subscriber_count"] = e.SubscriberCount
	}
	if len(e.RecentVideos) > 0 {
		m["yt_latest_video_id"] = e.RecentVideos[0].ID
		m["yt_latest_video_title"] = e.RecentVideos[0].Title
		rest := e.RecentVideos[1:]
		if len(rest) > 0 {
			videos := make([]map[string]any, len(rest))
			for i, v := range rest {
				videos[i] = map[string]any{"id": v.ID, "title": v.Title}
			}
			m["yt_recent_videos"] = videos
		}
	}
	if len(e.Playlists) > 0 {
		playlists := make([]map[string]any, len(e.Playlists))
		for i, p := range e.Playlists {
			playlists[i] = map[string]any{"id": p.ID, "title": p.Title}
		}
		m["yt_playlists"] = playlists
	}
	return m
}
