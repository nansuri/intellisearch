package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// translateMock spins up a fake LibreTranslate server (languages + translate)
// and returns a service wired to it.
func translateMock(t *testing.T) (*TranslateService, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/languages":
			_, _ = w.Write([]byte(`[{"code":"en","name":"English","targets":["id","ja"]},{"code":"id","name":"Indonesian"},{"code":"ja","name":"Japanese"}]`))
		case "/translate":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			body, _ := io.ReadAll(r.Body)
			var payload struct {
				Q      string `json:"q"`
				Source string `json:"source"`
				Target string `json:"target"`
			}
			if err := json.Unmarshal(body, &payload); err != nil || payload.Q == "" || payload.Target == "" {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"translatedText":"` + payload.Target + `:` + payload.Q + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return NewTranslateService(server.URL), server
}

func TestTranslateLanguages(t *testing.T) {
	service, _ := translateMock(t)
	languages, err := service.Languages(context.Background())
	if err != nil || len(languages) != 3 {
		t.Fatalf("unexpected languages: %#v err=%v", languages, err)
	}
	if languages[0].Code != "en" || languages[0].Name != "English" {
		t.Fatalf("unexpected first language %+v", languages[0])
	}
}

func TestTranslateRoundTrip(t *testing.T) {
	service, _ := translateMock(t)
	translated, err := service.Translate(context.Background(), "hello", "auto", "ja", "text")
	if err != nil {
		t.Fatal(err)
	}
	if translated != "ja:hello" {
		t.Fatalf("unexpected translation %q", translated)
	}
}

func TestTranslateValidation(t *testing.T) {
	service, _ := translateMock(t)
	// Empty text or missing target is rejected client-side.
	if _, err := service.Translate(context.Background(), "   ", "auto", "ja", "text"); !errors.Is(err, ErrTranslateInvalid) {
		t.Fatalf("expected ErrTranslateInvalid for empty text, got %v", err)
	}
	if _, err := service.Translate(context.Background(), "hello", "auto", "", "text"); !errors.Is(err, ErrTranslateInvalid) {
		t.Fatalf("expected ErrTranslateInvalid for missing target, got %v", err)
	}
}

func TestTranslateUnavailable(t *testing.T) {
	// A base URL pointing at a closed port surfaces ErrTranslateUnavailable.
	service := NewTranslateService("http://127.0.0.1:1")
	if _, err := service.Languages(context.Background()); !errors.Is(err, ErrTranslateUnavailable) {
		t.Fatalf("expected ErrTranslateUnavailable, got %v", err)
	}
	if _, err := service.Translate(context.Background(), "hello", "auto", "ja", "text"); !errors.Is(err, ErrTranslateUnavailable) {
		t.Fatalf("expected ErrTranslateUnavailable, got %v", err)
	}
}

func TestTranslateServiceAvailable(t *testing.T) {
	if NewTranslateService("http://libretranslate:5000").Available() != true {
		t.Fatal("expected configured service to be available")
	}
	if NewTranslateService("").Available() != false {
		t.Fatal("expected empty base URL to be unavailable")
	}
}
