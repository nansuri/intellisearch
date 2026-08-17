package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

var (
	// ErrPollinationsUnavailable indicates the Pollinations account API could
	// not be reached or returned an unexpected response.
	ErrPollinationsUnavailable = errors.New("pollinations account request failed")
	// ErrPollinationsUnauthorized indicates the Pollinations API key is
	// missing, invalid, or lacks the required account scope.
	ErrPollinationsUnauthorized = errors.New("pollinations api key unauthorized")
	// ErrPollinationsUploadFailed indicates a Pollinations media upload failed.
	ErrPollinationsUploadFailed = errors.New("pollinations upload failed")
)

// PollinationsProfile is GET /account/profile (PII fields only when the key
// has account:profile scope).
type PollinationsProfile struct {
	GithubUsername            string  `json:"githubUsername"`
	Image                     *string `json:"image"`
	CommunityEndpointsAllowed bool    `json:"communityEndpointsAllowed"`
	Name                      *string `json:"name,omitempty"`
	Email                     *string `json:"email,omitempty"`
}

// PollinationsKeyInfo is GET /account/key — key validity, type, and permissions.
type PollinationsKeyInfo struct {
	Valid       bool     `json:"valid"`
	Type        string   `json:"type"`
	Name        *string  `json:"name"`
	ExpiresAt   *string  `json:"expiresAt"`
	ExpiresIn   *float64 `json:"expiresIn"`
	Permissions struct {
		Models  []string `json:"models"`
		Account []string `json:"account"`
	} `json:"permissions"`
	PollenBudget     *float64 `json:"pollenBudget"`
	RateLimitEnabled bool     `json:"rateLimitEnabled"`
}

// PollinationsUsageRecord is one row of GET /account/usage (per-request history).
type PollinationsUsageRecord struct {
	Timestamp        string   `json:"timestamp"`
	Type             string   `json:"type"`
	Model            *string  `json:"model"`
	APIKeyName       *string  `json:"api_key"`
	MeterSource      *string  `json:"meter_source"`
	InputTextTokens  int64    `json:"input_text_tokens"`
	OutputTextTokens int64    `json:"output_text_tokens"`
	CostUSD          float64  `json:"cost_usd"`
	ResponseTimeMS   *float64 `json:"response_time_ms"`
}

// PollinationsDailyUsage is one row of GET /account/usage/daily.
type PollinationsDailyUsage struct {
	Date        string  `json:"date"`
	APIKeyName  *string `json:"api_key"`
	Model       *string `json:"model"`
	MeterSource *string `json:"meter_source"`
	Requests    int64   `json:"requests"`
	CostUSD     float64 `json:"cost_usd"`
}

// PollinationsModel is one entry of GET /v1/models (OpenAI-compatible list).
type PollinationsModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
}

// PollinationsUploadResult is POST /upload on the media API.
type PollinationsUploadResult struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	ContentType string   `json:"contentType"`
	Size        int64    `json:"size"`
	Tags        []string `json:"tags,omitempty"`
}

// PollinationsService proxies the Pollinations account API (balance, usage,
// models) and media uploads on behalf of the Owner Control Panel. The browser
// never calls Pollinations directly; every request goes through the Go API
// with the provider's decrypted API key.
type PollinationsService struct {
	client    *http.Client
	mediaBase string
}

func NewPollinationsService(mediaBaseURL string) *PollinationsService {
	return &PollinationsService{
		client:    &http.Client{Timeout: 10 * time.Second},
		mediaBase: strings.TrimRight(mediaBaseURL, "/"),
	}
}

// doJSON performs an authenticated GET and decodes a JSON envelope. A 401/403
// maps to ErrPollinationsUnauthorized (the key is invalid or lacks scope);
// anything else non-200 maps to ErrPollinationsUnavailable.
func (s *PollinationsService) doJSON(ctx context.Context, baseURL, apiKey, path string, out any) error {
	requestURL := strings.TrimRight(baseURL, "/") + path
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return ErrPollinationsUnavailable
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Accept", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return ErrPollinationsUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return ErrPollinationsUnauthorized
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrPollinationsUnavailable, response.StatusCode)
	}
	if out != nil {
		if err := json.NewDecoder(response.Body).Decode(out); err != nil {
			return ErrPollinationsUnavailable
		}
	}
	return nil
}

