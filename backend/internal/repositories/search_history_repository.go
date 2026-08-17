package repositories

import (
	"intellisearch/internal/models/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SearchHistoryRepository struct{ db *gorm.DB }

func NewSearchHistoryRepository(db *gorm.DB) *SearchHistoryRepository { return &SearchHistoryRepository{db: db} }

func (r *SearchHistoryRepository) Create(entry *entities.SearchHistory) error {
	return r.db.Create(entry).Error
}

// Recent returns a user's most recent history entries, newest first.
func (r *SearchHistoryRepository) Recent(userID uuid.UUID, limit int) ([]entities.SearchHistory, error) {
	if limit < 1 {
		limit = 20
	}
	var entries []entities.SearchHistory
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Limit(limit).Find(&entries).Error
	return entries, err
}

// RecentDistinct returns the user's most recently used queries without
// duplicates (each query appears once, ordered by its latest use). The GROUP BY
// works identically on SQLite and PostgreSQL.
func (r *SearchHistoryRepository) RecentDistinct(userID uuid.UUID, limit int) ([]string, error) {
	if limit < 1 {
		limit = 10
	}
	var queries []string
	err := r.db.Model(&entities.SearchHistory{}).
		Select("query").
		Where("user_id = ?", userID).
		Group("query").
		Order("MAX(created_at) DESC").
		Limit(limit).
		Scan(&queries).Error
	return queries, err
}

// Clear removes every history entry for a user.
func (r *SearchHistoryRepository) Clear(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&entities.SearchHistory{}).Error
}
