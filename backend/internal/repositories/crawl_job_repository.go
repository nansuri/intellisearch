package repositories

import (
	"intellisearch/internal/models/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type CrawlJobRepository struct{ db *gorm.DB }

func NewCrawlJobRepository(db *gorm.DB) *CrawlJobRepository { return &CrawlJobRepository{db: db} }

func (r *CrawlJobRepository) Create(job *entities.CrawlJob) error { return r.db.Create(job).Error }

func (r *CrawlJobRepository) UpdateStatus(id uuid.UUID, status string) error {
	return r.db.Model(&entities.CrawlJob{}).Where("id = ?", id).Updates(map[string]any{"status": status, "finished_at": time.Now().UTC()}).Error
}
