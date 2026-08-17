package services

import (
	"intellisearch/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type SourceItem struct {
	Position int    `json:"position"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Domain   string `json:"domain"`
	Snippet  string `json:"snippet"`
}

// ImageItem is one SearXNG image result (from the images category). Only the
// URLs are persisted/displayed — the API never fetches the images server-side.
type ImageItem struct {
	Position     int    `json:"position"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Source       string `json:"source"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
}
type SearchService struct {
	baseURL string
	client  *http.Client
}

func NewSearchService(cfg config.Config) *SearchService {
	return &SearchService{baseURL: strings.TrimRight(cfg.SearXNGBaseURL, "/"), client: &http.Client{Timeout: time.Duration(cfg.SearXNGTimeoutMS) * time.Millisecond}}
}
func (s *SearchService) Search(ctx context.Context, query string) ([]SourceItem, error) {
	requestURL := fmt.Sprintf("%s/search?q=%s&format=json", s.baseURL, url.QueryEscape(query))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search response status: %d", response.StatusCode)
	}
	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	items := make([]SourceItem, 0, len(payload.Results))
	for _, result := range payload.Results {
		parsed, err := url.Parse(result.URL)
		if err != nil || parsed.Hostname() == "" {
			continue
		}
		items = append(items, SourceItem{Position: len(items) + 1, Title: result.Title, URL: result.URL, Domain: parsed.Hostname(), Snippet: result.Content})
	}
	return items, nil
}

// SearchImages queries SearXNG's images category and maps the results to
// lightweight image items (thumbnails + origin URLs). Best-effort by design:
// callers treat an error as "no images" rather than failing the ask.
func (s *SearchService) SearchImages(ctx context.Context, query string) ([]ImageItem, error) {
	requestURL := fmt.Sprintf("%s/search?q=%s&categories=images&format=json", s.baseURL, url.QueryEscape(query))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image search response status: %d", response.StatusCode)
	}
	var payload struct {
		Results []struct {
			Title        string `json:"title"`
			URL          string `json:"url"`
			ImgSrc       string `json:"img_src"`
			ThumbnailSrc string `json:"thumbnail_src"`
			Source       string `json:"source"`
			Resolution   string `json:"resolution"`
		} `json:"results"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	items := make([]ImageItem, 0, len(payload.Results))
	for _, result := range payload.Results {
		if result.ImgSrc == "" && result.ThumbnailSrc == "" {
			continue
		}
		width, height := parseResolution(result.Resolution)
		items = append(items, ImageItem{
			Position:     len(items) + 1,
			Title:        result.Title,
			URL:          result.URL,
			ThumbnailURL: firstNonEmpty(result.ThumbnailSrc, result.ImgSrc),
			Source:       result.Source,
			Width:        width,
			Height:       height,
		})
	}
	return items, nil
}

// parseResolution parses SearXNG's "WxH" resolution string into width/height.
func parseResolution(raw string) (int, int) {
	parts := strings.SplitN(strings.TrimSpace(raw), "x", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	width, widthErr := strconv.Atoi(parts[0])
	height, heightErr := strconv.Atoi(parts[1])
	if widthErr != nil || heightErr != nil {
		return 0, 0
	}
	return width, height
}

