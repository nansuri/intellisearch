package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/services"
)

// AdminHandler exposes the Super Owner control-panel surface: user CRUD,
// statistics, AI provider management, queue knobs, branding, and the Ollama
// server introspection helpers for the provider form.
type AdminHandler struct {
	users  *services.UserService
	admin  *services.AdminService
	stats  *services.StatsService
	ollama *services.OllamaService
}

func NewAdminHandler(users *services.UserService, admin *services.AdminService, stats *services.StatsService, ollama *services.OllamaService) *AdminHandler {
	return &AdminHandler{users: users, admin: admin, stats: stats, ollama: ollama}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, total, err := h.users.List(c.Query("q"), queryInt(c, "page", 1), queryInt(c, "page_size", 20))
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN01001, "Users could not be loaded.", "list users", err)
		return
	}
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"users": users, "total": total, "page": page, "pageSize": pageSize}))
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var request struct {
		Name         string `json:"name"`
		Email        string `json:"email"`
		Password     string `json:"password"`
		Role         string `json:"role"`
		AIDailyQuota int    `json:"aiDailyQuota"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN01001, "Enter valid user details.", "parse create user", err)
		return
	}
	user, err := h.users.Create(request.Name, request.Email, request.Password, request.Role, request.AIDailyQuota)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN01001, "That user could not be created — check the details and try again.", "create user failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(user))
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id, ok := userIDParam(c)
	if !ok {
		return
	}
	var request struct {
		Role         *string `json:"role"`
		Status       *string `json:"status"`
		AIDailyQuota *int    `json:"aiDailyQuota"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN01001, "Enter valid user details.", "parse update user", err)
		return
	}
	quota := -1
	if request.AIDailyQuota != nil {
		quota = *request.AIDailyQuota
	}
	role, status := "", ""
	if request.Role != nil {
		role = *request.Role
	}
	if request.Status != nil {
		status = *request.Status
	}
	user, err := h.users.Update(id, role, status, quota)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN01001, "That user could not be updated.", "update user failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(user))
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, ok := userIDParam(c)
	if !ok {
		return
	}
	if err := h.users.Delete(id); err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN01001, "That user could not be deleted.", "delete user failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"deleted": true}))
}

func (h *AdminHandler) Stats(c *gin.Context) {
	stats, err := h.stats.UserStats()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN02001, "Statistics could not be computed.", "compute user stats", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(stats))
}

// Visitors returns the unique user/visitor summary for the control panel:
// registered accounts, active users, anonymous AI visitors, and unique
// register-page visitors — each with daily/weekly trends.
func (h *AdminHandler) Visitors(c *gin.Context) {
	stats, err := h.stats.VisitorStats()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.STTS01001, "Visitor statistics could not be computed.", "compute visitor stats", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(stats))
}

func (h *AdminHandler) AIStats(c *gin.Context) {
	stats, err := h.stats.AIStats(c.Query("type"))
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN02001, "AI statistics could not be computed.", "compute ai stats", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(stats))
}

func (h *AdminHandler) Trends(c *gin.Context) {
	trends, err := h.stats.Trends()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN02001, "Trend statistics could not be computed.", "compute trends", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(trends))
}

// TrendingWords returns word-level search trends (aggregated, masked — never
// verbatim queries) for the control panel's privacy-safe trending chart.
func (h *AdminHandler) TrendingWords(c *gin.Context) {
	trends, err := h.stats.TrendingWords(c.Query("window"))
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN02001, "Trend statistics could not be computed.", "compute trending words", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(trends))
}

func (h *AdminHandler) ListProviders(c *gin.Context) {
	providers, err := h.admin.Providers()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN03001, "Providers could not be loaded.", "list providers", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"providers": providers}))
}

func (h *AdminHandler) CreateProvider(c *gin.Context) {
	var request struct {
		Name         string          `json:"name"`
		ProviderType string          `json:"providerType"`
		BaseURL      string          `json:"baseUrl"`
		Model        string          `json:"model"`
		Parameters   json.RawMessage `json:"parameters"`
		APIKey       string          `json:"apiKey"`
		IsActive     bool            `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN03002, "Enter a valid provider configuration.", "parse create provider", err)
		return
	}
	provider, err := h.admin.CreateProvider(request.Name, request.ProviderType, request.BaseURL, request.Model, request.Parameters, request.APIKey, request.IsActive)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN03002, "That provider could not be saved — check the configuration.", "create provider failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(provider))
}

func (h *AdminHandler) UpdateProvider(c *gin.Context) {
	id, ok := providerIDParam(c)
	if !ok {
		return
	}
	var request struct {
		Name         *string         `json:"name"`
		ProviderType *string         `json:"providerType"`
		BaseURL      *string         `json:"baseUrl"`
		Model        *string         `json:"model"`
		Parameters   json.RawMessage `json:"parameters"`
		APIKey       string          `json:"apiKey"`
		IsActive     bool            `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN03002, "Enter a valid provider configuration.", "parse update provider", err)
		return
	}
	provider, err := h.admin.UpdateProvider(id, strPtr(request.Name), strPtr(request.ProviderType), strPtr(request.BaseURL), strPtr(request.Model), request.Parameters, request.APIKey, request.IsActive)
	if err != nil {
		if errors.Is(err, services.ErrProviderNotFound) {
			middleware.RespondError(c, http.StatusNotFound, contracts.ADMN03001, "That provider no longer exists.", "update provider not found", err)
			return
		}
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN03002, "That provider could not be saved — check the configuration.", "update provider failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(provider))
}

