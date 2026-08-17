package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"intellisearch/internal/contracts"
	"intellisearch/internal/handlers"
	"intellisearch/internal/middleware"
	"intellisearch/internal/services"
)

func New(corsOrigins, uploadsDir string, siteService *services.SiteService, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, sessionHandler *handlers.SessionHandler, aiHandler *handlers.AIHandler, adminHandler *handlers.AdminHandler, authService *services.AuthService) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.ErrorHandler())
	r.Use(cors(corsOrigins))
	r.GET("/health", func(c *gin.Context) { middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"status": "ok"})) })
	if uploadsDir != "" {
		r.Static("/uploads", uploadsDir)
	}
	v1 := r.Group("/api/v1")
	v1.GET("/site", func(c *gin.Context) {
		site, err := siteService.Public()
		if err != nil {
			middleware.JSON(c, http.StatusServiceUnavailable, contracts.Fail(contracts.SITE01001, "Site settings are temporarily unavailable."))
			return
		}
		middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{
			"siteName":         site.SiteName,
			"logoUrl":          site.LogoURL,
			"faviconUrl":       site.FaviconURL,
			"tagline":          site.Tagline,
			"googleSsoEnabled": authService.GoogleConfigured(),
		}))
	})
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/register", authHandler.Register)
	v1.GET("/auth/google", authHandler.GoogleStart)
	v1.GET("/auth/google/callback", authHandler.GoogleCallback)
	v1.POST("/ask", aiHandler.Ask)
	v1.POST("/ask/url", aiHandler.AskURL)
	v1.GET("/sessions/:id", sessionHandler.Get)
	protected := v1.Group("")
	protected.Use(middleware.RequireAuth(authService))
	protected.GET("/me", userHandler.Me)
	protected.PATCH("/me", userHandler.UpdateMe)
	protected.POST("/me/avatar", userHandler.Avatar)
	protected.GET("/me/history", userHandler.History)
	protected.GET("/me/history/suggestions", userHandler.Suggestions)
	protected.DELETE("/me/history", userHandler.ClearHistory)
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
	admin.GET("/stats/ai", adminHandler.AIStats)
	admin.GET("/ai/providers", adminHandler.ListProviders)
	admin.POST("/ai/providers", adminHandler.CreateProvider)
	admin.GET("/ai/ollama/models", adminHandler.OllamaModels)
	admin.GET("/ai/ollama/health", adminHandler.OllamaHealth)
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