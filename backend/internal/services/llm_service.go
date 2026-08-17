package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

var (
	ErrAIUnavailable   = errors.New("ai provider unavailable")
	ErrAITimeout       = errors.New("ai provider timeout")
	ErrAIProviderError = errors.New("ai provider returned an error")
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GenerateOptions struct {
	Temperature float64
	MaxTokens   int
}

// LLMService talks to the configured active AI provider (Ollama or an
// OpenAI-compatible API). API keys are decrypted at call time and never logged.
type LLMService struct {
	providers *repositories.ProviderRepository
	client    *http.Client
	key       []byte
}

func NewLLMService(providers *repositories.ProviderRepository, encryptionKey string) *LLMService {
	return &LLMService{providers: providers, client: &http.Client{Timeout: 90 * time.Second}, key: []byte(encryptionKey)}
}

// Generate runs the chat against the currently active provider.
func (s *LLMService) Generate(ctx context.Context, system string, messages []ChatMessage, opts GenerateOptions) (string, error) {
	provider, err := s.providers.Active()
	if err != nil {
		return "", ErrAIUnavailable
	}
	return s.GenerateWith(ctx, provider, system, messages, opts)
}

// GenerateWith runs the chat against an explicitly selected provider.
func (s *LLMService) GenerateWith(ctx context.Context, provider entities.AIProvider, system string, messages []ChatMessage, opts GenerateOptions) (string, error) {
	chat := []ChatMessage{{Role: "system", Content: system}}
	chat = append(chat, messages...)
	var body []byte
	var requestURL string
	var err error
	switch provider.ProviderType {
	case "openai_compatible":
		requestURL = strings.TrimRight(provider.BaseURL, "/") + "/v1/chat/completions"
		payload := map[string]any{"model": provider.Model, "messages": chat}
		if opts.Temperature != 0 {
			payload["temperature"] = opts.Temperature
		}
		if opts.MaxTokens > 0 {
			payload["max_tokens"] = opts.MaxTokens
		}
		body, err = json.Marshal(payload)
	case "ollama":
		requestURL = strings.TrimRight(provider.BaseURL, "/") + "/api/chat"
		payload := map[string]any{"model": provider.Model, "messages": chat, "stream": false}
		if opts.Temperature != 0 {
			payload["options"] = map[string]any{"temperature": opts.Temperature}
		}
		body, err = json.Marshal(payload)
	default:
		return "", ErrAIProviderError
	}
	if err != nil {
		return "", ErrAIProviderError
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return "", ErrAIUnavailable
	}
	request.Header.Set("Content-Type", "application/json")
	if provider.APIKeyEncrypted != nil && *provider.APIKeyEncrypted != "" {
		key, err := DecryptSecret(*provider.APIKeyEncrypted, s.key)
		if err != nil {
			return "", ErrAIUnavailable
		}
		request.Header.Set("Authorization", "Bearer "+key)
	}
	response, err := s.client.Do(request)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", ErrAITimeout
		}
		return "", ErrAIUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		io.Copy(io.Discard, response.Body)
		return "", fmt.Errorf("%w: status %d", ErrAIProviderError, response.StatusCode)
	}
	var content string
	if provider.ProviderType == "openai_compatible" {
		var payload struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			return "", ErrAIProviderError
		}
		if len(payload.Choices) == 0 {
			return "", ErrAIProviderError
		}
		content = payload.Choices[0].Message.Content
	} else {
		var payload struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			return "", ErrAIProviderError
		}
		content = payload.Message.Content
	}
	return strings.TrimSpace(content), nil
}
