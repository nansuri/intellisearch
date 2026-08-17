package repositories

import (
	"intellisearch/internal/models/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepository struct{ db *gorm.DB }

func NewMessageRepository(db *gorm.DB) *MessageRepository { return &MessageRepository{db: db} }
func (r *MessageRepository) Create(message *entities.Message) error {
	return r.db.Create(message).Error
}
func (r *MessageRepository) Update(message *entities.Message) error { return r.db.Save(message).Error }
func (r *MessageRepository) Sources(messageID uuid.UUID) ([]entities.SearchResult, error) {
	var sources []entities.SearchResult
	err := r.db.Where("message_id = ?", messageID).Order("position asc").Find(&sources).Error
	return sources, err
}
func (r *MessageRepository) CreateSources(sources []entities.SearchResult) error {
	if len(sources) == 0 {
		return nil
	}
	return r.db.Create(&sources).Error
}