// Account returns the balance, profile, and key info in one call (three
// parallel GETs to /account/balance, /account/profile, /account/key).
func (s *PollinationsService) Account(ctx context.Context, baseURL, apiKey string) (balance float64, profile *PollinationsProfile, key *PollinationsKeyInfo, err error) {
	type balanceResult struct {
		Balance float64 `json:"balance"`
	}
	var balancePayload balanceResult
	if err := s.doJSON(ctx, baseURL, apiKey, "/account/balance", &balancePayload); err != nil {
		return 0, nil, nil, err
	}
	balance = balancePayload.Balance

	var profilePayload PollinationsProfile
	if err := s.doJSON(ctx, baseURL, apiKey, "/account/profile", &profilePayload); err == nil {
		profile = &profilePayload
	}

	var keyPayload PollinationsKeyInfo
	if err := s.doJSON(ctx, baseURL, apiKey, "/account/key", &keyPayload); err == nil {
		key = &keyPayload
	}
	return balance, profile, key, nil
}

// Usage returns per-request usage history for the last `days` (default 30,
// max 90) via GET /account/usage.
func (s *PollinationsService) Usage(ctx context.Context, baseURL, apiKey string, days int) ([]PollinationsUsageRecord, error) {
	if days < 1 {
		days = 30
	}
	if days > 90 {
		days = 90
	}
	var payload struct {
		Usage []PollinationsUsageRecord `json:"usage"`
	}
	if err := s.doJSON(ctx, baseURL, apiKey, fmt.Sprintf("/account/usage?days=%d&format=json", days), &payload); err != nil {
		return nil, err
	}
	return payload.Usage, nil
}

// DailyUsage returns per-day aggregated usage via GET /account/usage/daily.
func (s *PollinationsService) DailyUsage(ctx context.Context, baseURL, apiKey string, days int) ([]PollinationsDailyUsage, error) {
	if days < 1 {
		days = 30
	}
	if days > 90 {
		days = 90
	}
	var payload struct {
		Usage []PollinationsDailyUsage `json:"usage"`
	}
	if err := s.doJSON(ctx, baseURL, apiKey, fmt.Sprintf("/account/usage/daily?days=%d&format=json", days), &payload); err != nil {
		return nil, err
	}
	return payload.Usage, nil
}

// Models lists the models available to the key via GET /v1/models
// (OpenAI-compatible `{ data: [...] }` envelope).
func (s *PollinationsService) Models(ctx context.Context, baseURL, apiKey string) ([]PollinationsModel, error) {
	var payload struct {
		Data []PollinationsModel `json:"data"`
	}
	if err := s.doJSON(ctx, baseURL, apiKey, "/v1/models?community=0", &payload); err != nil {
		return nil, err
	}
	return payload.Data, nil
}

// Upload sends an image to the Pollinations media API (multipart `file` form
// field) and returns the public URL.
func (s *PollinationsService) Upload(ctx context.Context, apiKey, filename, contentType string, data []byte) (PollinationsUploadResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return PollinationsUploadResult{}, ErrPollinationsUploadFailed
	}
	if _, err := part.Write(data); err != nil {
		return PollinationsUploadResult{}, ErrPollinationsUploadFailed
	}
	if err := writer.Close(); err != nil {
		return PollinationsUploadResult{}, ErrPollinationsUploadFailed
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.mediaBase+"/upload", &body)
	if err != nil {
		return PollinationsUploadResult{}, ErrPollinationsUploadFailed
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+apiKey)
	response, err := s.client.Do(request)
	if err != nil {
		return PollinationsUploadResult{}, ErrPollinationsUploadFailed
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return PollinationsUploadResult{}, ErrPollinationsUnauthorized
		}
		return PollinationsUploadResult{}, fmt.Errorf("%w: status %d: %s", ErrPollinationsUploadFailed, response.StatusCode, strings.TrimSpace(string(raw)))
	}
	var result PollinationsUploadResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return PollinationsUploadResult{}, ErrPollinationsUploadFailed
	}
	if result.URL == "" {
		return PollinationsUploadResult{}, ErrPollinationsUploadFailed
	}
	return result, nil
}
