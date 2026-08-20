package router

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"intellisearch/internal/contracts"
	"intellisearch/internal/handlers"
	"intellisearch/internal/middleware"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/services"
)

func New(corsOrigins, uploadsDir string, siteService *services.SiteService, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, sessionHandler *handlers.SessionHandler, aiHandler *handlers.AIHandler, adminHandler *handlers.AdminHandler, appsHandler *handlers.AppsHandler, pollinationsHandler *handlers.PollinationsHandler, visitorHandler *handlers.VisitorHandler, miniAppsHandler *handlers.MiniAppsHandler, authService *services.AuthService) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.ErrorHandler())
	r.Use(cors(corsOrigins))
	r.GET("/health", func(c *gin.Context) { middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"status": "ok"})) })
	// The PWA web manifest lives at the frontend origin and is rendered here so
	// it always carries the live site_settings branding (site name, tagline,
	// uploaded logo) — an installed app matches the Owner Control Panel. It is
	// NOT an API endpoint: raw JSON, no envelope, no-cache so branding edits
	// propagate immediately.
	r.GET("/manifest.webmanifest", func(c *gin.Context) {
		site, err := siteService.Public()
		if err != nil {
			logrus.WithError(err).Warn("manifest served with default branding; site settings unavailable")
			site = entities.SiteSettings{SiteName: "Intellisearch"}
		}
		data, err := json.Marshal(siteManifest(site))
		if err != nil {
			c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", []byte(`{"error":"manifest unavailable"}`))
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "application/manifest+json; charset=utf-8", data)
	})
	if uploadsDir != "" {
		r.Static("/uploads", uploadsDir)
	}
	v1 := r.Group("/api/v1")
	v1.GET("/site", func(c *gin.Context) {
		site, err := siteService.Public()
		if err != nil {
			middleware.RespondError(c, http.StatusServiceUnavailable, contracts.SITE01001, "Site settings are temporarily unavailable.", "load public site settings", err)
			return
		}
		middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{
			"siteName":         site.SiteName,
			"logoUrl":          site.LogoURL,
			"faviconUrl":       site.FaviconURL,
			"tagline":          site.Tagline,
			"copyright":        site.Copyright,
			"googleSsoEnabled": authService.GoogleConfigured(),
		}))
	})
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/register", authHandler.Register)
	v1.GET("/auth/google", authHandler.GoogleStart)
	v1.GET("/auth/google/callback", authHandler.GoogleCallback)
	v1.POST("/ask", aiHandler.Ask)
	v1.POST("/ask/url", aiHandler.AskURL)
	v1.POST("/stats/register-visit", visitorHandler.TrackRegisterVisit)
	v1.GET("/sessions/:id", sessionHandler.Get)
	v1.POST("/sessions/:id/suggestions", aiHandler.SessionSuggestions)
	// Mini apps: the shared app drawer/gallery and runner are public; the CRUD,
	// AI generation, and Studio surface live under /me (authenticated). Static
	// doc routes must be registered before the :slug param for routing safety.
	v1.GET("/mini-apps", miniAppsHandler.PublicList)
	v1.GET("/mini-apps/api-docs", miniAppsHandler.ApiDocs)
	v1.GET("/mini-apps/api-docs/ai.md", miniAppsHandler.ApiDocMarkdown)
	v1.GET("/mini-apps/api-docs/:file", miniAppsHandler.ApiDocMarkdown)
	v1.GET("/mini-apps/:slug", miniAppsHandler.PublicGet)
	protected := v1.Group("")
	protected.Use(middleware.RequireAuth(authService))
	protected.GET("/me", userHandler.Me)
	protected.PATCH("/me", userHandler.UpdateMe)
	protected.POST("/me/avatar", userHandler.Avatar)
	protected.GET("/me/history", userHandler.History)
	protected.GET("/me/history/suggestions", userHandler.Suggestions)
	protected.DELETE("/me/history", userHandler.ClearHistory)
	protected.GET("/me/mini-apps", miniAppsHandler.ListMine)
	protected.POST("/me/mini-apps", miniAppsHandler.Create)
	protected.POST("/me/mini-apps/generate", miniAppsHandler.Generate)
	protected.GET("/me/mini-apps/:id", miniAppsHandler.Get)
	protected.PATCH("/me/mini-apps/:id", miniAppsHandler.Update)
	protected.DELETE("/me/mini-apps/:id", miniAppsHandler.Delete)
	// Mini apps: notes (personal, incl. save-summary) and translator (proxied
	// server-side to the LibreTranslate container).
	protected.GET("/me/notes", appsHandler.ListNotes)
	protected.POST("/me/notes", appsHandler.CreateNote)
	protected.PATCH("/me/notes/:id", appsHandler.UpdateNote)
	protected.DELETE("/me/notes/:id", appsHandler.DeleteNote)
	protected.GET("/translate/languages", appsHandler.TranslateLanguages)
	protected.POST("/translate", appsHandler.Translate)
	protected.POST("/auth/logout", middleware.RequireSuperOwner(), authHandler.Logout)

	admin := protected.Group("/admin")
	admin.Use(middleware.RequireSuperOwner())
	admin.GET("/users", adminHandler.ListUsers)
	admin.POST("/users", adminHandler.CreateUser)
	admin.PATCH("/users/:id", adminHandler.UpdateUser)
	admin.DELETE("/users/:id", adminHandler.DeleteUser)
	admin.GET("/stats", adminHandler.Stats)
	admin.GET("/stats/trends", adminHandler.Trends)
	admin.GET("/stats/trending-words", adminHandler.TrendingWords)
	admin.GET("/stats/visitors", adminHandler.Visitors)
	admin.GET("/stats/ai", adminHandler.AIStats)
	admin.GET("/ai/providers", adminHandler.ListProviders)
	admin.POST("/ai/providers", adminHandler.CreateProvider)
	admin.GET("/ai/ollama/models", adminHandler.OllamaModels)
	admin.GET("/ai/ollama/health", adminHandler.OllamaHealth)
	// Pollinations account introspection (specialized handler, still under the
	// AI admin surface — the AI handler remains the single interface for AI).
	admin.POST("/ai/pollinations/account", pollinationsHandler.Account)
	admin.POST("/ai/pollinations/usage", pollinationsHandler.Usage)
	admin.POST("/ai/pollinations/usage/daily", pollinationsHandler.DailyUsage)
	admin.POST("/ai/pollinations/models", pollinationsHandler.Models)
	admin.POST("/ai/pollinations/upload", pollinationsHandler.Upload)
	admin.PATCH("/ai/providers/:id", adminHandler.UpdateProvider)
	admin.DELETE("/ai/providers/:id", adminHandler.DeleteProvider)
	admin.GET("/ai/queue-config", adminHandler.QueueConfig)
	admin.PATCH("/ai/queue-config", adminHandler.UpdateQueueConfig)
	admin.GET("/site-settings", adminHandler.SiteSettings)
	admin.PATCH("/site-settings", adminHandler.UpdateSiteSettings)
	admin.POST("/site-settings/logo", adminHandler.UploadLogo)
	admin.DELETE("/site-settings/logo", adminHandler.DeleteLogo)
	admin.POST("/site-settings/favicon", adminHandler.UploadFavicon)
	admin.DELETE("/site-settings/favicon", adminHandler.DeleteFavicon)
	return r
}

