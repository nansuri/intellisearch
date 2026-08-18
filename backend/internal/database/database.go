package database

import (
	"intellisearch/internal/config"
	"intellisearch/internal/models/entities"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
)

func Connect(cfg config.Config) (*gorm.DB, error) {
	if cfg.DBDriver == "sqlite" {
		if err := os.MkdirAll(filepath.Dir(cfg.DBSQLitePath), 0o750); err != nil {
			return nil, err
		}
		return gorm.Open(sqlite.Open(cfg.DBSQLitePath), &gorm.Config{})
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// seedSingleton inserts a singleton row only when the primary key is missing.
// Unlike FirstOrCreate, the lookup conditions on the primary key alone, so an
// admin-edited row (branding, queue knobs) is never re-inserted on boot.
func seedSingleton(db *gorm.DB, id uint, seed any) error {
	if err := db.First(seed, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return db.Create(seed).Error
		}
		return err
	}
	return nil
}

func MigrateAndSeed(db *gorm.DB, cfg config.Config) error {
	if err := db.AutoMigrate(&entities.User{}, &entities.SiteSettings{}, &entities.AIQueueConfig{}, &entities.AIProvider{}, &entities.ChatSession{}, &entities.Message{}, &entities.SearchResult{}, &entities.ImageResult{}, &entities.UsageLog{}, &entities.CrawlJob{}, &entities.SearchHistory{}, &entities.AnonymousUsage{}, &entities.Note{}, &entities.MapPoint{}, &entities.RegisterVisit{}); err != nil {
		return err
	}
	if err := seedSingleton(db, 1, &entities.SiteSettings{ID: 1, SiteName: "Intellisearch"}); err != nil {
		return err
	}
	if err := seedSingleton(db, 1, &entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10, SuggestionCacheHours: 6, DefaultDailyQuota: 3, MaxImageResults: 20}); err != nil {
		return err
	}
	// The seed-managed "local-ollama" provider mirrors the runtime env config
// (OLLAAMA_BASE_URL / AI_MODEL / AI_PROVIDER), which is the source of truth for
// host-run development. Insert it when missing and refresh its live fields on
// every boot so a changed .env no longer leaves a stale row behind.
	provider := &entities.AIProvider{Name: "local-ollama"}
	if err := db.Where("name = ?", provider.Name).First(provider).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&entities.AIProvider{ID: uuid.New(), Name: provider.Name, ProviderType: cfg.AIProvider, BaseURL: cfg.OllamaBaseURL, Model: cfg.AIModel, IsActive: true}).Error; err != nil {
			return err
		}
	} else if err := db.Model(provider).Updates(map[string]any{"provider_type": cfg.AIProvider, "base_url": cfg.OllamaBaseURL, "model": cfg.AIModel}).Error; err != nil {
		return err
	}
	if cfg.SuperOwnerEmail == "" || cfg.SuperOwnerPassword == "" {
		return nil
	}
	var count int64
	if err := db.Model(&entities.User{}).Where("role = ?", entities.RoleSuperOwner).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.SuperOwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return db.Create(&entities.User{ID: uuid.New(), Name: "Super Owner", Email: cfg.SuperOwnerEmail, PasswordHash: string(hash), Role: entities.RoleSuperOwner, Status: entities.StatusActive}).Error
}
