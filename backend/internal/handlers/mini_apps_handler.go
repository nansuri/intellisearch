package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/services"
)

// MiniAppsHandler is the specialized HTTP surface for user-created mini apps:
// per-user CRUD, AI generation (through the single AI handler, so it reuses the
// worker pool, queue, rate limits, and daily quota), the public app launcher,
// and the DB-stored API reference that powers the Studio's API list and the
// downloadable AI-API markdown export.
type MiniAppsHandler struct {
	apps *services.MiniAppService
	docs *services.ApiDocsService
	ai   *AIHandler
	auth *services.AuthService
}

func NewMiniAppsHandler(apps *services.MiniAppService, docs *services.ApiDocsService, ai *AIHandler, auth *services.AuthService) *MiniAppsHandler {
	return &MiniAppsHandler{apps: apps, docs: docs, ai: ai, auth: auth}
}

// meID returns the authenticated caller's user id from the required-auth
// middleware context.
func meID(c *gin.Context) uuid.UUID {
	return c.MustGet(middleware.UserIDKey).(uuid.UUID)
}

func (h *MiniAppsHandler) ListMine(c *gin.Context) {
	apps, err := h.apps.List(meID(c))
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.MINI01001, "Your mini apps could not be loaded.", "list mini apps", err)
		return
	}
	if apps == nil {
		apps = []entities.MiniApp{}
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"items": apps}))
}

func (h *MiniAppsHandler) Create(c *gin.Context) {
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		HTML        string `json:"html"`
		CSS         string `json:"css"`
		JS          string `json:"js"`
		Visibility  string `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.MINI01002, "Enter a name and the app's HTML to create it.", "parse create mini app", err)
		return
	}
	app, err := h.apps.Create(meID(c), services.MiniAppInput{Name: request.Name, Description: request.Description, Icon: request.Icon, HTML: request.HTML, CSS: request.CSS, JS: request.JS, Visibility: request.Visibility})
	if err != nil {
		h.respondAppError(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(app))
}

func (h *MiniAppsHandler) Get(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	app, err := h.apps.Get(meID(c), id)
	if err != nil {
		h.respondAppError(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(app))
}

func (h *MiniAppsHandler) Update(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	var request struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Icon        *string `json:"icon"`
		HTML        *string `json:"html"`
		CSS         *string `json:"css"`
		JS          *string `json:"js"`
		Visibility  *string `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.MINI01002, "That update could not be parsed.", "parse update mini app", err)
		return
	}
	patch := services.MiniAppPatch{Name: request.Name, Description: request.Description, Icon: request.Icon, HTML: request.HTML, CSS: request.CSS, JS: request.JS, Visibility: request.Visibility}
	app, err := h.apps.Update(meID(c), id, patch)
	if err != nil {
		h.respondAppError(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(app))
}

func (h *MiniAppsHandler) Delete(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	if err := h.apps.Delete(meID(c), id); err != nil {
		h.respondAppError(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"deleted": true}))
}

// Generate creates a mini app from a single prompt using the AI pipeline. It is
// routed through the AI handler, so it inherits the worker pool/queue, rate
// limiting, and the signed-in user's daily question quota. The generated draft
// is persisted immediately (as a private draft) so the Studio can open it in
// the IDE, refine it, and publish it.
func (h *MiniAppsHandler) Generate(c *gin.Context) {
	var request struct {
		Prompt string `json:"prompt"`
	}
	userID := meID(c)
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Prompt) == "" {
		middleware.RespondError(c, http.StatusBadRequest, contracts.MINI01002, "Describe the mini app you want the AI to build.", "parse generate mini app", err)
		return
	}
	draft, err := h.ai.GenerateMiniApp(c.Request.Context(), userID, request.Prompt)
	if err != nil {
		code, status := h.ai.errorResponse(err)
		middleware.RespondError(c, status, code, services.SanitizedErrorMessage(err), "mini app generation failed", err)
		return
	}
	app, err := h.apps.CreateDraft(userID, draft)
	if err != nil {
		h.respondAppError(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(app))
}

// PublicList serves the app drawer/gallery: every active public app as a
// source-free summary.
func (h *MiniAppsHandler) PublicList(c *gin.Context) {
	apps, err := h.apps.PublicList()
	if err != nil {
		logrus.WithError(err).Error("public mini apps list failed")
		middleware.RespondError(c, http.StatusInternalServerError, contracts.MINI01001, "Mini apps could not be loaded.", "public mini apps list", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"items": apps}))
}

// PublicGet serves one app by slug for the runner. Public apps are open;
// private apps require the owning user (optional valid session).
func (h *MiniAppsHandler) PublicGet(c *gin.Context) {
	slug := c.Param("slug")
	userID := h.optionalUser(c)
	app, err := h.apps.GetForRun(slug, userID)
	if err != nil {
		h.respondAppError(c, err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(app))
}

// ApiDocs returns the Mini Apps platform API reference grouped by section.
func (h *MiniAppsHandler) ApiDocs(c *gin.Context) {
	sections, err := h.docs.Groups()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.MINI03002, "The API documentation could not be loaded.", "load api docs", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"sections": sections}))
}

// ApiDocMarkdown serves the full API reference (including the AI ask endpoint)
// as a markdown document for download — the "AI API as markdown" export.
func (h *MiniAppsHandler) ApiDocMarkdown(c *gin.Context) {
	markdown, err := h.docs.Markdown()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.MINI03002, "The API documentation could not be loaded.", "export api docs", err)
		return
	}
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="mini-apps-api.md"`)
	c.String(http.StatusOK, markdown)
}

// optionalUser resolves the caller when a Bearer token is present; anonymous
// callers (no header) return nil. An invalid token is a hard 401 (it would be
// an error to treat a bad token as anonymous).
func (h *MiniAppsHandler) optionalUser(c *gin.Context) *uuid.UUID {
	raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if raw == c.GetHeader("Authorization") || raw == "" {
		return nil
	}
	claims, err := h.auth.Parse(raw)
	if err != nil {
		middleware.RespondError(c, http.StatusUnauthorized, contracts.AUTH01002, "Your session is invalid or has expired.", "unauthorized mini-app caller", err)
		return nil
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		middleware.RespondError(c, http.StatusUnauthorized, contracts.AUTH01002, "Your session is invalid or has expired.", "unauthorized mini-app caller", err)
		return nil
	}
	return &id
}

func (h *MiniAppsHandler) parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.MINI01003, "That mini app could not be found.", "parse mini app id", err)
		return uuid.Nil, false
	}
	return id, true
}

func (h *MiniAppsHandler) respondAppError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrMiniAppNotFound):
		middleware.RespondError(c, http.StatusNotFound, contracts.MINI01003, "That mini app no longer exists.", "mini app not found", err)
	case errors.Is(err, services.ErrMiniAppPrivate):
		middleware.RespondError(c, http.StatusForbidden, contracts.MINI03001, "That mini app is private.", "mini app private", err)
	case errors.Is(err, services.ErrMiniAppInvalid):
		middleware.RespondError(c, http.StatusBadRequest, contracts.MINI01002, "That mini app is not valid — check the name and the size of your HTML/CSS/JS.", "invalid mini app", err)
	case errors.Is(err, services.ErrMiniAppSlugTaken):
		middleware.RespondError(c, http.StatusConflict, contracts.MINI01005, "That app name is already taken — try a different one.", "mini app slug taken", err)
	default:
		middleware.RespondError(c, http.StatusInternalServerError, contracts.MINI01001, "That mini app could not be saved.", "mini app store failed", err)
	}
}