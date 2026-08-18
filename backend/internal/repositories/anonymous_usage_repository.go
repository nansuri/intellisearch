package repositories

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"intellisearch/internal/models/entities"
)

type AnonymousUsageRepository struct{ db *gorm.DB }

func NewAnonymousUsageRepository(db *gorm.DB) *AnonymousUsageRepository { return &AnonymousUsageRepository{db: db} }

// HashIP returns a one-way hash of a client IP so raw addresses are never
// stored (log hygiene: PII minimized). A static salt keeps hashes stable
// across restarts so the per-IP claim survives redeploys.
func HashIP(ip string) string {
	sum := sha256.Sum256([]byte("intellisearch-anon-ip:" + ip))
	return hex.EncodeToString(sum[:])
}

// ByVisitorID returns the usage row for a visitor token, or gorm.ErrRecordNotFound.
func (r *AnonymousUsageRepository) ByVisitorID(visitorID uuid.UUID) (entities.AnonymousUsage, error) {
	var row entities.AnonymousUsage
	err := r.db.Where("visitor_id = ?", visitorID).First(&row).Error
	return row, err
}

// Release removes a visitor's claim so a failed ask (provider down, timeout,
// queue full…) does not burn the guest's single allowance — only a successful
// AI usage consumes it. Best-effort: deleting a non-existent row is a no-op.
func (r *AnonymousUsageRepository) Release(visitorID uuid.UUID) error {
	return r.db.Where("visitor_id = ?", visitorID).Delete(&entities.AnonymousUsage{}).Error
}

// Claim atomically reserves the single anonymous AI allowance for a
// (visitorID, ipHash) pair: the insert is a no-op if either the visitor token
// or the IP is already claimed, then the winning row is returned. Callers
// decide whether the winner is the requesting visitor (allowed) or another
// visitor (this IP already used its allowance). The unique constraints make
// concurrent first asks race-safe: exactly one visitor wins each IP.
func (r *AnonymousUsageRepository) Claim(visitorID uuid.UUID, ipHash string) (entities.AnonymousUsage, error) {
	now := time.Now().UTC()
	row := entities.AnonymousUsage{VisitorID: visitorID, IPHash: ipHash, CreatedAt: now, UpdatedAt: now}
	if err := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return entities.AnonymousUsage{}, err
	}
	var winner entities.AnonymousUsage
	err := r.db.Where("visitor_id = ? OR ip_hash = ?", visitorID, ipHash).First(&winner).Error
	return winner, err
}

// Count returns the total number of unique anonymous visitors that have used
// the AI service.
func (r *AnonymousUsageRepository) Count() (int64, error) {
	var total int64
	err := r.db.Model(&entities.AnonymousUsage{}).Count(&total).Error
	return total, err
}

// CreatedSince returns the creation times of anonymous-usage claims since a
// cutoff (used to bucket daily/weekly visitor trends in the service layer).
func (r *AnonymousUsageRepository) CreatedSince(start time.Time) ([]time.Time, error) {
	var rows []entities.AnonymousUsage
	if err := r.db.Where("created_at >= ?", start).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	times := make([]time.Time, 0, len(rows))
	for _, row := range rows {
		times = append(times, row.CreatedAt)
	}
	return times, nil
}
