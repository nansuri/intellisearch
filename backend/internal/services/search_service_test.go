package services

import (
	"intellisearch/internal/config"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
