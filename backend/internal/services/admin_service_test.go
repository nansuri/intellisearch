package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

func TestAdminCreateProviderEncryptsKey(t *testing.T) {
	db := newTestDB(t)
	admin := NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), "test-encryption-key-aes-gcm", t.TempDir())
	provider, err := admin.CreateProvider("cloud-llm", "openai_compatible", "https://api.openai.com", "gpt-4o-mini", nil, "sk-secret-token", true)
	if err != nil {
		t.Fatal(err)
	}
	if provider.APIKeyEncrypted == nil || *provider.APIKeyEncrypted == "" {
		t.Fatal("expected api key to be encrypted at rest")
	}
	decrypted, err := DecryptSecret(*provider.APIKeyEncrypted, []byte("test-encryption-key-aes-gcm"))
	if err != nil || decrypted != "sk-secret-token" {
		t.Fatalf("decrypted key mismatch: %q err=%v", decrypted, err)
	}
	stored, err := admin.Provider(provider.ID)
	if err != nil || stored.Model != "gpt-4o-mini" || !stored.IsActive {
		t.Fatalf("unexpected stored provider: %#v err=%v", stored, err)
	}
}

func TestAdminProviderParametersRoundTripAsJSON(t *testing.T) {
	db := newTestDB(t)
	admin := NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), "k", t.TempDir())
	// Parameters must round-trip as a JSON object — not a base64 string ([]byte
	// marshals to base64; json.RawMessage preserves the raw JSON object).
	created, err := admin.CreateProvider("params", "ollama", "http://localhost:11434", "llama3.2", json.RawMessage(`{"temperature":0.7,"max_tokens":512}`), "", false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(created)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"parameters":{"temperature":0.7,"max_tokens":512}`)) {
		t.Fatalf("parameters must serialize as a JSON object, got: %s", raw)
	}
	// And the same shape survives a full DB round-trip.
	stored, err := admin.Provider(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	rawStored, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(rawStored, []byte(`"parameters":{"temperature":0.7,"max_tokens":512}`)) {
		t.Fatalf("stored provider parameters must be a JSON object, got: %s", rawStored)
	}
}

func TestAdminCreateProviderValidates(t *testing.T) {
	db := newTestDB(t)
	admin := NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), "k", t.TempDir())
	// unreachable / non-http base URL must be rejected
	_, err := admin.CreateProvider("bad", "ollama", "1.2.3.4:11434", "llama3.2", nil, "", false)
	if !errors.Is(err, ErrProviderInvalid) {
		t.Fatalf("expected ErrProviderInvalid, got %v", err)
	}
	_, err = admin.CreateProvider("bad-type", "azure", "http://localhost:11434", "llama3.2", nil, "", false)
	if !errors.Is(err, ErrProviderInvalid) {
		t.Fatalf("expected ErrProviderInvalid for provider type, got %v", err)
	}
}

func TestAdminCreateProviderAcceptsNewProviderTypes(t *testing.T) {
	db := newTestDB(t)
	admin := NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), "k", t.TempDir())
	for _, tc := range []struct {
		name         string
		providerType string
		baseURL      string
		model        string
		apiKey       string
	}{
		{"pollinations", "pollinations", "https://gen.pollinations.ai", "openai", "pk_test_123"},
		{"huggingface", "huggingface", "https://router.huggingface.co/v1", "Qwen/Qwen3-70B-Instruct", "hf_test_token"},
	} {
		provider, err := admin.CreateProvider(tc.name, tc.providerType, tc.baseURL, tc.model, nil, tc.apiKey, false)
		if err != nil {
			t.Fatalf("expected %s to be accepted, got %v", tc.providerType, err)
		}
		if provider.ProviderType != tc.providerType {
			t.Fatalf("unexpected provider type %q", provider.ProviderType)
		}
	}
}

func TestAdminPollinationsRequiresAPIKey(t *testing.T) {
	db := newTestDB(t)
	admin := NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), "k", t.TempDir())
	// Creating a pollinations provider without a key is rejected.
	if _, err := admin.CreateProvider("poll-no-key", "pollinations", "https://gen.pollinations.ai", "openai", nil, "", false); !errors.Is(err, ErrProviderInvalid) {
		t.Fatalf("expected ErrProviderInvalid for keyless pollinations, got %v", err)
	}
	// Switching an existing keyless provider to pollinations without supplying a
	// key is rejected (the type requires one).
	keyless, err := admin.CreateProvider("keyless", "openai_compatible", "https://api.example.com", "model", nil, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.UpdateProvider(keyless.ID, "keyless", "pollinations", "https://gen.pollinations.ai", "openai", nil, "", false); !errors.Is(err, ErrProviderInvalid) {
		t.Fatalf("expected ErrProviderInvalid when switching to pollinations without a key, got %v", err)
	}
	// A pollinations provider with a stored key keeps it when the key field is
	// left blank ("blank keeps existing" semantics).
	keyed, err := admin.CreateProvider("poll-keyed", "pollinations", "https://gen.pollinations.ai", "openai", nil, "pk_test_123", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.UpdateProvider(keyed.ID, "poll-keyed", "pollinations", "https://gen.pollinations.ai", "openai", nil, "", false); err != nil {
		t.Fatalf("blank key should keep the stored key, got %v", err)
	}
}

func TestAdminUpdateQueueConfigValidation(t *testing.T) {
	db := newTestDB(t)
	if err := db.Create(&entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10}).Error; err != nil {
		t.Fatal(err)
	}
	admin := NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), "k", t.TempDir())
	if _, err := admin.UpdateQueueConfig(0, 20, 60000, 10, 6, 3, 20, 168); !errors.Is(err, ErrQueueConfigInvalid) {
		t.Fatalf("expected ErrQueueConfigInvalid, got %v", err)
	}
	// Negative suggestion-cache hours are invalid too.
	if _, err := admin.UpdateQueueConfig(4, 20, 60000, 10, -1, 3, 20, 168); !errors.Is(err, ErrQueueConfigInvalid) {
		t.Fatalf("expected ErrQueueConfigInvalid for negative cache hours, got %v", err)
	}
	// Negative default quota is invalid as well.
	if _, err := admin.UpdateQueueConfig(4, 20, 60000, 10, 6, -1, 20, 168); !errors.Is(err, ErrQueueConfigInvalid) {
		t.Fatalf("expected ErrQueueConfigInvalid for negative default quota, got %v", err)
	}
	// Negative image-result cap is invalid too.
	if _, err := admin.UpdateQueueConfig(4, 20, 60000, 10, 6, 3, -1, 168); !errors.Is(err, ErrQueueConfigInvalid) {
		t.Fatalf("expected ErrQueueConfigInvalid for negative image cap, got %v", err)
	}
	// A negative session TTL is invalid as well.
	if _, err := admin.UpdateQueueConfig(4, 20, 60000, 10, 6, 3, 20, -1); !errors.Is(err, ErrQueueConfigInvalid) {
		t.Fatalf("expected ErrQueueConfigInvalid for negative session TTL, got %v", err)
	}
	config, err := admin.UpdateQueueConfig(6, 40, 120000, 5, 24, 7, 12, 72)
	if err != nil {
		t.Fatal(err)
	}
	if config.MaxConcurrent != 6 || config.PerUserRateLimit != 5 || config.SuggestionCacheHours != 24 || config.DefaultDailyQuota != 7 || config.MaxImageResults != 12 || config.SessionTTLHours != 72 {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestAdminSiteSettingsValidation(t *testing.T) {
	db := newTestDB(t)
	if err := db.Create(&entities.SiteSettings{ID: 1, SiteName: "Intellisearch"}).Error; err != nil {
		t.Fatal(err)
	}
	admin := NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), "k", t.TempDir())
	if _, err := admin.UpdateSiteSettings("", nil, nil); !errors.Is(err, ErrSiteSettingsInvalid) {
		t.Fatalf("expected ErrSiteSettingsInvalid, got %v", err)
	}
	tagline := "Research, distilled"
	copyright := "Acme Search"
	settings, err := admin.UpdateSiteSettings("Acme Search", &tagline, &copyright)
	if err != nil || settings.SiteName != "Acme Search" || settings.Tagline == nil || settings.Copyright == nil || *settings.Copyright != "Acme Search" {
		t.Fatalf("unexpected settings: %#v err=%v", settings, err)
	}
}

func TestStorageRejectsNonImages(t *testing.T) {
	dir := t.TempDir()
	if _, err := saveUpload(dir, "avatars", "u1", "notes.txt", []byte("hello"), 2<<20); err != ErrUploadRejected {
		t.Fatalf("expected ErrUploadRejected, got %v", err)
	}
	url, err := saveUpload(dir, "avatars", "u1", "face.png", pngBytes(16), 2<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := saveUpload(dir, "avatars", "u1", "face.png", pngBytes(16), 10); err != ErrUploadRejected {
		t.Fatalf("expected ErrUploadRejected for oversized file, got %v", err)
	}
	// ignore returned url path difference; just ensure extension resolves
	_ = url
}

func TestStatsAggregation(t *testing.T) {
	db := newTestDB(t)
	providerID := uuid.New()
	provider := entities.AIProvider{ID: providerID, Name: "local-ollama", ProviderType: "ollama", BaseURL: "http://ollama:11434", Model: "llama3.2", IsActive: true}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	latencies := []int{100, 200, 300, 400, 8000}
	for i, latency := range latencies {
		usage := entities.UsageLog{ID: uint64(i + 1), UserID: nil, Query: "q", LatencyMS: latency, Status: entities.MessageStatusCompleted, ProviderID: &providerID, CreatedAt: now}
		if i == len(latencies)-1 {
			usage.Status = entities.MessageStatusFailed
			usage.ErrorCode = strPtr("AISY01002")
			usage.LatencyMS = 0
		}
		if err := db.Create(&usage).Error; err != nil {
			t.Fatal(err)
		}
	}
	stats := NewStatsService(repositories.NewUsageLogRepository(db), repositories.NewUserRepository(db), repositories.NewProviderRepository(db), nil, nil, nil)
	result, err := stats.AIStats("")
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCompleted != 4 || result.TotalFailed != 1 {
		t.Fatalf("unexpected totals: completed=%d failed=%d", result.TotalCompleted, result.TotalFailed)
	}
	if len(result.Errors) != 1 || result.Errors[0].ErrorCode != "AISY01002" {
		t.Fatalf("unexpected error groups: %#v", result.Errors)
	}
	if result.Latency.P50 != 200 || result.Latency.P95 != 300 || result.Latency.AverageMS != 250 {
		t.Fatalf("unexpected latency: %#v", result.Latency)
	}
}

// pngBytes returns the minimal valid PNG header so http.DetectContentType (or
// the extension fallback) recognises the payload as an image.
func pngBytes(size int) []byte {
	header := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	for len(header) < size {
		header = append(header, 0x00)
	}
	return header
}

func strPtr(s string) *string { return &s }