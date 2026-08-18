package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/services"
)

// AppsHandler backs the user-facing mini apps: the Notes app (personal
// notes, including "save summary to notes" from the result page) and the
// Translator app (a server-side proxy to the LibreTranslate container — the
// browser never talks to it directly).
type AppsHandler struct {
	notes     *services.NoteService
	translate *services.TranslateService
	limiter   services.Limiter
}

func NewAppsHandler(notes *services.NoteService, translate *services.TranslateService, limiter services.Limiter) *AppsHandler {
	return &AppsHandler{notes: notes, translate: translate, limiter: limiter}
}

// ---------------------------------------------------------------------------
// Notes
// ---------------------------------------------------------------------------

func (h *AppsHandler) ListNotes(c *gin.Context) {
	notes, err := h.notes.List(c.MustGet(middleware.UserIDKey).(uuid.UUID))
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.NOTE01001, "Your notes could not be loaded.", "notes list failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"items": notes}))
}

func (h *AppsHandler) CreateNote(c *gin.Context) {
	var request struct {
		Title       string     `json:"title"`
		Content     string     `json:"content"`
		SourceQuery string     `json:"sourceQuery"`
		SessionID   *uuid.UUID `json:"sessionId"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.NOTE01002, "Enter a title and some content for your note.", "parse create note", err)
		return
	}
	note, err := h.notes.Create(c.MustGet(middleware.UserIDKey).(uuid.UUID), request.Title, request.Content, request.SourceQuery, request.SessionID)
	if err != nil {
		if errors.Is(err, services.ErrNoteInvalid) {
			middleware.RespondError(c, http.StatusBadRequest, contracts.NOTE01002, "Enter a title and some content for your note.", "invalid note", err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, contracts.NOTE01002, "That note could not be saved.", "note create failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(note))
}

func (h *AppsHandler) UpdateNote(c *gin.Context) {
	id, ok := noteIDParam(c)
	if !ok {
		return
	}
	var request struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.NOTE01002, "Enter a title and some content for your note.", "parse update note", err)
		return
	}
	note, err := h.notes.Update(c.MustGet(middleware.UserIDKey).(uuid.UUID), id, request.Title, request.Content)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNoteNotFound):
			middleware.RespondError(c, http.StatusNotFound, contracts.NOTE01003, "That note no longer exists.", "update note not found", err)
		case errors.Is(err, services.ErrNoteInvalid):
			middleware.RespondError(c, http.StatusBadRequest, contracts.NOTE01002, "Enter a title and some content for your note.", "invalid note", err)
		default:
			middleware.RespondError(c, http.StatusInternalServerError, contracts.NOTE01002, "That note could not be saved.", "note update failed", err)
		}
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(note))
}

func (h *AppsHandler) DeleteNote(c *gin.Context) {
	id, ok := noteIDParam(c)
	if !ok {
		return
	}
	if err := h.notes.Delete(c.MustGet(middleware.UserIDKey).(uuid.UUID), id); err != nil {
		if errors.Is(err, services.ErrNoteNotFound) {
			middleware.RespondError(c, http.StatusNotFound, contracts.NOTE01003, "That note no longer exists.", "delete note not found", err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, contracts.NOTE01004, "That note could not be deleted.", "note delete failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"deleted": true}))
}

// ---------------------------------------------------------------------------
// Translator
// ---------------------------------------------------------------------------

// Languages proxies LibreTranslate's language list for the translator UI.
func (h *AppsHandler) TranslateLanguages(c *gin.Context) {
	if !h.translate.Available() {
		middleware.RespondError(c, http.StatusServiceUnavailable, contracts.TRAN01001, "The translator is not configured on this deployment.", "translator not configured", nil)
		return
	}
	languages, err := h.translate.Languages(c.Request.Context())
	if err != nil {
		middleware.RespondError(c, http.StatusBadGateway, contracts.TRAN01001, "The translator is temporarily unavailable.", "translate languages failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"languages": languages}))
}

// Translate proxies a translation request, rate-limited per user so the
// LibreTranslate container cannot be spammed through the API.
func (h *AppsHandler) Translate(c *gin.Context) {
	if !h.translate.Available() {
		middleware.RespondError(c, http.StatusServiceUnavailable, contracts.TRAN01001, "The translator is not configured on this deployment.", "translator not configured", nil)
		return
	}
	var request struct {
		Q      string `json:"q"`
		Source string `json:"source"`
		Target string `json:"target"`
		Format string `json:"format"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.TRAN01002, "Enter text to translate and choose a target language.", "parse translate request", err)
		return
	}
	allowed, err := h.limiter.Allow(c.Request.Context(), "translate", c.MustGet(middleware.UserIDKey).(uuid.UUID).String(), 30, time.Minute)
	if err != nil {
		logrus.WithError(err).WithField("scope", "translate").Error("rate limiter unavailable; allowing request")
	} else if !allowed {
		middleware.RespondError(c, http.StatusTooManyRequests, contracts.TRAN01003, "You're translating too quickly — slow down and try again.", "translate rate limited", nil)
		return
	}
	translated, err := h.translate.Translate(c.Request.Context(), request.Q, request.Source, request.Target, request.Format)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTranslateInvalid):
			middleware.RespondError(c, http.StatusBadRequest, contracts.TRAN01002, "Enter text to translate and choose a target language.", "invalid translate request", err)
		default:
			middleware.RespondError(c, http.StatusBadGateway, contracts.TRAN01001, "The translator is temporarily unavailable.", "translate failed", err)
		}
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"translatedText": translated}))
}

// noteIDParam parses the :id path param (numeric note id).
func noteIDParam(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.NOTE01003, "That note could not be found.", "parse note id", err)
		return 0, false
	}
	return id, true
}
