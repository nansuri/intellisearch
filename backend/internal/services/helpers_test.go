package services

import (
	"fmt"
	"sync/atomic"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"intellisearch/internal/models/entities"
)

var testDBCounter atomic.Int64

// newTestDB returns an isolated in-memory SQLite database with the schema
// migrated. A unique shared-cache name keeps every test's DB separate.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := fmt.Sprintf("file:ai-test-%d?mode=memory&cache=shared", testDBCounter.Add(1))
	db, err := gorm.Open(sqlite.Open(name), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&entities.User{}, &entities.SiteSettings{}, &entities.AIQueueConfig{}, &entities.AIProvider{}, &entities.ChatSession{}, &entities.Message{}, &entities.SearchResult{}, &entities.ImageResult{}, &entities.UsageLog{}, &entities.CrawlJob{}, &entities.SearchHistory{}, &entities.AnonymousUsage{}, &entities.Note{}, &entities.MapPoint{}); err != nil {
		t.Fatal(err)
	}
	return db
}