// siteManifest builds the PWA manifest JSON from the live site settings. The
// static icon files ship in the frontend build (frontend/scripts/gen-pwa-icons.mjs)
// and are referenced relative to the frontend origin; an uploaded logo is
// appended as an additional high-resolution icon when set.
func siteManifest(site entities.SiteSettings) map[string]any {
	name := site.SiteName
	if name == "" {
		name = "Intellisearch"
	}
	short := name
	if runes := []rune(short); len(runes) > 12 {
		short = string(runes[:12])
	}
	description := ""
	if site.Tagline != nil {
		description = *site.Tagline
	}
	icons := []map[string]any{
		{"src": "/pwa-192x192.png", "sizes": "192x192", "type": "image/png", "purpose": "any"},
		{"src": "/pwa-512x512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"},
		{"src": "/pwa-maskable-512x512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
	}
	if site.LogoURL != nil && strings.TrimSpace(*site.LogoURL) != "" {
		icons = append(icons, map[string]any{"src": *site.LogoURL, "purpose": "any"})
	}
	return map[string]any{
		"id":               "/",
		"name":             name,
		"short_name":       short,
		"description":      description,
		"start_url":        "/",
		"scope":            "/",
		"display":          "standalone",
		"background_color": "#eef2f9",
		"theme_color":      "#4f6ef7",
		"icons":            icons,
	}
}

func cors(origins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origins)
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