func (h *AdminHandler) DeleteProvider(c *gin.Context) {
	id, ok := providerIDParam(c)
	if !ok {
		return
	}
	if err := h.admin.DeleteProvider(id); err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN03001, "That provider could not be deleted.", "delete provider failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"deleted": true}))
}

// OllamaModels lists the models on an Ollama server so the provider form can
// offer a picker. The base URL comes from the provider being configured and is
// fetched server-side (the browser never calls Ollama directly).
func (h *AdminHandler) OllamaModels(c *gin.Context) {
	models, err := h.ollama.Models(c.Request.Context(), c.Query("baseUrl"))
	if err != nil {
		h.respondOllamaError(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"models": models}))
}

// OllamaHealth reports the server version plus loaded-model stats (/api/ps).
func (h *AdminHandler) OllamaHealth(c *gin.Context) {
	health, err := h.ollama.Health(c.Request.Context(), c.Query("baseUrl"))
	if err != nil {
		h.respondOllamaError(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"ok": true, "version": health.Version, "runningModels": health.RunningModels}))
}

func (h *AdminHandler) respondOllamaError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrInvalidOllamaURL) {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN06001, "Enter a valid Ollama base URL (http:// or https://).", "invalid ollama url", err)
		return
	}
	middleware.RespondError(c, http.StatusBadGateway, contracts.ADMN06001, "Couldn't reach the Ollama server — check the base URL and that it's running.", "ollama request failed", err)
}

func (h *AdminHandler) QueueConfig(c *gin.Context) {
	config, err := h.admin.QueueConfig()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN04001, "Queue configuration could not be loaded.", "load queue config", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(config))
}

func (h *AdminHandler) UpdateQueueConfig(c *gin.Context) {
	var request struct {
		MaxConcurrent        int `json:"maxConcurrent"`
		MaxQueueSize         int `json:"maxQueueSize"`
		RequestTimeoutMS     int `json:"requestTimeoutMs"`
		PerUserRateLimit     int `json:"perUserRateLimit"`
		SuggestionCacheHours int `json:"suggestionCacheHours"`
		DefaultDailyQuota    int `json:"defaultDailyQuota"`
		MaxImageResults      int `json:"maxImageResults"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN04001, "Enter valid queue settings.", "parse queue config", err)
		return
	}
	config, err := h.admin.UpdateQueueConfig(request.MaxConcurrent, request.MaxQueueSize, request.RequestTimeoutMS, request.PerUserRateLimit, request.SuggestionCacheHours, request.DefaultDailyQuota, request.MaxImageResults)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN04001, "That queue configuration is not valid.", "queue config invalid", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(config))
}

func (h *AdminHandler) SiteSettings(c *gin.Context) {
	settings, err := h.admin.SiteSettings()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN05001, "Site settings could not be loaded.", "load site settings", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(settings))
}

func (h *AdminHandler) UpdateSiteSettings(c *gin.Context) {
	var request struct {
		SiteName  string  `json:"siteName"`
		Tagline   *string `json:"tagline"`
		Copyright *string `json:"copyright"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN05001, "Enter a valid site name.", "parse site settings", err)
		return
	}
	settings, err := h.admin.UpdateSiteSettings(request.SiteName, request.Tagline, request.Copyright)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN05001, "That site configuration is not valid.", "site settings invalid", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(settings))
}

func (h *AdminHandler) UploadLogo(c *gin.Context) {
	header, err := c.FormFile("logo")
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN05002, "Choose a logo image to upload.", "missing logo file", err)
		return
	}
	data, err := readUpload(header)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN05002, "That logo could not be read.", "unreadable logo file", err)
		return
	}
	url, err := h.admin.Logo(header.Filename, data)
	if err != nil {
		if errors.Is(err, services.ErrUploadRejected) {
			middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN05002, "The logo must be a JPG, PNG, GIF, or WebP under 2 MB.", "logo upload rejected", err)
			return
		}
		// ADMN05002 with the cause in the logs — upload failures are almost
		// always a missing/unwritable UPLOADS_DIR in the container.
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN05002, "That logo could not be saved.", "site logo upload failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"logoUrl": url}))
}

func (h *AdminHandler) DeleteLogo(c *gin.Context) {
	if err := h.admin.RemoveLogo(); err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN05002, "The logo could not be removed.", "remove logo failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"logoUrl": nil}))
}

func (h *AdminHandler) UploadFavicon(c *gin.Context) {
	header, err := c.FormFile("favicon")
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN05003, "Choose a favicon image to upload.", "missing favicon file", err)
		return
	}
	data, err := readUpload(header)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN05003, "That favicon could not be read.", "unreadable favicon file", err)
		return
	}
	url, err := h.admin.Favicon(header.Filename, data)
	if err != nil {
		if errors.Is(err, services.ErrUploadRejected) {
			middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN05003, "The favicon must be a JPG, PNG, GIF, or WebP under 2 MB.", "favicon upload rejected", err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN05003, "That favicon could not be saved.", "site favicon upload failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"faviconUrl": url}))
}

func (h *AdminHandler) DeleteFavicon(c *gin.Context) {
	if err := h.admin.RemoveFavicon(); err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.ADMN05003, "The favicon could not be removed.", "remove favicon failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"faviconUrl": nil}))
}

func strPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func readUpload(header *multipart.FileHeader) ([]byte, error) {
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	const maxRead = 2<<20 + 1
	data, err := io.ReadAll(io.LimitReader(file, maxRead))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func queryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return fallback
	}
	return value
}

func userIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, contracts.USER01001, "That user could not be found.", "parse user id", err)
		return uuid.Nil, false
	}
	return id, true
}

func providerIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, contracts.ADMN03001, "That provider could not be found.", "parse provider id", err)
		return uuid.Nil, false
	}
	return id, true
}