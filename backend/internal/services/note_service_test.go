package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"intellisearch/internal/repositories"
)

func TestNoteCRUDAndOwnership(t *testing.T) {
	db := newTestDB(t)
	service := NewNoteService(repositories.NewNoteRepository(db))
	alice := uuid.New()
	bob := uuid.New()

	// Alice creates a note (with a source link from a search).
	sessionID := uuid.New()
	note, err := service.Create(alice, "Tokyo notes", "Best ramen at Ichiran.", "best ramen tokyo", &sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if note.Title != "Tokyo notes" || note.Content != "Best ramen at Ichiran." || note.SourceQuery != "best ramen tokyo" || note.SessionID == nil || *note.SessionID != sessionID {
		t.Fatalf("unexpected created note %+v", note)
	}
	// A second note for ordering.
	if _, err := service.Create(alice, "Second", "Another note", "", nil); err != nil {
		t.Fatal(err)
	}
	// Bob creates his own.
	bobNote, err := service.Create(bob, "Bob's", "Bob's private note", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	notes, err := service.List(alice)
	if err != nil || len(notes) != 2 {
		t.Fatalf("expected 2 notes for alice, got %d (err=%v)", len(notes), err)
	}
	if notes[0].Title != "Second" { // newest first
		t.Fatalf("expected newest note first, got %+v", notes[0])
	}

	// Alice updates her note; Bob cannot touch it.
	updated, err := service.Update(alice, note.ID, "Tokyo notes v2", "Updated content.")
	if err != nil || updated.Content != "Updated content." {
		t.Fatalf("update failed: %+v err=%v", updated, err)
	}
	if _, err := service.Update(bob, note.ID, "hijack", "hijacked"); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound for cross-user update, got %v", err)
	}
	// Bob cannot delete Alice's note either.
	if err := service.Delete(bob, note.ID); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound for cross-user delete, got %v", err)
	}
	// Alice can delete her own; Bob's remains.
	if err := service.Delete(alice, note.ID); err != nil {
		t.Fatal(err)
	}
	remaining, err := service.List(alice)
	if err != nil || len(remaining) != 1 {
		t.Fatalf("expected 1 note left for alice, got %d", len(remaining))
	}
	// Bob's note never appears in alice's list (per-user isolation).
	for _, note := range remaining {
		if note.ID == bobNote.ID {
			t.Fatalf("bob's note leaked into alice's list")
		}
	}
}

func TestNoteValidation(t *testing.T) {
	db := newTestDB(t)
	service := NewNoteService(repositories.NewNoteRepository(db))
	userID := uuid.New()
	// Empty title or content is rejected.
	if _, err := service.Create(userID, "", "content", "", nil); !errors.Is(err, ErrNoteInvalid) {
		t.Fatalf("expected ErrNoteInvalid for empty title, got %v", err)
	}
	if _, err := service.Create(userID, "title", "   ", "", nil); !errors.Is(err, ErrNoteInvalid) {
		t.Fatalf("expected ErrNoteInvalid for empty content, got %v", err)
	}
	// Oversized content is rejected.
	huge := strings.Repeat("x", 50001)
	if _, err := service.Create(userID, "title", huge, "", nil); !errors.Is(err, ErrNoteInvalid) {
		t.Fatalf("expected ErrNoteInvalid for oversized content, got %v", err)
	}
}
