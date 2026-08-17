package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"intellisearch/internal/models/entities"
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
func (r *MessageRepository) CreateMapPoints(points []entities.MapPoint) error {
	if len(points) == 0 {
		return nil
	}
	return r.db.Create(&points).Error
}
func (r *MessageRepository) MapPoints(messageID uuid.UUID) ([]entities.MapPoint, error) {
	var points []entities.MapPoint
	err := r.db.Where("message_id = ?", messageID).Order("position asc").Find(&points).Error
	return points, err
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
