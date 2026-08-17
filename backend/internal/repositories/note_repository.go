package repositories

import (
	"intellisearch/internal/models/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NoteRepository struct{ db *gorm.DB }

func NewNoteRepository(db *gorm.DB) *NoteRepository { return &NoteRepository{db: db} }

// List returns the user's notes, newest first.
func (r *NoteRepository) List(userID uuid.UUID) ([]entities.Note, error) {
	var notes []entities.Note
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&notes).Error
	return notes, err
}

// Get returns one note, scoped to the owning user (a cross-user id returns
// gorm.ErrRecordNotFound).
func (r *NoteRepository) Get(userID uuid.UUID, id uint64) (entities.Note, error) {
	var note entities.Note
	err := r.db.Where("user_id = ? AND id = ?", userID, id).First(&note).Error
	return note, err
}

func (r *NoteRepository) Create(note *entities.Note) error { return r.db.Create(note).Error }
func (r *NoteRepository) Update(note *entities.Note) error { return r.db.Save(note).Error }

// Delete removes one note, scoped to the owning user (returns the affected
// row count so callers can tell a cross-user delete apart from a real one).
func (r *NoteRepository) Delete(userID uuid.UUID, id uint64) (int64, error) {
	result := r.db.Where("user_id = ? AND id = ?", userID, id).Delete(&entities.Note{})
	return result.RowsAffected, result.Error
}
