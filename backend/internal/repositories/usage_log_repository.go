package repositories

import (
	"intellisearch/internal/models/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type UsageLogRepository struct{ db *gorm.DB }

func NewUsageLogRepository(db *gorm.DB) *UsageLogRepository       { return &UsageLogRepository{db: db} }
func (r *UsageLogRepository) Create(log *entities.UsageLog) error { return r.db.Create(log).Error }
func (r *UsageLogRepository) Update(log *entities.UsageLog) error { return r.db.Save(log).Error }
func (r *UsageLogRepository) CompletedToday(userID uuid.UUID, start time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&entities.UsageLog{}).Where("user_id = ? AND status = ? AND created_at >= ?", userID, "completed", start).Count(&count).Error
	return count, err
}

// CountSince counts all usage attempts (any status) for a user since start; used for the daily question quota.
func (r *UsageLogRepository) CountSince(userID uuid.UUID, start time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&entities.UsageLog{}).Where("user_id = ? AND created_at >= ?", userID, start).Count(&count).Error
	return count, err
}

// CountBetween counts all usage attempts between two timestamps (exclusive of end).
func (r *UsageLogRepository) CountBetween(start, end time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&entities.UsageLog{}).Where("created_at >= ? AND created_at < ?", start, end).Count(&count).Error
	return count, err
}

// CreatedSince returns the created_at timestamps of every usage log since start,
// oldest first. The service buckets them per day/week so bucketing stays
// identical across the SQLite (local) and PostgreSQL (production) drivers.
func (r *UsageLogRepository) CreatedSince(start time.Time) ([]time.Time, error) {
	var values []time.Time
	err := r.db.Model(&entities.UsageLog{}).Where("created_at >= ?", start).Order("created_at asc").Pluck("created_at", &values).Error
	return values, err
}

// CountFailedSince counts failed asks (with an error code) since start.
func (r *UsageLogRepository) CountFailedSince(start time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&entities.UsageLog{}).Where("status = ? AND error_code <> '' AND created_at >= ?", "failed", start).Count(&count).Error
	return count, err
}

// ActiveUsersSince counts distinct users with usage since start.
func (r *UsageLogRepository) ActiveUsersSince(start time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&entities.UsageLog{}).Distinct("user_id").Where("user_id IS NOT NULL AND created_at >= ?", start).Count(&count).Error
	return count, err
}

// TopQueries returns the most-asked sanitized queries with their counts.
func (r *UsageLogRepository) TopQueries(limit int) ([]TopQuery, error) {
	if limit < 1 {
		limit = 10
	}
	var rows []TopQuery
	err := r.db.Model(&entities.UsageLog{}).Select("query, COUNT(*) AS count").Group("query").Order("count DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

// PerUserUsage returns each user's ask count, ordered by usage.
func (r *UsageLogRepository) PerUserUsage() ([]UserUsage, error) {
	var rows []UserUsage
	err := r.db.Model(&entities.UsageLog{}).Select("user_id, COUNT(*) AS count").Where("user_id IS NOT NULL").Group("user_id").Order("count DESC").Scan(&rows).Error
	return rows, err
}

// ErrorGroups returns failed asks grouped by error code with counts and last
// seen. An optional filterType restricts to one error code (e.g. "AISY01002").
// last_seen is scanned as a string so the query runs on both SQLite and
// PostgreSQL (MAX(timestamp) scans as string or time.Time depending on driver).
func (r *UsageLogRepository) ErrorGroups(filterType string, limit int) ([]ErrorGroup, error) {
	if limit < 1 {
		limit = 20
	}
	tx := r.db.Model(&entities.UsageLog{}).Where("status = ? AND error_code <> ''", "failed")
	if filterType != "" {
		tx = tx.Where("error_code = ?", filterType)
	}
	var rows []ErrorGroup
	err := tx.Select("error_code, error_message, COUNT(*) AS count, MAX(created_at) AS last_seen").Group("error_code, error_message").Order("count DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

// Latencies returns the latency_ms of every completed ask since start, oldest first.
func (r *UsageLogRepository) Latencies(start time.Time) ([]int, error) {
	var values []int
	err := r.db.Model(&entities.UsageLog{}).Where("status = ? AND latency_ms > 0 AND created_at >= ?", "completed", start).Order("latency_ms asc").Pluck("latency_ms", &values).Error
	return values, err
}

// ProviderPerformance aggregates success counts per provider for the AI-stats panel.
func (r *UsageLogRepository) ProviderPerformance() ([]ProviderStats, error) {
	var rows []ProviderStats
	err := r.db.Model(&entities.UsageLog{}).Select(
		"provider_id, SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS successes, COUNT(*) AS total",
	).Where("provider_id IS NOT NULL").Group("provider_id").Scan(&rows).Error
	return rows, err
}

type TopQuery struct {
	Query string `json:"query"`
	Count int64  `json:"count"`
}

type UserUsage struct {
	UserID uuid.UUID `json:"userId"`
	Count  int64     `json:"count"`
}

type ErrorGroup struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	Count        int64  `json:"count"`
	LastSeen     string `json:"lastSeen"`
}

type ProviderStats struct {
	ProviderID uuid.UUID `json:"providerId"`
	Successes  int64     `json:"successes"`
	Total      int64     `json:"total"`
}