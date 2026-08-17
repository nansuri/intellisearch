package repositories

import (
	"intellisearch/internal/models/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionRepository struct{ db *gorm.DB }

func NewSessionRepository(db *gorm.DB) *SessionRepository { return &SessionRepository{db: db} }
func (r *SessionRepository) Create(session *entities.ChatSession) error {
	return r.db.Create(session).Error
}
func (r *SessionRepository) Get(id uuid.UUID) (entities.ChatSession, error) {
	var session entities.ChatSession
	err := r.db.First(&session, "id = ?", id).Error
	return session, err
}
func (r *SessionRepository) Messages(id uuid.UUID) ([]entities.Message, error) {
	var messages []entities.Message
	err := r.db.Where("session_id = ?", id).Order("created_at asc").Find(&messages).Error
	return messages, err
}
func (r *SessionRepository) Update(session *entities.ChatSession) error { return r.db.Save(session).Error }
