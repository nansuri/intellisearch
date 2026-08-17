package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"time"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

var ErrCrawlFailed = errors.New("crawl failed")

// CrawlService fetches page content through the Playwright sidecar. Every URL is
// validated by the SSRF guard before the crawler is called, and each attempt is
// recorded as a CrawlJob (blocked when the guard rejects it).
type CrawlService struct {
	baseURL string
	client  *http.Client
	jobs    *repositories.CrawlJobRepository
}

func NewCrawlService(baseURL string, timeoutMS int, jobs *repositories.CrawlJobRepository) *CrawlService {
	return &CrawlService{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: time.Duration(timeoutMS) * time.Millisecond}, jobs: jobs}
}

type CrawledPage struct {
	Title string
	Text  string
}

func (s *CrawlService) FetchPage(ctx context.Context, rawURL string, userID *uuid.UUID) (CrawledPage, error) {
	parsed, err := ValidateExternalURL(rawURL)
	if err != nil {
		if errors.Is(err, ErrURLBlocked) {
			if jobID, createErr := s.record(userID, rawURL, entities.CrawlStatusBlocked); createErr == nil {
				_ = jobID
			}
			return CrawledPage{}, ErrURLBlocked
		}
		return CrawledPage{}, ErrURLInvalid
	}
	job := entities.CrawlJob{ID: uuid.New(), UserID: userID, URL: parsed.String(), Status: entities.CrawlStatusQueued}
	if err := s.jobs.Create(&job); err != nil {
		return CrawledPage{}, ErrCrawlFailed
	}
	if err := s.jobs.UpdateStatus(job.ID, entities.CrawlStatusRunning); err != nil {
		return CrawledPage{}, ErrCrawlFailed
	}
	body, _ := json.Marshal(map[string]string{"url": parsed.String()})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/fetch", bytes.NewReader(body))
	if err != nil {
		_ = s.jobs.UpdateStatus(job.ID, entities.CrawlStatusFailed)
		return CrawledPage{}, ErrCrawlFailed
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		_ = s.jobs.UpdateStatus(job.ID, entities.CrawlStatusFailed)
		return CrawledPage{}, ErrCrawlFailed
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden {
		_ = s.jobs.UpdateStatus(job.ID, entities.CrawlStatusBlocked)
		return CrawledPage{}, ErrURLBlocked
	}
	if response.StatusCode != http.StatusOK {
		io.Copy(io.Discard, response.Body)
		_ = s.jobs.UpdateStatus(job.ID, entities.CrawlStatusFailed)
		return CrawledPage{}, ErrCrawlFailed
	}
	var payload struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		_ = s.jobs.UpdateStatus(job.ID, entities.CrawlStatusFailed)
		return CrawledPage{}, ErrCrawlFailed
	}
	if err := s.jobs.UpdateStatus(job.ID, entities.CrawlStatusCompleted); err != nil {
		return CrawledPage{}, ErrCrawlFailed
	}
	return CrawledPage{Title: payload.Title, Text: payload.Text}, nil
}

func (s *CrawlService) record(userID *uuid.UUID, rawURL, status string) (uuid.UUID, error) {
	job := entities.CrawlJob{ID: uuid.New(), UserID: userID, URL: rawURL, Status: status}
	if err := s.jobs.Create(&job); err != nil {
		return uuid.Nil, fmt.Errorf("crawl job record: %w", err)
	}
	return job.ID, nil
}
