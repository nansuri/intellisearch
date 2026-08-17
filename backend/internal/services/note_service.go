package services

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

var (
	ErrNoteInvalid = errors.New("invalid note")
	ErrNoteNotFound = errors.New("note not found")
)

// maxNoteContentLength caps a note's content (runes) so the notes mini-app
// cannot be abused as an unbounded text store.
const maxNoteContentLength = 50000

type NoteService struct {
	repo *repositories.NoteRepository
}

func NewNoteService(repo *repositories.NoteRepository) *NoteService {
	return &NoteService{repo: repo}
}

func (s *NoteService) List(userID uuid.UUID) ([]entities.Note, error) {
	return s.repo.List(userID)
}

// Create validates and persists a new note. sourceQuery/sessionID optionally
// link the note to the search it was saved from.
func (s *NoteService) Create(userID uuid.UUID, title, content, sourceQuery string, sessionID *uuid.UUID) (entities.Note, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" || len([]rune(content)) > maxNoteContentLength {
		return entities.Note{}, ErrNoteInvalid
	}
	now := time.Now().UTC()
	note := entities.Note{ID: uint64(now.UnixNano()), UserID: userID, Title: title, Content: content, SourceQuery: strings.TrimSpace(sourceQuery), SessionID: sessionID, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(&note); err != nil {
		return entities.Note{}, err
	}
	return note, nil
}

// Update edits a note's title/content, scoped to the owning user.
func (s *NoteService) Update(userID uuid.UUID, id uint64, title, content string) (entities.Note, error) {
	note, err := s.repo.Get(userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Note{}, ErrNoteNotFound
		}
		return entities.Note{}, err
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" || len([]rune(content)) > maxNoteContentLength {
		return entities.Note{}, ErrNoteInvalid
	}
	note.Title = title
	note.Content = content
	note.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(&note); err != nil {
		return entities.Note{}, err
	}
	return note, nil
}

// Delete removes a note scoped to the owning user.
func (s *NoteService) Delete(userID uuid.UUID, id uint64) error {
	affected, err := s.repo.Delete(userID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNoteNotFound
	}
	return nil
}
