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
		_, _ = w.Write([]byte(`{"message":{"content":"  cited answer [1]  "},"prompt_eval_count":120,"eval_count":45}`))
	}))
	defer server.Close()

	service := NewLLMService(nil, "key")
	provider := entities.AIProvider{ProviderType: "ollama", BaseURL: server.URL, Model: "llama3.2"}
	result, err := service.GenerateWith(context.Background(), provider, "system prompt", []ChatMessage{{Role: "user", Content: "hello"}}, GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "cited answer [1]" {
		t.Fatalf("unexpected answer %q", result.Content)
	}
	if result.InputTokens != 120 || result.OutputTokens != 45 {
		t.Fatalf("unexpected token usage: in=%d out=%d", result.InputTokens, result.OutputTokens)
	}
	if result.Duration <= 0 {
		t.Fatal("expected a positive generation duration")
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
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"openai answer"}}],"usage":{"prompt_tokens":200,"completion_tokens":80}}`))
	}))
	defer server.Close()

	sealed, err := EncryptSecret("sk-test-key", []byte("key"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewLLMService(nil, "key")
	provider := entities.AIProvider{ProviderType: "openai_compatible", BaseURL: server.URL, Model: "gpt-4o-mini", APIKeyEncrypted: &sealed}
	result, err := service.GenerateWith(context.Background(), provider, "system", []ChatMessage{{Role: "user", Content: "hi"}}, GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "openai answer" {
		t.Fatalf("unexpected answer %q", result.Content)
	}
	if result.InputTokens != 200 || result.OutputTokens != 80 {
		t.Fatalf("unexpected token usage: in=%d out=%d", result.InputTokens, result.OutputTokens)
	}
	if authHeader != "Bearer sk-test-key" {
		t.Fatalf("unexpected auth header %q", authHeader)
	}
	if received["max_tokens"] != nil {
		t.Fatal("max_tokens should be omitted when unset")
	}
}

func TestGenerateWithPollinations(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"pollinations answer"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer server.Close()

	sealed, err := EncryptSecret("pk_test_pollinations", []byte("key"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewLLMService(nil, "key")
	// gen.pollinations.ai speaks the OpenAI wire format and requires a Bearer key.
	provider := entities.AIProvider{ProviderType: "pollinations", BaseURL: server.URL, Model: "openai", APIKeyEncrypted: &sealed}
	result, err := service.GenerateWith(context.Background(), provider, "system", []ChatMessage{{Role: "user", Content: "hi"}}, GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "pollinations answer" {
		t.Fatalf("unexpected answer %q", result.Content)
	}
	if authHeader != "Bearer pk_test_pollinations" {
		t.Fatalf("unexpected auth header %q", authHeader)
	}
	if received["model"] != "openai" {
		t.Fatalf("unexpected model %#v", received["model"])
	}
}

func TestGenerateWithPollinationsWithoutKey(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	service := NewLLMService(nil, "key")
	provider := entities.AIProvider{ProviderType: "pollinations", BaseURL: server.URL, Model: "openai"}
	if _, err := service.GenerateWith(context.Background(), provider, "system", nil, GenerateOptions{}); err != nil {
		t.Fatal(err)
	}
	// Validation prevents keyless pollinations providers, but if one slips
	// through, no Authorization header is sent (the API rejects with 401).
	if authHeader != "" {
		t.Fatalf("expected no auth header without a key, got %q", authHeader)
	}
}

func TestGenerateWithHuggingFace(t *testing.T) {
	var received map[string]any
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hf answer"}}]}`))
	}))
	defer server.Close()

	sealed, err := EncryptSecret("hf_test_token", []byte("key"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewLLMService(nil, "key")
	provider := entities.AIProvider{ProviderType: "huggingface", BaseURL: server.URL, Model: "Qwen/Qwen3-70B-Instruct", APIKeyEncrypted: &sealed}
	result, err := service.GenerateWith(context.Background(), provider, "system", []ChatMessage{{Role: "user", Content: "hi"}}, GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "hf answer" {
		t.Fatalf("unexpected answer %q", result.Content)
	}
	if authHeader != "Bearer hf_test_token" {
		t.Fatalf("unexpected auth header %q", authHeader)
	}
}

func TestGenerateWithUnknownProviderType(t *testing.T) {
	service := NewLLMService(nil, "key")
	provider := entities.AIProvider{ProviderType: "azure", BaseURL: "https://example.com", Model: "m"}
	if _, err := service.GenerateWith(context.Background(), provider, "s", nil, GenerateOptions{}); !errors.Is(err, ErrAIProviderError) {
		t.Fatalf("expected ErrAIProviderError, got %v", err)
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
