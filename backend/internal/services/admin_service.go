package services

import (
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrProviderInvalid   = errors.New("invalid provider configuration")
	ErrProviderNotFound  = errors.New("provider not found")
	ErrQueueConfigInvalid = errors.New("invalid queue configuration")
	ErrSiteSettingsInvalid = errors.New("invalid site settings")
)

// AdminService owns the control-panel write operations that touch domain
// config: AI providers, queue knobs, and branding site settings.
type AdminService struct {
	providers   *repositories.ProviderRepository
	queueConfig *repositories.QueueConfigRepository
	site        *repositories.SiteRepository
	key         []byte
	uploads     string
}

func NewAdminService(providers *repositories.ProviderRepository, queueConfig *repositories.QueueConfigRepository, site *repositories.SiteRepository, encryptionKey string, uploads string) *AdminService {
	return &AdminService{providers: providers, queueConfig: queueConfig, site: site, key: []byte(encryptionKey), uploads: uploads}
}

// Providers lists all AI providers.
func (s *AdminService) Providers() ([]entities.AIProvider, error) { return s.providers.List() }

// Provider returns one provider by id.
func (s *AdminService) Provider(id uuid.UUID) (entities.AIProvider, error) {
	return s.providers.ByID(id)
}

// CreateProvider adds a provider; a supplied API key is encrypted at rest and
// a new active provider deactivates any other active row.
func (s *AdminService) CreateProvider(name, providerType, baseURL, model string, parameters json.RawMessage, apiKey string, isActive bool) (entities.AIProvider, error) {
	if !validProvider(name, providerType, baseURL, model) {
		return entities.AIProvider{}, ErrProviderInvalid
	}
	encrypted, err := optionalEncrypt(apiKey, s.key)
	if err != nil {
		return entities.AIProvider{}, ErrProviderInvalid
	}
	// gen.pollinations.ai requires a Bearer key for every request.
	if providerType == "pollinations" && encrypted == nil {
		return entities.AIProvider{}, ErrProviderInvalid
	}
	provider := entities.AIProvider{ID: uuid.New(), Name: strings.TrimSpace(name), ProviderType: providerType, BaseURL: strings.TrimRight(baseURL, "/"), Model: strings.TrimSpace(model), Parameters: normalizeParameters(parameters), APIKeyEncrypted: encrypted, IsActive: isActive, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.providers.Create(&provider); err != nil {
		return entities.AIProvider{}, err
	}
	if isActive {
		if err := s.providers.SetActive(provider.ID); err != nil {
			return entities.AIProvider{}, err
		}
	}
	return provider, nil
}

// UpdateProvider changes a provider's fields. Leaving apiKey empty keeps the
// existing encrypted key; setting isActive promotes it to the sole active one.
func (s *AdminService) UpdateProvider(id uuid.UUID, name, providerType, baseURL, model string, parameters json.RawMessage, apiKey string, isActive bool) (entities.AIProvider, error) {
	provider, err := s.providers.ByID(id)
	if err != nil {
		return entities.AIProvider{}, ErrProviderNotFound
	}
	if name != "" && !validProvider(name, providerType, baseURL, model) {
		return entities.AIProvider{}, ErrProviderInvalid
	}
	if name != "" {
		provider.Name = strings.TrimSpace(name)
	}
	if providerType != "" {
		provider.ProviderType = providerType
	}
	if baseURL != "" {
		provider.BaseURL = strings.TrimRight(baseURL, "/")
	}
	if model != "" {
		provider.Model = strings.TrimSpace(model)
	}
	if len(parameters) > 0 {
		provider.Parameters = normalizeParameters(parameters)
	}
	if apiKey != "" {
		encrypted, err := optionalEncrypt(apiKey, s.key)
		if err != nil {
			return entities.AIProvider{}, ErrProviderInvalid
		}
		provider.APIKeyEncrypted = encrypted
	}
	// gen.pollinations.ai requires a Bearer key; a provider of that type must
	// end up with one (either supplied now or already stored).
	resolvedType := providerType
	if resolvedType == "" {
		resolvedType = provider.ProviderType
	}
	if resolvedType == "pollinations" && provider.APIKeyEncrypted == nil {
		return entities.AIProvider{}, ErrProviderInvalid
	}
	if isActive {
		if err := s.providers.SetActive(id); err != nil {
			return entities.AIProvider{}, err
		}
		provider.IsActive = true
	}
	provider.UpdatedAt = time.Now().UTC()
	if err := s.providers.Update(&provider); err != nil {
		return entities.AIProvider{}, err
	}
	return provider, nil
}

// DeleteProvider removes a provider by id.
func (s *AdminService) DeleteProvider(id uuid.UUID) error {
	return s.providers.Delete(id)
}

// QueueConfig returns the singleton queue configuration.
func (s *AdminService) QueueConfig() (entities.AIQueueConfig, error) { return s.queueConfig.Get() }

// UpdateQueueConfig validates and persists new concurrency/queue knobs. The AI
// handler picks them up within its cache TTL, so changes apply without restart.
// suggestionCacheHours tunes how long the AI-composed history suggestions are
// reused per user before being recomposed (0 = always compose fresh);
// defaultDailyQuota is the quota applied to accounts registered afterwards
// (0 = unlimited); maxImageResults caps image results per search (0 = unlimited).
func (s *AdminService) UpdateQueueConfig(maxConcurrent, maxQueueSize, requestTimeoutMS, perUserRateLimit, suggestionCacheHours, defaultDailyQuota, maxImageResults int) (entities.AIQueueConfig, error) {
	config, err := s.queueConfig.Get()
	if err != nil {
		return config, err
	}
	if maxConcurrent < 1 || maxQueueSize < 1 || requestTimeoutMS < 100 || perUserRateLimit < 0 || suggestionCacheHours < 0 || defaultDailyQuota < 0 || maxImageResults < 0 {
		return config, ErrQueueConfigInvalid
	}
	config.MaxConcurrent = maxConcurrent
	config.MaxQueueSize = maxQueueSize
	config.RequestTimeoutMS = requestTimeoutMS
	config.PerUserRateLimit = perUserRateLimit
	config.SuggestionCacheHours = suggestionCacheHours
	config.DefaultDailyQuota = defaultDailyQuota
	config.MaxImageResults = maxImageResults
	config.UpdatedAt = time.Now().UTC()
	if err := s.queueConfig.Update(config); err != nil {
		return config, err
	}
	return config, nil
}

// SiteSettings returns the persisted branding row.
func (s *AdminService) SiteSettings() (entities.SiteSettings, error) { return s.site.Get() }

// UpdateSiteSettings saves the site name and tagline.
func (s *AdminService) UpdateSiteSettings(name string, tagline *string) (entities.SiteSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 60 {
		return entities.SiteSettings{}, ErrSiteSettingsInvalid
	}
	settings, err := s.site.Get()
	if err != nil {
		return settings, err
	}
	settings.SiteName = name
	settings.Tagline = tagline
	settings.UpdatedAt = time.Now().UTC()
	if err := s.site.Update(settings); err != nil {
		return settings, err
	}
	return settings, nil
}

// Logo replaces the site logo image and returns the public URL.
func (s *AdminService) Logo(filename string, data []byte) (string, error) {
	url, err := saveUpload(s.uploads, "branding", "logo", filename, data, 2<<20)
	if err != nil {
		if err == ErrUploadRejected {
			return "", ErrUploadRejected
		}
		return "", err
	}
	settings, err := s.site.Get()
	if err != nil {
		return "", err
	}
	settings.LogoURL = &url
	settings.UpdatedAt = time.Now().UTC()
	if err := s.site.Update(settings); err != nil {
		return "", err
	}
	return url, nil
}

// RemoveLogo clears the site logo so the public pages fall back to the default
// initials mark. The previously stored image file is removed best-effort.
func (s *AdminService) RemoveLogo() error {
	settings, err := s.site.Get()
	if err != nil {
		return err
	}
	if settings.LogoURL != nil && s.uploads != "" {
		path := filepath.Join(s.uploads, filepath.Base(strings.TrimPrefix(*settings.LogoURL, "/uploads/")))
		_ = os.Remove(path)
	}
	settings.LogoURL = nil
	settings.UpdatedAt = time.Now().UTC()
	return s.site.Update(settings)
}

// Favicon replaces the site favicon image and returns the public URL.
func (s *AdminService) Favicon(filename string, data []byte) (string, error) {
	url, err := saveUpload(s.uploads, "branding", "favicon", filename, data, 2<<20)
	if err != nil {
		if err == ErrUploadRejected {
			return "", ErrUploadRejected
		}
		return "", err
	}
	settings, err := s.site.Get()
	if err != nil {
		return "", err
	}
	settings.FaviconURL = &url
	settings.UpdatedAt = time.Now().UTC()
	if err := s.site.Update(settings); err != nil {
		return "", err
	}
	return url, nil
}

// RemoveFavicon clears the site favicon so browsers fall back to the bundled
// default mark. The previously stored image file is removed best-effort.
func (s *AdminService) RemoveFavicon() error {
	settings, err := s.site.Get()
	if err != nil {
		return err
	}
	if settings.FaviconURL != nil && s.uploads != "" {
		path := filepath.Join(s.uploads, filepath.Base(strings.TrimPrefix(*settings.FaviconURL, "/uploads/")))
		_ = os.Remove(path)
	}
	settings.FaviconURL = nil
	settings.UpdatedAt = time.Now().UTC()
	return s.site.Update(settings)
}

func validProvider(name, providerType, baseURL, model string) bool {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" {
		return false
	}
	return strings.TrimSpace(name) != "" &&
		isSupportedProviderType(providerType) &&
		strings.TrimSpace(model) != ""
}

// isSupportedProviderType lists the provider backends the LLM service knows
// how to talk to. Keep in sync with openAIEndpoint in llm_service.go.
func isSupportedProviderType(providerType string) bool {
	switch providerType {
	case "ollama", "openai_compatible", "pollinations", "huggingface":
		return true
	default:
		return false
	}
}

func optionalEncrypt(apiKey string, key []byte) (*string, error) {
	if apiKey == "" {
		return nil, nil
	}
	encrypted, err := EncryptSecret(apiKey, key)
	if err != nil {
		return nil, err
	}
	return &encrypted, nil
}

// normalizeParameters keeps an empty parameters set as nil so the DB stores JSON null.
func normalizeParameters(raw json.RawMessage) []byte {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}