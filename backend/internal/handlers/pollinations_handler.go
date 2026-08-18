package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/services"
)

// PollinationsHandler is the specialized handler for Pollinations.ai account
// operations (balance/credits, usage, model list, media upload). It sits
// alongside the AI handler — the AI handler remains the single interface for
// AI work (ask/url/queue), while this handler owns the provider-account
// introspection the Owner Control Panel needs. All routes are Super Owner only,
// and the browser never calls Pollinations directly.
type PollinationsHandler struct {
	admin        *services.AdminService
	pollinations *services.PollinationsService
}

func NewPollinationsHandler(admin *services.AdminService, pollinations *services.PollinationsService) *PollinationsHandler {
	return &PollinationsHandler{admin: admin, pollinations: pollinations}
}

// resolveCredentials determines the base URL + API key from the request:
//   - providerId  → load the stored provider and decrypt its key server-side
//     (preferred; the key never leaves the database);
//   - apiKey + baseUrl → use the credentials directly (for a provider being
//     configured but not yet saved — same as the Ollama baseUrl flow).
func (h *PollinationsHandler) resolveCredentials(c *gin.Context) (baseURL, apiKey string, ok bool) {
	var request struct {
		ProviderID *uuid.UUID `json:"providerId"`
		APIKey     string     `json:"apiKey"`
		BaseURL    string     `json:"baseUrl"`
	}
	_ = c.ShouldBindJSON(&request)
	if request.ProviderID != nil {
		baseURL, apiKey, err := h.admin.PollinationsCredentials(*request.ProviderID)
		if err != nil {
			middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN03001, "That Pollinations provider could not be found.", "resolve pollinations provider", err)
			return "", "", false
		}
		return baseURL, apiKey, true
	}
	if request.APIKey == "" {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN07002, "Enter the Pollinations API key, or pick a saved Pollinations provider.", "missing pollinations key", nil)
		return "", "", false
	}
	baseURL = request.BaseURL
	if baseURL == "" {
		baseURL = "https://gen.pollinations.ai"
	}
	return baseURL, request.APIKey, true
}

// respond maps Pollinations service errors to the ADMN07 error catalog.
func (h *PollinationsHandler) respond(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrPollinationsUnauthorized):
		middleware.RespondError(c, http.StatusBadGateway, contracts.ADMN07002, "That Pollinations API key was rejected — check it's valid and has account access.", "pollinations key rejected", err)
	case errors.Is(err, services.ErrPollinationsForbidden):
		middleware.RespondError(c, http.StatusBadGateway, contracts.ADMN07004, "That Pollinations API key is valid but lacks account access — create one with the account:usage / account:balance scopes on enter.pollinations.ai.", "pollinations key missing account scope", err)
	case errors.Is(err, services.ErrPollinationsUploadFailed):
		middleware.RespondError(c, http.StatusBadGateway, contracts.ADMN07003, "The image could not be uploaded to Pollinations.", "pollinations upload failed", err)
	case errors.Is(err, services.ErrPollinationsPaymentRequired):
		middleware.RespondError(c, http.StatusPaymentRequired, contracts.ADMN07005, "Pollinations balance or API-key budget is exhausted — top up or raise the key budget at enter.pollinations.ai.", "pollinations balance exhausted", err)
	case errors.Is(err, services.ErrPollinationsRateLimited):
		middleware.RespondError(c, http.StatusTooManyRequests, contracts.ADMN07006, "Pollinations is rate-limiting requests — wait a moment and try again.", "pollinations rate limited", err)
	default:
		middleware.RespondError(c, http.StatusBadGateway, contracts.ADMN07001, "Couldn't reach the Pollinations account API — try again in a moment.", "pollinations account request failed", err)
	}
}

// Account returns the balance, profile, and key info for the Pollinations key.
func (h *PollinationsHandler) Account(c *gin.Context) {
	baseURL, apiKey, ok := h.resolveCredentials(c)
	if !ok {
		return
	}
	balance, profile, key, err := h.pollinations.Account(c.Request.Context(), baseURL, apiKey)
	if err != nil {
		h.respond(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"ok": true, "balance": balance, "profile": profile, "key": key}))
}

// Usage returns per-request usage history (last `days`, default 30, max 90).
func (h *PollinationsHandler) Usage(c *gin.Context) {
	baseURL, apiKey, ok := h.resolveCredentials(c)
	if !ok {
		return
	}
	days := queryInt(c, "days", 30)
	usage, err := h.pollinations.Usage(c.Request.Context(), baseURL, apiKey, days)
	if err != nil {
		h.respond(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"usage": usage, "count": len(usage)}))
}

// DailyUsage returns per-day aggregated usage (last `days`).
func (h *PollinationsHandler) DailyUsage(c *gin.Context) {
	baseURL, apiKey, ok := h.resolveCredentials(c)
	if !ok {
		return
	}
	days := queryInt(c, "days", 30)
	usage, err := h.pollinations.DailyUsage(c.Request.Context(), baseURL, apiKey, days)
	if err != nil {
		h.respond(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"usage": usage, "count": len(usage)}))
}

// Models lists the models available to the Pollinations key (fed to the
// provider form's model dropdown, like the Ollama model picker).
func (h *PollinationsHandler) Models(c *gin.Context) {
	baseURL, apiKey, ok := h.resolveCredentials(c)
	if !ok {
		return
	}
	models, err := h.pollinations.Models(c.Request.Context(), baseURL, apiKey)
	if err != nil {
		h.respond(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"models": models}))
}

// Upload forwards an image to the Pollinations media API and returns the
// public URL. The file is received by the Go API (browser → Go → Pollinations).
func (h *PollinationsHandler) Upload(c *gin.Context) {
	// The multipart form carries the credentials as fields plus the file.
	providerID, _ := uuid.Parse(c.PostForm("providerId"))
	apiKey := c.PostForm("apiKey")
	baseURL := c.PostForm("baseUrl")
	if providerID == uuid.Nil && apiKey == "" {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN07002, "Enter the Pollinations API key, or pick a saved Pollinations provider.", "missing pollinations key", nil)
		return
	}
	if providerID != uuid.Nil {
		var err error
		baseURL, apiKey, err = h.admin.PollinationsCredentials(providerID)
		if err != nil {
			middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN03001, "That Pollinations provider could not be found.", "resolve pollinations provider", err)
			return
		}
	}
	if baseURL == "" {
		baseURL = "https://gen.pollinations.ai"
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN07003, "Attach an image file to upload.", "missing upload file", err)
		return
	}
	defer file.Close()
	data := make([]byte, header.Size)
	if _, err := file.Read(data); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.ADMN07003, "That image could not be read.", "unreadable upload file", err)
		return
	}
	result, err := h.pollinations.Upload(c.Request.Context(), apiKey, header.Filename, header.Header.Get("Content-Type"), data)
	if err != nil {
		h.respond(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(result))
}
