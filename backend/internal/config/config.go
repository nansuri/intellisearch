package config

import (
	"os"
	"strconv"
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
	GoogleClientID, GoogleClientSecret, GoogleRedirectURL, FrontendOrigin      string
}

func Load() Config {
	return Config{
		Port: env("PORT", "8080"), AppEnv: env("APP_ENV", "development"),
		CORSOrigins:     env("CORS_ORIGINS", "http://localhost:5173"),
		JWTSecret:       env("JWT_SECRET", "change-me-32-chars-min"),
		JWTTTLHours:     envInt("JWT_TTL_HOURS", 24),
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
		CrawlerBaseURL: env("CRAWLER_BASE_URL", "http://localhost:3000"), CrawlerTimeoutMS: envInt("CRAWLER_TIMEOUT_MS", 15000),
		CrawlTopN: envInt("CRAWL_TOP_N", 3),
		UploadsDir: env("UPLOADS_DIR", "./uploads"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  env("GOOGLE_REDIRECT_URL", "http://localhost:5173/api/v1/auth/google/callback"),
		FrontendOrigin:     env("FRONTEND_ORIGIN", "http://localhost:5173"),
	}
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
