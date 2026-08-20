package services

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

var (
	ErrMiniAppInvalid   = errors.New("invalid mini app")
	ErrMiniAppNotFound  = errors.New("mini app not found")
	ErrMiniAppSlugTaken = errors.New("mini app slug taken")
	ErrMiniAppPrivate   = errors.New("mini app is private")
	// ErrMiniAppGenerate marks an AI generation whose reply could not be parsed
	// into a usable app draft (mapped to MINI02001).
	ErrMiniAppGenerate = errors.New("ai mini app draft could not be parsed")
)

const (
	maxMiniAppNameLength        = 80
	maxMiniAppDescriptionLength = 500
	maxMiniAppIconLength        = 16
	// Per-source caps keep a user-provided app from becoming an unbounded text
	// store; JS gets more room than markup.
	maxMiniAppSourceLength = 60000
	maxMiniAppJSLength     = 120000
)

// MiniAppInput is the validated, user-supplied profile+source of a mini app.
type MiniAppInput struct {
	Name        string
	Description string
	Icon        string
	HTML        string
	CSS         string
	JS          string
	Visibility  string
}

// MiniAppPatch is a partial update: nil fields are left unchanged, so a patch
// that only edits the CSS can never blank the HTML/JS by accident.
type MiniAppPatch struct {
	Name        *string
	Description *string
	Icon        *string
	HTML        *string
	CSS         *string
	JS          *string
	Visibility  *string
}

