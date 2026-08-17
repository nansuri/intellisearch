package entities

import (
	"github.com/google/uuid"
	"time"
)

const (
	RoleGeneralUser = "general_user"
	RoleSuperOwner  = "super_owner"
	StatusActive    = "active"
	StatusSuspended = "suspended"

	MessageRoleSystem    = "system"
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"

	MessageStatusQueued    = "queued"
	MessageStatusStreaming = "streaming"
	MessageStatusCompleted = "completed"
	MessageStatusFailed    = "failed"

	CrawlStatusQueued    = "queued"
	CrawlStatusRunning   = "running"
	CrawlStatusCompleted = "completed"
	CrawlStatusFailed    = "failed"
	CrawlStatusBlocked   = "blocked"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string     `gorm:"not null" json:"name"`
	Email        string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"not null" json:"-"`
	Role         string     `gorm:"not null;default:general_user" json:"role"`
	Status       string     `gorm:"not null;default:active" json:"status"`
	AvatarURL    *string    `json:"avatarUrl"`
	AIDailyQuota int        `gorm:"not null;default:0" json:"aiDailyQuota"`
	LastLoginAt  *time.Time `json:"lastLoginAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type SiteSettings struct {
	ID          uint      `gorm:"primaryKey" json:"-"`
	SiteName    string    `gorm:"not null" json:"siteName"`
	LogoURL     *string   `json:"logoUrl"`
	FaviconURL  *string   `json:"faviconUrl"`
	Tagline     *string   `json:"tagline"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AIQueueConfig struct {
	ID               uint      `gorm:"primaryKey" json:"-"`
	MaxConcurrent    int       `gorm:"not null" json:"maxConcurrent"`
	MaxQueueSize     int       `gorm:"not null" json:"maxQueueSize"`
	RequestTimeoutMS int       `gorm:"not null" json:"requestTimeoutMs"`
	PerUserRateLimit int       `gorm:"not null" json:"perUserRateLimit"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type AIProvider struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name            string    `gorm:"not null;uniqueIndex" json:"name"`
	ProviderType    string    `gorm:"not null" json:"providerType"`
	BaseURL         string    `gorm:"not null" json:"baseUrl"`
	Model           string    `gorm:"not null" json:"model"`
	Parameters      []byte    `gorm:"type:jsonb" json:"parameters"`
	APIKeyEncrypted *string   `json:"-"`
	IsActive        bool      `gorm:"not null;default:false" json:"isActive"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type ChatSession struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    *uuid.UUID `gorm:"type:uuid;index" json:"userId,omitempty"`
	Title     string     `gorm:"not null" json:"title"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type Message struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SessionID uuid.UUID `gorm:"type:uuid;index;not null" json:"sessionId"`
	Role      string    `gorm:"not null" json:"role"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Status    string    `gorm:"not null" json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type SearchResult struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	MessageID uuid.UUID `gorm:"type:uuid;index;not null" json:"messageId"`
	Position  int       `gorm:"not null" json:"position"`
	Title     string    `gorm:"not null" json:"title"`
	URL       string    `gorm:"type:text;not null" json:"url"`
	Domain    string    `gorm:"not null" json:"domain"`
	Snippet   string    `gorm:"type:text" json:"snippet"`
	CreatedAt time.Time `json:"createdAt"`
}

type UsageLog struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	UserID       *uuid.UUID `gorm:"type:uuid;index" json:"userId,omitempty"`
	Query        string     `gorm:"type:text;not null" json:"query"`
	LatencyMS    int        `json:"latencyMs"`
	Status       string     `gorm:"not null" json:"status"`
	ErrorCode    *string    `json:"errorCode,omitempty"`
	ErrorMessage *string    `gorm:"type:text" json:"errorMessage,omitempty"`
	ProviderID   *uuid.UUID `gorm:"type:uuid" json:"providerId,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type CrawlJob struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     *uuid.UUID `gorm:"type:uuid;index" json:"userId,omitempty"`
	URL        string     `gorm:"type:text;not null" json:"url"`
	Status     string     `gorm:"not null" json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}
