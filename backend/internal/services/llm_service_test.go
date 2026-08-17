package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"intellisearch/internal/models/entities"
)

func TestGenerateWithOllama(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"message":{"content":"  cited answer [1]  "}}`))
	}))
	defer server.Close()

	service := NewLLMService(nil, "key")
	provider := entities.AIProvider{ProviderType: "ollama", BaseURL: server.URL, Model: "llama3.2"}
	answer, err := service.GenerateWith(context.Background(), provider, "system prompt", []ChatMessage{{Role: "user", Content: "hello"}}, GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if answer != "cited answer [1]" {
		t.Fatalf("unexpected answer %q", answer)
	}
	if received["model"] != "llama3.2" {
		t.Fatalf("unexpected model %#v", received["model"])
	}
	messages := received["messages"].([]any)
	if len(messages) != 2 || messages[0].(map[string]any)["role"] != "system" {
		t.Fatalf("unexpected messages %#v", messages)
	}
}

func TestGenerateWithOpenAICompatible(t *testing.T) {
	var received map[string]any
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"openai answer"}}]}`))
	}))
	defer server.Close()

	sealed, err := EncryptSecret("sk-test-key", []byte("key"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewLLMService(nil, "key")
	provider := entities.AIProvider{ProviderType: "openai_compatible", BaseURL: server.URL, Model: "gpt-4o-mini", APIKeyEncrypted: &sealed}
	answer, err := service.GenerateWith(context.Background(), provider, "system", []ChatMessage{{Role: "user", Content: "hi"}}, GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if answer != "openai answer" {
		t.Fatalf("unexpected answer %q", answer)
	}
	if authHeader != "Bearer sk-test-key" {
		t.Fatalf("unexpected auth header %q", authHeader)
	}
	if received["max_tokens"] != nil {
		t.Fatal("max_tokens should be omitted when unset")
	}
}

func TestGenerateWithMapsErrors(t *testing.T) {
	service := NewLLMService(nil, "key")
	provider := entities.AIProvider{ProviderType: "ollama", BaseURL: "http://127.0.0.1:1", Model: "m"}
	if _, err := service.GenerateWith(context.Background(), provider, "s", nil, GenerateOptions{}); !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("expected ErrAIUnavailable, got %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	provider.BaseURL = server.URL
	if _, err := service.GenerateWith(context.Background(), provider, "s", nil, GenerateOptions{}); !errors.Is(err, ErrAIProviderError) {
		t.Fatalf("expected ErrAIProviderError, got %v", err)
	}
}
