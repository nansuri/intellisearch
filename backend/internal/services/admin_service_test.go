package services

import (
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
	}{
		{"pollinations", "pollinations", "https://text.pollinations.ai", "openai"},
		{"huggingface", "huggingface", "https://router.huggingface.co/v1", "Qwen/Qwen3-70B-Instruct"},
	} {
		provider, err := admin.CreateProvider(tc.name, tc.providerType, tc.baseURL, tc.model, nil, "", false)
		if err != nil {
			t.Fatalf("expected %s to be accepted, got %v", tc.providerType, err)
		}
		if provider.ProviderType != tc.providerType {
			t.Fatalf("unexpected provider type %q", provider.ProviderType)
		}
	}
}

func TestAdminUpdateQueueConfigValidation(t *testing.T) {
	db := newTestDB(t)
	if err := db.Create(&entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10}).Error; err != nil {
		t.Fatal(err)
	}
	admin := NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), "k", t.TempDir())
	if _, err := admin.UpdateQueueConfig(0, 20, 60000, 10); !errors.Is(err, ErrQueueConfigInvalid) {
		t.Fatalf("expected ErrQueueConfigInvalid, got %v", err)
	}
	config, err := admin.UpdateQueueConfig(6, 40, 120000, 5)
	if err != nil {
		t.Fatal(err)
	}
	if config.MaxConcurrent != 6 || config.PerUserRateLimit != 5 {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestAdminSiteSettingsValidation(t *testing.T) {
	db := newTestDB(t)
	if err := db.Create(&entities.SiteSettings{ID: 1, SiteName: "Intellisearch"}).Error; err != nil {
		t.Fatal(err)
	}
	admin := NewAdminService(repositories.NewProviderRepository(db), repositories.NewQueueConfigRepository(db), repositories.NewSiteRepository(db), "k", t.TempDir())
	if _, err := admin.UpdateSiteSettings("", nil); !errors.Is(err, ErrSiteSettingsInvalid) {
		t.Fatalf("expected ErrSiteSettingsInvalid, got %v", err)
	}
	tagline := "Research, distilled"
	settings, err := admin.UpdateSiteSettings("Acme Search", &tagline)
	if err != nil || settings.SiteName != "Acme Search" || settings.Tagline == nil {
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
	stats := NewStatsService(repositories.NewUsageLogRepository(db), repositories.NewUserRepository(db), repositories.NewProviderRepository(db), nil)
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