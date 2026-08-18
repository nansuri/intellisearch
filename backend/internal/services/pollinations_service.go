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
	"strconv"
	"strings"
	"time"
)

var (
	// ErrPollinationsUnavailable indicates the Pollinations account API could
	// not be reached or returned an unexpected response.
	ErrPollinationsUnavailable = errors.New("pollinations account request failed")
	// ErrPollinationsUnauthorized indicates the Pollinations API key is
	// missing or invalid (upstream 401).
	ErrPollinationsUnauthorized = errors.New("pollinations api key unauthorized")
	// ErrPollinationsForbidden indicates the Pollinations API key is valid but
	// lacks the account scope required by the endpoint (upstream 403).
	ErrPollinationsForbidden = errors.New("pollinations api key missing account scope")
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

// read performs an authenticated GET and returns the raw response body. A
// 401 maps to ErrPollinationsUnauthorized (invalid key), a 403 to
// ErrPollinationsForbidden (valid key without the account scope), and any
// other non-200 status to ErrPollinationsUnavailable.
func (s *PollinationsService) read(ctx context.Context, baseURL, apiKey, path string) ([]byte, error) {
	requestURL := strings.TrimRight(baseURL, "/") + path
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, ErrPollinationsUnavailable
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Accept", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return nil, ErrPollinationsUnavailable
	}
	defer response.Body.Close()
	switch {
	case response.StatusCode == http.StatusUnauthorized:
		return nil, ErrPollinationsUnauthorized
	case response.StatusCode == http.StatusForbidden:
		return nil, ErrPollinationsForbidden
	case response.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("%w: status %d", ErrPollinationsUnavailable, response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

// doJSON performs an authenticated GET and decodes the JSON response body.
func (s *PollinationsService) doJSON(ctx context.Context, baseURL, apiKey, path string, out any) error {
	raw, err := s.read(ctx, baseURL, apiKey, path)
	if err != nil {
		return err
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return ErrPollinationsUnavailable
		}
	}
	return nil
}

// decodeBalance parses the /account/balance body. The upstream endpoint may
// return a bare JSON number (no format param), a JSON-quoted number string
// (with format=json), or an object with a "balance" field — accept all three.
func decodeBalance(raw []byte) (float64, error) {
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, nil
	}
	var quoted string
	if err := json.Unmarshal(raw, &quoted); err == nil {
		return strconv.ParseFloat(quoted, 64)
	}
	var object struct {
		Balance json.Number `json:"balance"`
	}
	if err := json.Unmarshal(raw, &object); err == nil && object.Balance.String() != "" && object.Balance.String() != "null" {
		if f, err := strconv.ParseFloat(object.Balance.String(), 64); err == nil {
			return f, nil
		}
	}
	return 0, errors.New("unexpected /account/balance response")
}

// Account returns the balance, profile, and key info in one call (three
// parallel GETs to /account/balance, /account/profile, /account/key).
func (s *PollinationsService) Account(ctx context.Context, baseURL, apiKey string) (balance float64, profile *PollinationsProfile, key *PollinationsKeyInfo, err error) {
	raw, err := s.read(ctx, baseURL, apiKey, "/account/balance")
	if err != nil {
		return 0, nil, nil, err
	}
	balance, err = decodeBalance(raw)
	if err != nil {
		return 0, nil, nil, ErrPollinationsUnavailable
	}

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
