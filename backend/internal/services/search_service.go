package services

import (
	"intellisearch/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
