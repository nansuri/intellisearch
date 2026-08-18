package repositories

import (
	"errors"
	"time"

	"intellisearch/internal/models/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }
func (r *UserRepository) ByEmail(email string) (entities.User, error) {
	var user entities.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return user, err
}
func (r *UserRepository) ByID(id uuid.UUID) (entities.User, error) {
	var user entities.User
	err := r.db.First(&user, "id = ?", id).Error
	return user, err
}
func (r *UserRepository) Save(user *entities.User) error { return r.db.Save(user).Error }

// Create inserts a new user row.
func (r *UserRepository) Create(user *entities.User) error { return r.db.Create(user).Error }

// Delete removes a user by id.
func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.User{}, "id = ?", id).Error
}

// Count returns the total number of registered accounts.
func (r *UserRepository) Count() (int64, error) {
	var total int64
	err := r.db.Model(&entities.User{}).Count(&total).Error
	return total, err
}

// CreatedSince returns the creation times of all accounts created at or after
// a cutoff (used to bucket new-registration daily/weekly trends).
func (r *UserRepository) CreatedSince(start time.Time) ([]time.Time, error) {
	var users []entities.User
	if err := r.db.Where("created_at >= ?", start).Order("created_at asc").Find(&users).Error; err != nil {
		return nil, err
	}
	times := make([]time.Time, 0, len(users))
	for _, user := range users {
		times = append(times, user.CreatedAt)
	}
	return times, nil
}

// List returns a searchable, paginated page of users ordered by creation date.
// An empty query matches every user. Page is 1-based; pageSize is capped at 100.
func (r *UserRepository) List(query string, page, pageSize int) ([]entities.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	tx := r.db.Model(&entities.User{})
	if query != "" {
		pattern := "%" + query + "%"
		tx = tx.Where("name LIKE ? OR email LIKE ?", pattern, pattern)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []entities.User
	err := tx.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return users, total, nil
	}
	return users, total, err
}