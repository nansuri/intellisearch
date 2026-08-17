package repositories

import (
	"intellisearch/internal/models/entities"
	"errors"
	"gorm.io/gorm"
)

type SiteRepository struct{ db *gorm.DB }

func NewSiteRepository(db *gorm.DB) *SiteRepository { return &SiteRepository{db: db} }
func (r *SiteRepository) Get() (entities.SiteSettings, error) {
	var value entities.SiteSettings
	err := r.db.First(&value, 1).Error
	return value, err
}
func (r *SiteRepository) Update(value entities.SiteSettings) error { return r.db.Save(&value).Error }
func (r *SiteRepository) Exists() (bool, error) {
	_, err := r.Get()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}
