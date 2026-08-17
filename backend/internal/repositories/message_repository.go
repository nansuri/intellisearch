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
func (r *MessageRepository) CreateImages(images []entities.ImageResult) error {
	if len(images) == 0 {
		return nil
	}
	return r.db.Create(&images).Error
}
func (r *MessageRepository) Images(messageID uuid.UUID) ([]entities.ImageResult, error) {
	var images []entities.ImageResult
	err := r.db.Where("message_id = ?", messageID).Order("position asc").Find(&images).Error
	return images, err
}

// Summaries returns the content of the given messages keyed by message ID,
// used to render search-history summaries without duplicating answer text in
// the search_history table.
func (r *MessageRepository) Summaries(messageIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string, len(messageIDs))
	if len(messageIDs) == 0 {
		return out, nil
	}
	var messages []entities.Message
	if err := r.db.Where("id IN ?", messageIDs).Find(&messages).Error; err != nil {
		return nil, err
	}
	for _, message := range messages {
		out[message.ID] = message.Content
	}
	return out, nil
}
