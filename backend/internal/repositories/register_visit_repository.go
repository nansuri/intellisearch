package repositories

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"intellisearch/internal/models/entities"
)

type RegisterVisitRepository struct{ db *gorm.DB }

func NewRegisterVisitRepository(db *gorm.DB) *RegisterVisitRepository { return &RegisterVisitRepository{db: db} }

// Claim records a unique visitor who opened the register page. The insert is a
// no-op (OnConflict DoNothing) when that visitor token already opened it, so
// replaying the page never inflates the count. The returned row is the winning
// row regardless; `created` reports whether this call made a new record.
func (r *RegisterVisitRepository) Claim(visitorID uuid.UUID, ipHash string) (entities.RegisterVisit, bool, error) {
	now := time.Now().UTC()
	row := entities.RegisterVisit{VisitorID: visitorID, IPHash: ipHash, CreatedAt: now, UpdatedAt: now}
	result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if result.Error != nil {
		return entities.RegisterVisit{}, false, result.Error
	}
	created := result.RowsAffected == 1
	var winner entities.RegisterVisit
	err := r.db.Where("visitor_id = ?", visitorID).First(&winner).Error
	return winner, created, err
}

// Count returns the number of unique visitors that have opened the register page.
func (r *RegisterVisitRepository) Count() (int64, error) {
	var total int64
	err := r.db.Model(&entities.RegisterVisit{}).Count(&total).Error
	return total, err
}

// CreatedSince returns the creation times of all register-page visits since a
// cutoff (used to bucket daily/weekly trends in the service layer).
func (r *RegisterVisitRepository) CreatedSince(start time.Time) ([]time.Time, error) {
	var rows []entities.RegisterVisit
	if err := r.db.Where("created_at >= ?", start).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	times := make([]time.Time, 0, len(rows))
	for _, row := range rows {
		times = append(times, row.CreatedAt)
	}
	return times, nil
}
