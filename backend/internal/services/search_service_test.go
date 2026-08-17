package services

import (
	"intellisearch/internal/config"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchImagesParsesSearXNGImageResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("categories") != "images" {
			t.Fatalf("expected images category, got %q", r.URL.Query().Get("categories"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"A photo","url":"https://photos.example.com/a","img_src":"https://photos.example.com/a.jpg","thumbnail_src":"https://photos.example.com/a-thumb.jpg","source":"example.com","resolution":"1920x1080"},{"title":"No thumb","url":"https://photos.example.com/b","img_src":"https://photos.example.com/b.jpg","resolution":"bad"}]}`))
	}))
	defer server.Close()
	service := NewSearchService(config.Config{SearXNGBaseURL: server.URL, SearXNGTimeoutMS: 1000})
	images, err := service.SearchImages(context.Background(), "mountains", 0)
	if err != nil || len(images) != 2 {
		t.Fatalf("unexpected images: %#v, %v", images, err)
	}
	// A positive limit caps the results (0 = no limit).
	capped, err := service.SearchImages(context.Background(), "mountains", 1)
	if err != nil || len(capped) != 1 || capped[0].Position != 1 {
		t.Fatalf("expected a single capped image, got %#v (err=%v)", capped, err)
	}
	first := images[0]
	if first.Position != 1 || first.ThumbnailURL != "https://photos.example.com/a-thumb.jpg" || first.Width != 1920 || first.Height != 1080 {
		t.Fatalf("unexpected first image: %#v", first)
	}
	// A missing thumbnail falls back to the full image URL; bad resolution → 0x0.
	if images[1].ThumbnailURL != "https://photos.example.com/b.jpg" || images[1].Width != 0 || images[1].Height != 0 {
		t.Fatalf("unexpected fallback image: %#v", images[1])
	}
}

func TestSearchMapsSearXNGResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "json" {
			t.Fatal("expected json request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"Example","url":"https://example.com/a","content":"A useful source"}]}`))
	}))
	defer server.Close()
	service := NewSearchService(config.Config{SearXNGBaseURL: server.URL, SearXNGTimeoutMS: 1000})
	results, err := service.Search(context.Background(), "hello world")
	if err != nil || len(results) != 1 || results[0].Domain != "example.com" || results[0].Position != 1 {
		t.Fatalf("unexpected result: %#v, %v", results, err)
	}
}
