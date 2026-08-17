package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

func TestCrawlServiceFetchPage(t *testing.T) {
	db := newTestDB(t)
	jobs := repositories.NewCrawlJobRepository(db)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fetch" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "https://example.com/page" {
			t.Fatalf("unexpected url %q", body["url"])
		}
		_, _ = w.Write([]byte(`{"title":"Example Page","text":"Some readable page text"}`))
	}))
	defer server.Close()

	service := NewCrawlService(server.URL, 5000, jobs)
	page, err := service.FetchPage(context.Background(), "https://example.com/page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if page.Title != "Example Page" || page.Text != "Some readable page text" {
		t.Fatalf("unexpected page %#v", page)
	}
	var job entities.CrawlJob
	if err := db.First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != entities.CrawlStatusCompleted {
		t.Fatalf("expected completed job, got %q", job.Status)
	}
}

func TestCrawlServiceRejectsInternalHosts(t *testing.T) {
	db := newTestDB(t)
	jobs := repositories.NewCrawlJobRepository(db)
	service := NewCrawlService("http://crawler:3000", 5000, jobs)
	_, err := service.FetchPage(context.Background(), "http://localhost:8080/admin", nil)
	if !errors.Is(err, ErrURLBlocked) {
		t.Fatalf("expected ErrURLBlocked, got %v", err)
	}
	var job entities.CrawlJob
	if err := db.First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != entities.CrawlStatusBlocked {
		t.Fatalf("expected blocked job, got %q", job.Status)
	}
}

func TestCrawlServiceRejectsInvalidURL(t *testing.T) {
	db := newTestDB(t)
	service := NewCrawlService("http://crawler:3000", 5000, repositories.NewCrawlJobRepository(db))
	if _, err := service.FetchPage(context.Background(), "file:///etc/passwd", nil); !errors.Is(err, ErrURLInvalid) {
		t.Fatalf("expected ErrURLInvalid, got %v", err)
	}
}

func TestCrawlServiceMarksFailedOnCrawlerError(t *testing.T) {
	db := newTestDB(t)
	jobs := repositories.NewCrawlJobRepository(db)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	service := NewCrawlService(server.URL, 5000, jobs)
	if _, err := service.FetchPage(context.Background(), "https://example.com", nil); !errors.Is(err, ErrCrawlFailed) {
		t.Fatalf("expected ErrCrawlFailed, got %v", err)
	}
	var job entities.CrawlJob
	if err := db.First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != entities.CrawlStatusFailed {
		t.Fatalf("expected failed job, got %q", job.Status)
	}
}
