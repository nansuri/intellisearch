package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

func TestSessionGetIncludesSourcesAndImages(t *testing.T) {
	db := handlerTestDB(t)
	sessionRepo := repositories.NewSessionRepository(db)
	messageRepo := repositories.NewMessageRepository(db)
	sessionID := uuid.New()
	messageID := uuid.New()
	now := time.Now().UTC()
	if err := sessionRepo.Create(&entities.ChatSession{ID: sessionID, Title: "images test", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := messageRepo.Create(&entities.Message{ID: messageID, SessionID: sessionID, Role: entities.MessageRoleAssistant, Content: "answer", Status: entities.MessageStatusCompleted, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := messageRepo.CreateSources([]entities.SearchResult{{MessageID: messageID, Position: 1, Title: "s", URL: "https://example.com", Domain: "example.com", Snippet: "snip"}}); err != nil {
		t.Fatal(err)
	}
	if err := messageRepo.CreateImages([]entities.ImageResult{{MessageID: messageID, Position: 1, Title: "img", URL: "https://example.com/p", ThumbnailURL: "https://example.com/t.jpg", Source: "example.com", Width: 640, Height: 480}}); err != nil {
		t.Fatal(err)
	}

	handler := NewSessionHandler(sessionRepo, messageRepo, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID.String(), nil)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}

	handler.Get(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var envelope struct {
		Data struct {
			Messages []struct {
				Sources []json.RawMessage `json:"sources"`
				Images  []json.RawMessage `json:"images"`
			} `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(envelope.Data.Messages))
	}
	message := envelope.Data.Messages[0]
	if len(message.Sources) != 1 || len(message.Images) != 1 {
		t.Fatalf("expected sources and images on the assistant message, got sources=%d images=%d", len(message.Sources), len(message.Images))
	}
}
