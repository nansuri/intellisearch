package handlers

import (
	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

type UserHandler struct{ service *services.UserService }

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Me returns the user's profile plus today's AI usage and remaining quota.
func (h *UserHandler) Me(c *gin.Context) {
	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	user, err := h.service.Get(userID)
	if err != nil {
		middleware.JSON(c, http.StatusNotFound, contracts.Fail(contracts.USER01001, "Your account could not be found."))
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
		middleware.JSON(c, http.StatusBadRequest, contracts.Fail(contracts.USER01002, "Enter a valid name and email address."))
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
		middleware.JSON(c, http.StatusBadRequest, contracts.Fail(code, message))
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(user))
}

// Avatar handles single-file avatar uploads.
func (h *UserHandler) Avatar(c *gin.Context) {
	header, err := c.FormFile("avatar")
	if err != nil {
		middleware.JSON(c, http.StatusBadRequest, contracts.Fail(contracts.USER02001, "Choose an image to upload for your avatar."))
		return
	}
	data, err := readUpload(header)
	if err != nil {
		middleware.JSON(c, http.StatusBadRequest, contracts.Fail(contracts.USER02001, "That image could not be read."))
		return
	}
	url, err := h.service.Avatar(c.MustGet(middleware.UserIDKey).(uuid.UUID), header.Filename, data)
	if err != nil {
		if err == services.ErrUploadRejected {
			middleware.JSON(c, http.StatusBadRequest, contracts.Fail(contracts.USER02002, "The avatar must be a JPG, PNG, GIF, or WebP under 2 MB."))
			return
		}
		middleware.JSON(c, http.StatusInternalServerError, contracts.Fail(contracts.USER02001, "Your avatar could not be saved."))
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