package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
	"intellisearch/internal/services"
)

type SessionHandler struct {
	sessions *repositories.SessionRepository
	messages *repositories.MessageRepository
	auth     *services.AuthService
}

func NewSessionHandler(sessions *repositories.SessionRepository, messages *repositories.MessageRepository, auth *services.AuthService) *SessionHandler {
	return &SessionHandler{sessions: sessions, messages: messages, auth: auth}
}

// Get returns one session with its message history and per-message sources.
// Anonymous sessions (no owner) are readable by id; owned sessions require the owner.
func (h *SessionHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, contracts.SESS01001, "That conversation could not be found.", "parse session id", err)
		return
	}
	session, err := h.sessions.Get(id)
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, contracts.SESS01001, "That conversation could not be found.", "load session", err)
		return
	}
	if session.UserID != nil {
		callerID, authErr := h.optionalUserID(c)
		if authErr != nil || callerID == nil || *callerID != *session.UserID {
			middleware.RespondError(c, http.StatusForbidden, contracts.SESS01002, "You don't have access to that conversation.", "session access denied", authErr)
			return
		}
	}
	messages, err := h.sessions.Messages(id)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.SESS01001, "That conversation could not be loaded.", "load session messages", err)
		return
	}
	views := make([]gin.H, 0, len(messages))
	for _, message := range messages {
		view := gin.H{"id": message.ID, "role": message.Role, "content": message.Content, "status": message.Status, "createdAt": message.CreatedAt}
		if message.Role == entities.MessageRoleAssistant {
			sources, err := h.messages.Sources(message.ID)
			if err == nil {
				view["sources"] = sources
			}
			images, err := h.messages.Images(message.ID)
			if err == nil {
				view["images"] = images
			}
			mapPoints, err := h.messages.MapPoints(message.ID)
			if err == nil {
				view["mapPoints"] = mapPoints
			}
		}
		views = append(views, view)
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{
		"sessionId": session.ID,
		"title":     session.Title,
		"createdAt": session.CreatedAt,
		"messages":  views,
	}))
}

func (h *SessionHandler) optionalUserID(c *gin.Context) (*uuid.UUID, error) {
	raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if raw == c.GetHeader("Authorization") || raw == "" {
		return nil, nil
	}
	claims, err := h.auth.Parse(raw)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
