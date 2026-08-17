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
	ID         uint    `gorm:"primaryKey" json:"-"`
	SiteName   string  `gorm:"not null" json:"siteName"`
	LogoURL    *string `json:"logoUrl"`
	FaviconURL *string `json:"faviconUrl"`
	Tagline    *string `json:"tagline"`
	// Copyright is the short legal line rendered in the site footer
	// ("© 2026 Acme Search"). Null falls back to the site name.
	Copyright *string   `json:"copyright"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AIQueueConfig struct {
	ID                   uint `gorm:"primaryKey" json:"-"`
	MaxConcurrent        int  `gorm:"not null" json:"maxConcurrent"`
	MaxQueueSize         int  `gorm:"not null" json:"maxQueueSize"`
	RequestTimeoutMS     int  `gorm:"not null" json:"requestTimeoutMs"`
	PerUserRateLimit     int  `gorm:"not null" json:"perUserRateLimit"`
	SuggestionCacheHours int  `gorm:"not null;default:6" json:"suggestionCacheHours"`
	// DefaultDailyQuota is the daily AI-usage quota applied to newly registered
	// users (password or Google SSO). 0 means unlimited, matching the per-user
	// AIDailyQuota semantics; the Owner Control Panel can change it anytime and
	// it only affects accounts created afterwards.
	DefaultDailyQuota int `gorm:"not null;default:3" json:"defaultDailyQuota"`
	// MaxImageResults caps how many image results are returned/persisted per
	// web-search ask (0 = unlimited). Follow-up asks skip the image search
	// entirely, so this only bounds the primary search. Admin-configurable.
	MaxImageResults int       `gorm:"not null;default:20" json:"maxImageResults"`
	UpdatedAt       time.Time `json:"updatedAt"`
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

// ImageResult is a SearXNG image result persisted against the assistant
// message that searched for it (both search-only and enhanced asks), so the
// result page can re-render the image grid on restore without re-searching.
type ImageResult struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	MessageID    uuid.UUID `gorm:"type:uuid;index;not null" json:"messageId"`
	Position     int       `gorm:"not null" json:"position"`
	Title        string    `gorm:"not null" json:"title"`
	URL          string    `gorm:"type:text;not null" json:"url"`
	ThumbnailURL string    `gorm:"type:text;not null" json:"thumbnailUrl"`
	Source       string    `gorm:"not null" json:"source"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	CreatedAt    time.Time `json:"createdAt"`
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

// SearchHistory is a per-user record of every search they run. It powers the
// "recent searches" chips on the main page, the AI-composed suggestions, and
// the search-history page, and can be cleared by the user. Instead of storing
// the full answer text, each row keeps the IDs of the chat session and the
// assistant message that answered it, so summaries are fetched on demand from
// the messages table (two UUIDs per row, no answer-text duplication).
type SearchHistory struct {
	ID        uint64     `gorm:"primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_search_history_user,priority:1" json:"userId"`
	Query     string     `gorm:"type:text;not null" json:"query"`
	SessionID *uuid.UUID `gorm:"type:uuid" json:"sessionId,omitempty"`
	MessageID *uuid.UUID `gorm:"type:uuid" json:"messageId,omitempty"`
	CreatedAt time.Time  `gorm:"index:idx_search_history_user,priority:2" json:"createdAt"`
}

// Note is a user-owned mini-app note. It doubles as a "save summary to
// notes" target from the result page, so it optionally links back to the
// search (source query + session) that produced the content.
type Note struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index:idx_notes_user,priority:1" json:"userId"`
	Title       string     `gorm:"not null" json:"title"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	SourceQuery string     `gorm:"type:text" json:"sourceQuery,omitempty"`
	SessionID   *uuid.UUID `gorm:"type:uuid" json:"sessionId,omitempty"`
	CreatedAt   time.Time  `gorm:"index:idx_notes_user,priority:2" json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// AnonymousUsage is the fraud backstop for the anonymous guest limit: each
// row claims the single AI-usage allowance for one visitor token AND one IP
// (both unique). A row's existence means that visitor/IP has already used its
// one guest AI search, so clearing cookies or local storage cannot reset the
// count — only changing IP would, and the IP hash is derived from the
// proxy-verified client address.
type AnonymousUsage struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	VisitorID uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"visitorId"`
	IPHash    string    `gorm:"type:varchar(64);uniqueIndex" json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CrawlJob struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     *uuid.UUID `gorm:"type:uuid;index" json:"userId,omitempty"`
	URL        string     `gorm:"type:text;not null" json:"url"`
	Status     string     `gorm:"not null" json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}