// MiniAppSummary is the lightweight, source-free view of a mini app used by the
// public list (the app drawer/gallery). Source is only sent for the runner.
type MiniAppSummary struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"userId"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Visibility  string    `json:"visibility"`
	IsActive    bool      `json:"isActive"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// MiniAppService validates and persists user-created mini apps and serves the
// public app list. It is the only service with mini-app domain rules.
type MiniAppService struct {
	repo *repositories.MiniAppRepository
}

func NewMiniAppService(repo *repositories.MiniAppRepository) *MiniAppService {
	return &MiniAppService{repo: repo}
}

// List returns the user's own apps, newest first.
func (s *MiniAppService) List(userID uuid.UUID) ([]entities.MiniApp, error) {
	return s.repo.ListByUser(userID)
}

// PublicList returns active public apps as lightweight summaries.
func (s *MiniAppService) PublicList() ([]MiniAppSummary, error) {
	apps, err := s.repo.PublicList()
	if err != nil {
		return nil, err
	}
	summaries := make([]MiniAppSummary, 0, len(apps))
	for _, app := range apps {
		summaries = append(summaries, MiniAppSummary{ID: app.ID, UserID: app.UserID, Name: app.Name, Slug: app.Slug, Description: app.Description, Icon: app.Icon, Visibility: app.Visibility, IsActive: app.IsActive, UpdatedAt: app.UpdatedAt})
	}
	return summaries, nil
}

// Get returns one of the user's own apps.
func (s *MiniAppService) Get(userID, id uuid.UUID) (entities.MiniApp, error) {
	app, err := s.repo.GetUserApp(userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.MiniApp{}, ErrMiniAppNotFound
		}
		return entities.MiniApp{}, err
	}
	return app, nil
}

// GetForRun resolves an app by slug for rendering. Public apps are readable by
// anyone; private apps require the owning user (caller passes the optional
// user id from a valid session, nil for anonymous).
func (s *MiniAppService) GetForRun(slug string, userID *uuid.UUID) (entities.MiniApp, error) {
	app, err := s.repo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.MiniApp{}, ErrMiniAppNotFound
		}
		return entities.MiniApp{}, err
	}
	if app.Visibility == entities.MiniAppVisibilityPrivate && (userID == nil || *userID != app.UserID) {
		return entities.MiniApp{}, ErrMiniAppPrivate
	}
	if !app.IsActive && (userID == nil || *userID != app.UserID) {
		return entities.MiniApp{}, ErrMiniAppNotFound
	}
	return app, nil
}

// Create validates and persists a new mini app, deriving a unique slug.
func (s *MiniAppService) Create(userID uuid.UUID, input MiniAppInput) (entities.MiniApp, error) {
	if err := validateMiniApp(input); err != nil {
		return entities.MiniApp{}, err
	}
	slug, err := s.makeUniqueSlug(input.Name)
	if err != nil {
		return entities.MiniApp{}, err
	}
	now := time.Now().UTC()
	app := entities.MiniApp{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        strings.TrimSpace(input.Name),
		Slug:        slug,
		Description: strings.TrimSpace(input.Description),
		Icon:        strings.TrimSpace(input.Icon),
		HTML:        input.HTML,
		CSS:         input.CSS,
		JS:          input.JS,
		Visibility:  input.Visibility,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(&app); err != nil {
		return entities.MiniApp{}, err
	}
	return app, nil
}

// CreateDraft persists an AI-generated app as an editable draft. It mirrors
// Create but accepts a generated name/draft without extra validation of the
// source (generation already enforces the caps).
func (s *MiniAppService) CreateDraft(userID uuid.UUID, draft MiniAppDraft) (entities.MiniApp, error) {
	input := MiniAppInput{Name: draft.Name, Description: draft.Description, Icon: draft.Icon, HTML: draft.HTML, CSS: draft.CSS, JS: draft.JS, Visibility: entities.MiniAppVisibilityPrivate}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		input.Name = "AI mini app"
	}
	if err := validateMiniApp(input); err != nil {
		return entities.MiniApp{}, err
	}
	slug, err := s.makeUniqueSlug(name)
	if err != nil {
		return entities.MiniApp{}, err
	}
	now := time.Now().UTC()
	app := entities.MiniApp{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(input.Description),
		Icon:        strings.TrimSpace(input.Icon),
		HTML:        input.HTML,
		CSS:         input.CSS,
		JS:          input.JS,
		Visibility:  entities.MiniAppVisibilityPrivate,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(&app); err != nil {
		return entities.MiniApp{}, err
	}
	return app, nil
}

// Update validates and applies a partial update to one of the user's apps.
// Nil patch fields are preserved; a changed name re-derives a unique slug.
func (s *MiniAppService) Update(userID, id uuid.UUID, patch MiniAppPatch) (entities.MiniApp, error) {
	app, err := s.repo.GetUserApp(userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.MiniApp{}, ErrMiniAppNotFound
		}
		return entities.MiniApp{}, err
	}
	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return entities.MiniApp{}, ErrMiniAppInvalid
		}
		if name != app.Name {
			slug, slugErr := s.makeUniqueSlug(name)
			if slugErr != nil {
				return entities.MiniApp{}, slugErr
			}
			app.Slug = slug
		}
		app.Name = name
	}
	if patch.Description != nil {
		app.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.Icon != nil {
		app.Icon = strings.TrimSpace(*patch.Icon)
	}
	if patch.HTML != nil {
		app.HTML = *patch.HTML
	}
	if patch.CSS != nil {
		app.CSS = *patch.CSS
	}
	if patch.JS != nil {
		app.JS = *patch.JS
	}
	if patch.Visibility != nil {
		app.Visibility = *patch.Visibility
	}
	if err := validateMiniApp(MiniAppInput{Name: app.Name, Description: app.Description, Icon: app.Icon, HTML: app.HTML, CSS: app.CSS, JS: app.JS, Visibility: app.Visibility}); err != nil {
		return entities.MiniApp{}, err
	}
	app.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(&app); err != nil {
		return entities.MiniApp{}, err
	}
	return app, nil
}

// Delete removes one of the user's apps; missing apps return ErrMiniAppNotFound.
func (s *MiniAppService) Delete(userID, id uuid.UUID) error {
	affected, err := s.repo.Delete(userID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMiniAppNotFound
	}
	return nil
}

// validateMiniApp enforces the name/source caps and the visibility enum.
func validateMiniApp(input MiniAppInput) error {
	name := strings.TrimSpace(input.Name)
	if name == "" || len([]rune(name)) > maxMiniAppNameLength {
		return ErrMiniAppInvalid
	}
	if len([]rune(input.Description)) > maxMiniAppDescriptionLength {
		return ErrMiniAppInvalid
	}
	if len([]rune(input.Icon)) > maxMiniAppIconLength {
		return ErrMiniAppInvalid
	}
	if len([]rune(input.HTML)) > maxMiniAppSourceLength ||
		len([]rune(input.CSS)) > maxMiniAppSourceLength ||
		len([]rune(input.JS)) > maxMiniAppJSLength {
		return ErrMiniAppInvalid
	}
	if input.Visibility != entities.MiniAppVisibilityPublic && input.Visibility != entities.MiniAppVisibilityPrivate {
		return ErrMiniAppInvalid
	}
	return nil
}

// makeUniqueSlug derives a URL-safe slug from an app name and guarantees it is
// globally unique (the reason slugs are per-app, not per-user: a public app's
// /mini-apps/:slug link must stay stable and unique).
func (s *MiniAppService) makeUniqueSlug(name string) (string, error) {
	base := slugify(name)
	candidate := base
	for n := 2; ; n++ {
		taken, err := s.repo.SlugExists(candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		candidate = base + "-" + strconv.Itoa(n)
		if len(candidate) > 64 {
			return "", ErrMiniAppInvalid
		}
	}
}

// slugify turns an app name into a lowercase URL-safe token: unicode letters
// are dropped, runs of spaces/punctuation collapse to a single hyphen.
func slugify(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			// drop punctuation/unicode
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "app"
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}