package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"intellisearch/internal/models/entities"
)

// MiniAppRepository owns persistence for user-created mini apps. All access is
// scoped by user id for the owner endpoints; the public list is filtered to
// active, public apps only.
type MiniAppRepository struct{ db *gorm.DB }

func NewMiniAppRepository(db *gorm.DB) *MiniAppRepository { return &MiniAppRepository{db: db} }

// ListByUser returns every app owned by the user, most recently updated first.
func (r *MiniAppRepository) ListByUser(userID uuid.UUID) ([]entities.MiniApp, error) {
	var apps []entities.MiniApp
	err := r.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&apps).Error
	return apps, err
}

// PublicList returns every active, public app, most recently updated first.
func (r *MiniAppRepository) PublicList() ([]entities.MiniApp, error) {
	var apps []entities.MiniApp
	err := r.db.Where("visibility = ? AND is_active = ?", entities.MiniAppVisibilityPublic, true).Order("updated_at DESC").Find(&apps).Error
	return apps, err
}

// GetByID returns any app by id (used by the owner routes, which scope the
// result to the requesting user via GetUserApp).
func (r *MiniAppRepository) GetByID(id uuid.UUID) (entities.MiniApp, error) {
	var app entities.MiniApp
	err := r.db.First(&app, "id = ?", id).Error
	return app, err
}

// GetUserApp returns one of the user's own apps.
func (r *MiniAppRepository) GetUserApp(userID, id uuid.UUID) (entities.MiniApp, error) {
	var app entities.MiniApp
	err := r.db.First(&app, "id = ? AND user_id = ?", id, userID).Error
	return app, err
}

// GetBySlug returns an app by its URL slug (used by the public runner route
// and the drawer deep links).
func (r *MiniAppRepository) GetBySlug(slug string) (entities.MiniApp, error) {
	var app entities.MiniApp
	err := r.db.First(&app, "slug = ?", slug).Error
	return app, err
}

// SlugExists reports whether a slug is already taken (globally unique slugs
// keep public apps at stable, shareable URLs).
func (r *MiniAppRepository) SlugExists(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&entities.MiniApp{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}

func (r *MiniAppRepository) Create(app *entities.MiniApp) error { return r.db.Create(app).Error }

func (r *MiniAppRepository) Update(app *entities.MiniApp) error { return r.db.Save(app).Error }

// Delete removes one of the user's apps and returns the number of rows affected.
func (r *MiniAppRepository) Delete(userID, id uuid.UUID) (int64, error) {
	result := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&entities.MiniApp{})
	return result.RowsAffected, result.Error
}