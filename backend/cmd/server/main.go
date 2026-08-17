package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"intellisearch/internal/cache"
	"intellisearch/internal/config"
	"intellisearch/internal/database"
	"intellisearch/internal/handlers"
	"intellisearch/internal/repositories"
	"intellisearch/internal/router"
	"intellisearch/internal/services"
)

func main() {
	cfg := config.Load()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.WithField("environment", cfg.AppEnv).Info("starting API")
	db, err := database.Connect(cfg)
	if err != nil {
		logrus.WithError(err).Error("database connection failed")
		os.Exit(1)
	}
	if err := database.MigrateAndSeed(db, cfg); err != nil {
		logrus.WithError(err).Error("database migration failed")
		os.Exit(1)
	}

	rdb, err := cache.NewRedis(cfg.RedisAddr, cfg.RedisPassword)
	var limiter services.Limiter
	if err != nil {
		logrus.WithError(err).Error("redis unavailable; rate limiting disabled")
		limiter = services.NoopLimiter{}
	} else {
		limiter = services.NewRedisLimiter(rdb)
	}

	siteService := services.NewSiteService(repositories.NewSiteRepository(db))
	userRepository := repositories.NewUserRepository(db)
	authService := services.NewAuthService(userRepository, cfg)
	usageLogRepository := repositories.NewUsageLogRepository(db)
	searchHistoryRepository := repositories.NewSearchHistoryRepository(db)
	userService := services.NewUserService(userRepository, usageLogRepository, cfg.UploadsDir)
	sessionRepository := repositories.NewSessionRepository(db)
	messageRepository := repositories.NewMessageRepository(db)
	providerRepository := repositories.NewProviderRepository(db)
	queueConfigRepository := repositories.NewQueueConfigRepository(db)
	crawlJobRepository := repositories.NewCrawlJobRepository(db)

	searchService := services.NewSearchService(cfg)
	crawlService := services.NewCrawlService(cfg.CrawlerBaseURL, cfg.CrawlerTimeoutMS, crawlJobRepository)
	llmService := services.NewLLMService(providerRepository, cfg.EncryptionKey)
	geoService := services.NewGeoService(cfg.NominatimBaseURL, "Intellisearch/1.0", cfg.NominatimTimeoutMS)
	aiService := services.NewAIService(sessionRepository, messageRepository, usageLogRepository, providerRepository, userRepository, searchHistoryRepository, searchService, crawlService, llmService, geoService, cfg.CrawlTopN)
	aiHandler := handlers.NewAIHandler(aiService, queueConfigRepository, userRepository, usageLogRepository, limiter, authService)
	adminService := services.NewAdminService(providerRepository, queueConfigRepository, repositories.NewSiteRepository(db), cfg.EncryptionKey, cfg.UploadsDir)
	statsService := services.NewStatsService(usageLogRepository, userRepository, providerRepository, aiHandler)
	adminHandler := handlers.NewAdminHandler(userService, adminService, statsService)
	sessionHandler := handlers.NewSessionHandler(sessionRepository, messageRepository, authService)
	historyService := services.NewSearchHistoryService(searchHistoryRepository, llmService)

	api := router.New(cfg.CORSOrigins, cfg.UploadsDir, siteService, handlers.NewAuthHandler(authService), handlers.NewUserHandler(userService, historyService), sessionHandler, aiHandler, adminHandler, authService)
	if err := api.Run(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		logrus.WithError(err).Error("API stopped")
		os.Exit(1)
	}
}