package repositories

import (
	"intellisearch/internal/models/entities"
	"gorm.io/gorm"
)

type QueueConfigRepository struct{ db *gorm.DB }

func NewQueueConfigRepository(db *gorm.DB) *QueueConfigRepository { return &QueueConfigRepository{db: db} }

func (r *QueueConfigRepository) Get() (entities.AIQueueConfig, error) {
	var config entities.AIQueueConfig
	err := r.db.First(&config, 1).Error
	return config, err
}

func (r *QueueConfigRepository) Update(config entities.AIQueueConfig) error { return r.db.Save(&config).Error }
