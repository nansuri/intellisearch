package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	ErrTranslateUnavailable = errors.New("translation service unavailable")
	ErrTranslateInvalid     = errors.New("invalid translation request")
)

// maxTranslateLength caps the source text (runes) sent to LibreTranslate.
const maxTranslateLength = 5000

// TranslateLanguage is one LibreTranslate language entry.
type TranslateLanguage struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// TranslateService proxies the LibreTranslate container — the browser never
// talks to it directly (its host/IP is internal-only). The base URL comes
// from configuration, never from user input, so there is no SSRF surface.
type TranslateService struct {
	baseURL string
	client  *http.Client
}

func NewTranslateService(baseURL string) *TranslateService {
	return &TranslateService{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 15 * time.Second}}
}

// Languages returns the supported languages (code + display name). The
// LibreTranslate /languages payload is [{ code, name, targets }]; the targets
// list is dropped — the UI shows all languages in both selects.
func (s *TranslateService) Languages(ctx context.Context) ([]TranslateLanguage, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/languages", nil)
	if err != nil {
		return nil, ErrTranslateUnavailable
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, ErrTranslateUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, ErrTranslateUnavailable
	}
	var payload []struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, ErrTranslateUnavailable
	}
	languages := make([]TranslateLanguage, 0, len(payload))
	for _, entry := range payload {
		if entry.Code == "" {
			continue
		}
		languages = append(languages, TranslateLanguage{Code: entry.Code, Name: entry.Name})
	}
	return languages, nil
}

// Translate proxies a LibreTranslate translate call and returns the
// translated text. source "auto" is passed through (LibreTranslate detects).
func (s *TranslateService) Translate(ctx context.Context, q, source, target, format string) (string, error) {
	q = strings.TrimSpace(q)
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	if q == "" || target == "" || len([]rune(q)) > maxTranslateLength {
		return "", ErrTranslateInvalid
	}
	if source == "" {
		source = "auto"
	}
	if format == "" {
		format = "text"
	}
	body, err := json.Marshal(map[string]string{"q": q, "source": source, "target": target, "format": format})
	if err != nil {
		return "", ErrTranslateInvalid
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/translate", bytes.NewReader(body))
	if err != nil {
		return "", ErrTranslateUnavailable
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return "", ErrTranslateUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", ErrTranslateUnavailable
	}
	var payload struct {
		TranslatedText string `json:"translatedText"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", ErrTranslateUnavailable
	}
	if payload.TranslatedText == "" {
		return "", ErrTranslateUnavailable
	}
	return payload.TranslatedText, nil
}

// Available reports whether a LibreTranslate base URL is configured so the
// router can 503 instead of proxying when the service isn't deployed.
func (s *TranslateService) Available() bool {
	return s.baseURL != ""
}
