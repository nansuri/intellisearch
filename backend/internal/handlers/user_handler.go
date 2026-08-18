package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/services"
)

type UserHandler struct {
	service *services.UserService
	history *services.SearchHistoryService
}

func NewUserHandler(service *services.UserService, history *services.SearchHistoryService) *UserHandler {
	return &UserHandler{service: service, history: history}
}

// Me returns the user's profile plus today's AI usage and remaining quota.
func (h *UserHandler) Me(c *gin.Context) {
	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	user, err := h.service.Get(userID)
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, contracts.USER01001, "Your account could not be found.", "load profile", err)
		return
	}
	used, quota, err := h.service.Usage(userID)
	if err != nil {
		used, quota = 0, user.AIDailyQuota
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{
		"id":           user.ID,
		"name":         user.Name,
		"email":        user.Email,
		"role":         user.Role,
		"status":       user.Status,
		"avatarUrl":    user.AvatarURL,
		"aiDailyQuota": user.AIDailyQuota,
		"lastLoginAt":  user.LastLoginAt,
		"createdAt":    user.CreatedAt,
		"usage": gin.H{"usedToday": used, "quota": quota, "remaining": remaining(used, quota)},
	}))
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	var request struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.USER01002, "Enter a valid name and email address.", "parse profile update", err)
		return
	}
	user, err := h.service.UpdateProfile(c.MustGet(middleware.UserIDKey).(uuid.UUID), request.Name, request.Email)
	if err != nil {
		code := contracts.USER01001
		message := "Your account could not be found."
		if err == services.ErrProfileInvalid {
			code = contracts.USER01002
			message = "Enter a valid name and email address."
		}
		middleware.RespondError(c, http.StatusBadRequest, code, message, "profile update failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(user))
}

// History returns the signed-in user's recent searches, newest first, each with
// an on-demand summary of the answer (truncated server-side).
func (h *UserHandler) History(c *gin.Context) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}
	entries, err := h.history.RecentDetailed(c.MustGet(middleware.UserIDKey).(uuid.UUID), limit)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.USER03001, "Your search history could not be loaded.", "search history load failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"items": entries}))
}

// Suggestions returns AI-composed follow-up questions derived from the user's
// search history. Results are cached server-side for a configurable window
// (ai_queue_config.suggestion_cache_hours); ?refresh=1 bypasses the cache (the
// main page's ↻ button). Provider failures degrade to an empty list so the UI
// just hides the row.
func (h *UserHandler) Suggestions(c *gin.Context) {
	refresh := c.Query("refresh") == "1"
	suggestions, err := h.history.Suggestions(c.Request.Context(), c.MustGet(middleware.UserIDKey).(uuid.UUID), refresh)
	if err != nil {
		if !errors.Is(err, services.ErrHistoryEmpty) {
			logrus.WithError(err).Error("search history suggestions failed")
		}
		suggestions = []string{}
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"suggestions": suggestions}))
}

// ClearHistory deletes all of the signed-in user's search history.
func (h *UserHandler) ClearHistory(c *gin.Context) {
	if err := h.history.Clear(c.MustGet(middleware.UserIDKey).(uuid.UUID)); err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.USER03002, "Your search history could not be cleared.", "search history clear failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"cleared": true}))
}

// Avatar handles single-file avatar uploads.
func (h *UserHandler) Avatar(c *gin.Context) {
	header, err := c.FormFile("avatar")
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.USER02001, "Choose an image to upload for your avatar.", "missing avatar file", err)
		return
	}
	data, err := readUpload(header)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.USER02001, "That image could not be read.", "unreadable avatar file", err)
		return
	}
	url, err := h.service.Avatar(c.MustGet(middleware.UserIDKey).(uuid.UUID), header.Filename, data)
	if err != nil {
		if err == services.ErrUploadRejected {
			middleware.RespondError(c, http.StatusBadRequest, contracts.USER02002, "The avatar must be a JPG, PNG, GIF, or WebP under 2 MB.", "avatar upload rejected", err)
			return
		}
		// Avatar saves share the logo problem: a missing/unwritable UPLOADS_DIR
		// in the container surfaces here — log the cause for diagnosis.
		middleware.RespondError(c, http.StatusInternalServerError, contracts.USER02001, "Your avatar could not be saved.", "avatar upload failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"avatarUrl": url}))
}

func remaining(used int64, quota int) int64 {
	if quota <= 0 {
		return -1 // unlimited
	}
	remaining := int64(quota) - used
	if remaining < 0 {
		return 0
	}
	return remaining
}