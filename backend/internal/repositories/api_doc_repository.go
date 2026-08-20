package repositories

import (
	"gorm.io/gorm"

	"intellisearch/internal/models/entities"
)

// ApiDocRepository owns the seeded Mini Apps platform API documentation rows.
// The docs are data (not frontend code), so they can be edited in the database
// and rendered/exported by the Studio without a deploy.
type ApiDocRepository struct{ db *gorm.DB }

func NewApiDocRepository(db *gorm.DB) *ApiDocRepository { return &ApiDocRepository{db: db} }

// List returns every doc row, grouped and ordered for rendering: section then
// sort order then title.
func (r *ApiDocRepository) List() ([]entities.MiniAppApiDoc, error) {
	var docs []entities.MiniAppApiDoc
	err := r.db.Order("sort_order ASC, id ASC").Find(&docs).Error
	return docs, err
}