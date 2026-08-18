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

// GenerateResult is a completed provider call: the trimmed answer text plus the
// provider-reported token usage and the call's own duration. Usage is zero when
// the provider omits it; Duration measures only the LLM call, not the surrounding
// search/crawl work, so tokens-per-second stays an inference-speed metric.
type GenerateResult struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Duration     time.Duration
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
func (s *LLMService) Generate(ctx context.Context, system string, messages []ChatMessage, opts GenerateOptions) (GenerateResult, error) {
	provider, err := s.providers.Active()
	if err != nil {
		return GenerateResult{}, ErrAIUnavailable
	}
	return s.GenerateWith(ctx, provider, system, messages, opts)
}

// GenerateWith runs the chat against an explicitly selected provider.
//
// Supported provider types:
//   - ollama            POST {base}/api/chat
//   - openai_compatible POST {base}/v1/chat/completions
//   - pollinations      POST {base}/v1/chat/completions  (OpenAI-compatible, Bearer key)
//   - huggingface       POST {base}/chat/completions    (OpenAI-compatible, Bearer key)
func (s *LLMService) GenerateWith(ctx context.Context, provider entities.AIProvider, system string, messages []ChatMessage, opts GenerateOptions) (GenerateResult, error) {
	chat := []ChatMessage{{Role: "system", Content: system}}
	chat = append(chat, messages...)
	var body []byte
	var requestURL string
	var err error
	if provider.ProviderType == "ollama" {
		requestURL = strings.TrimRight(provider.BaseURL, "/") + "/api/chat"
		payload := map[string]any{"model": provider.Model, "messages": chat, "stream": false}
		if opts.Temperature != 0 {
			payload["options"] = map[string]any{"temperature": opts.Temperature}
		}
		body, err = json.Marshal(payload)
	} else {
		suffix, ok := openAIEndpoint(provider.ProviderType)
		if !ok {
			return GenerateResult{}, ErrAIProviderError
		}
		requestURL = strings.TrimRight(provider.BaseURL, "/") + suffix
		payload := map[string]any{"model": provider.Model, "messages": chat}
		if opts.Temperature != 0 {
			payload["temperature"] = opts.Temperature
		}
		if opts.MaxTokens > 0 {
			payload["max_tokens"] = opts.MaxTokens
		}
		body, err = json.Marshal(payload)
	}
	if err != nil {
		return GenerateResult{}, ErrAIProviderError
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return GenerateResult{}, ErrAIUnavailable
	}
	request.Header.Set("Content-Type", "application/json")
	if provider.APIKeyEncrypted != nil && *provider.APIKeyEncrypted != "" {
		key, err := DecryptSecret(*provider.APIKeyEncrypted, s.key)
		if err != nil {
			return GenerateResult{}, ErrAIUnavailable
		}
		request.Header.Set("Authorization", "Bearer "+key)
	}
	started := time.Now()
	response, err := s.client.Do(request)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return GenerateResult{}, ErrAITimeout
		}
		return GenerateResult{}, ErrAIUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		io.Copy(io.Discard, response.Body)
		return GenerateResult{}, fmt.Errorf("%w: status %d", ErrAIProviderError, response.StatusCode)
	}
	duration := time.Since(started)
	var content string
	var inputTokens, outputTokens int
	if provider.ProviderType == "ollama" {
		var payload struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			PromptEvalCount int `json:"prompt_eval_count"`
			EvalCount       int `json:"eval_count"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			return GenerateResult{}, ErrAIProviderError
		}
		content = payload.Message.Content
		inputTokens = payload.PromptEvalCount
		outputTokens = payload.EvalCount
	} else {
		var payload struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			return GenerateResult{}, ErrAIProviderError
		}
		if len(payload.Choices) == 0 {
			return GenerateResult{}, ErrAIProviderError
		}
		content = payload.Choices[0].Message.Content
		inputTokens = payload.Usage.PromptTokens
		outputTokens = payload.Usage.CompletionTokens
	}
	return GenerateResult{Content: strings.TrimSpace(content), InputTokens: inputTokens, OutputTokens: outputTokens, Duration: duration}, nil
}

// openAIEndpoint returns the chat-completions path for every provider type
// that speaks the OpenAI wire format (payload + response shape).
func openAIEndpoint(providerType string) (string, bool) {
	switch providerType {
	case "openai_compatible":
		return "/v1/chat/completions", true
	case "pollinations":
		// gen.pollinations.ai exposes the same OpenAI-compatible chat-completions
		// route as openai_compatible (base https://gen.pollinations.ai); the key
		// is mandatory and sent as a Bearer token.
		return "/v1/chat/completions", true
	case "huggingface":
		return "/chat/completions", true
	default:
		return "", false
	}
}
