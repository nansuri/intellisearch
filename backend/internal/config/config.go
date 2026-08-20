package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port, AppEnv, CORSOrigins, JWTSecret, SuperOwnerEmail, SuperOwnerPassword string
	DBHost, DBPort, DBUser, DBPassword, DBName, DBSSLMode                     string
	DBDriver, DBSQLitePath                                                    string
	RedisAddr, RedisPassword, EncryptionKey                                   string
	AIProvider, OllamaBaseURL, AIModel, OpenAIBaseURL, OpenAIAPIKey           string
	SearXNGBaseURL                                                            string
	SearXNGTimeoutMS                                                          int
	NominatimBaseURL                                                          string
	NominatimTimeoutMS                                                        int
	CrawlerBaseURL                                                            string
	CrawlerTimeoutMS, CrawlTopN                                               int
	JWTTTLHours                                                               int
	UploadsDir                                                                string
	GoogleClientID, GoogleClientSecret, GoogleRedirectURL, FrontendOrigin     string
	TrustedProxies                                                            []string
	LibreTranslateBaseURL                                                     string
	PollinationsMediaBaseURL                                                  string
}

func Load() Config {
	return Config{
		Port: env("PORT", "8088"), AppEnv: env("APP_ENV", "development"),
		CORSOrigins: env("CORS_ORIGINS", "http://localhost:5173"),
		JWTSecret:   env("JWT_SECRET", "change-me-32-chars-min"),
		// JWT_TTL_HOURS is the fallback session lifetime. The Owner Control
		// Panel's admin-editable ai_queue_config.session_ttl_hours overrides it
		// when set (see auth_service.sessionTTL); the default is 7 days.
		JWTTTLHours:     envInt("JWT_TTL_HOURS", 168),
		SuperOwnerEmail: os.Getenv("SUPER_OWNER_EMAIL"), SuperOwnerPassword: os.Getenv("SUPER_OWNER_PASSWORD"),
		DBHost: env("DB_HOST", "localhost"), DBPort: env("DB_PORT", "5432"), DBUser: env("DB_USER", "aimain"),
		DBPassword: env("DB_PASSWORD", "aimain"), DBName: env("DB_NAME", "aimain"), DBSSLMode: env("DB_SSLMODE", "disable"),
		DBDriver: env("DB_DRIVER", defaultDBDriver()), DBSQLitePath: env("DB_SQLITE_PATH", "./data/intellisearch.db"),
		RedisAddr: env("REDIS_ADDR", "localhost:6379"), RedisPassword: env("REDIS_PASSWORD", ""),
		EncryptionKey: env("ENCRYPTION_KEY", "change-me-32-byte-key-for-aes-gcm"),
		AIProvider:    env("AI_PROVIDER", "ollama"), OllamaBaseURL: env("OLLAMA_BASE_URL", "http://localhost:11434"),
		AIModel: env("AI_MODEL", "llama3.2"), OpenAIBaseURL: env("OPENAI_BASE_URL", ""), OpenAIAPIKey: env("OPENAI_API_KEY", ""),
		SearXNGBaseURL: env("SEARXNG_BASE_URL", "http://localhost:8081"), SearXNGTimeoutMS: envInt("SEARXNG_TIMEOUT_MS", 10000),
		NominatimBaseURL: env("NOMINATIM_BASE_URL", "https://nominatim.openstreetmap.org"), NominatimTimeoutMS: envInt("NOMINATIM_TIMEOUT_MS", 5000),
		CrawlerBaseURL: env("CRAWLER_BASE_URL", "http://localhost:3002"), CrawlerTimeoutMS: envInt("CRAWLER_TIMEOUT_MS", 15000),
		CrawlTopN:  envInt("CRAWL_TOP_N", 3),
		UploadsDir: env("UPLOADS_DIR", "./uploads"),
		// LibreTranslate is an internal-only container (attached via the
		// production compose network); the API proxies it for the translator app.
		LibreTranslateBaseURL: env("LIBRETRANSLATE_BASE_URL", "http://libretranslate:5000"),
		// Pollinations media uploads go to media.pollinations.ai (the account
		// API lives on the provider's stored base URL, e.g. gen.pollinations.ai).
		PollinationsMediaBaseURL: env("POLLINATIONS_MEDIA_BASE_URL", "https://media.pollinations.ai"),
		GoogleClientID:           os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:       os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:        env("GOOGLE_REDIRECT_URL", "http://localhost:5173/api/v1/auth/google/callback"),
		FrontendOrigin:           env("FRONTEND_ORIGIN", "http://localhost:5173"),
		// Proxies whose X-Forwarded-For is trusted (dev Vite proxy, Docker
		// nginx, LAN). Only these may supply the client IP used for the
		// anonymous per-IP allowance — external clients cannot spoof it.
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES"), "127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"),
	}
}

// splitList splits a comma-separated env value into a trimmed non-empty list.
func splitList(raw, fallback string) []string {
	if raw == "" {
		raw = fallback
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func defaultDBDriver() string {
	if os.Getenv("APP_ENV") == "production" {
		return "postgres"
	}
	return "sqlite"
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
