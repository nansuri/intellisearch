package repositories

import (
	"intellisearch/internal/models/entities"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProviderRepository struct{ db *gorm.DB }

func NewProviderRepository(db *gorm.DB) *ProviderRepository { return &ProviderRepository{db: db} }

func (r *ProviderRepository) List() ([]entities.AIProvider, error) {
	var providers []entities.AIProvider
	err := r.db.Order("created_at asc").Find(&providers).Error
	return providers, err
}

func (r *ProviderRepository) Active() (entities.AIProvider, error) {
	var provider entities.AIProvider
	err := r.db.Where("is_active = ?", true).First(&provider).Error
	return provider, err
}

func (r *ProviderRepository) ByID(id uuid.UUID) (entities.AIProvider, error) {
	var provider entities.AIProvider
	err := r.db.First(&provider, "id = ?", id).Error
	return provider, err
}

func (r *ProviderRepository) Create(provider *entities.AIProvider) error { return r.db.Create(provider).Error }

func (r *ProviderRepository) Update(provider *entities.AIProvider) error { return r.db.Save(provider).Error }

func (r *ProviderRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.AIProvider{}, "id = ?", id).Error
}

// SetActive marks the given provider as the single active one. The provided
// provider is loaded and persisted so the owner-facing update stays a plain save.
func (r *ProviderRepository) SetActive(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entities.AIProvider{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
			return err
		}
		result := tx.Model(&entities.AIProvider{}).Where("id = ?", id).Update("is_active", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("provider not found")
		}
		return nil
	})
}
