package services

import (
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrProfileInvalid = errors.New("invalid profile")
	ErrAdminInvalid   = errors.New("invalid admin input")
)

type UserService struct {
	repo   *repositories.UserRepository
	usage  *repositories.UsageLogRepository
	uploads string
}

func NewUserService(repo *repositories.UserRepository, usage *repositories.UsageLogRepository, uploads string) *UserService {
	return &UserService{repo: repo, usage: usage, uploads: uploads}
}
func (s *UserService) Get(id uuid.UUID) (entities.User, error) { return s.repo.ByID(id) }
func (s *UserService) UpdateProfile(id uuid.UUID, name, email string) (entities.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" || !strings.Contains(email, "@") {
		return entities.User{}, ErrProfileInvalid
	}
	user, err := s.repo.ByID(id)
	if err != nil {
		return user, err
	}
	user.Name = name
	user.Email = email
	return user, s.repo.Save(&user)
}

// Usage reports how many asks the user has used today and their daily quota.
func (s *UserService) Usage(userID uuid.UUID) (used int64, quota int, err error) {
	user, err := s.repo.ByID(userID)
	if err != nil {
		return 0, 0, err
	}
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	used, err = s.usage.CountSince(userID, start)
	return used, user.AIDailyQuota, err
}

// Avatar validates and stores an uploaded avatar image, then updates the user.
func (s *UserService) Avatar(userID uuid.UUID, filename string, data []byte) (string, error) {
	url, err := saveUpload(s.uploads, "avatars", userID.String(), filename, data, 2<<20)
	if err != nil {
		return "", err
	}
	user, err := s.repo.ByID(userID)
	if err != nil {
		return "", err
	}
	user.AvatarURL = &url
	return url, s.repo.Save(&user)
}

// List returns a searchable, paginated page of users for the control panel.
func (s *UserService) List(query string, page, pageSize int) ([]entities.User, int64, error) {
	return s.repo.List(strings.TrimSpace(query), page, pageSize)
}

// Create adds a new user with a bcrypt-hashed password.
func (s *UserService) Create(name, email, password, role string, quota int) (entities.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" || !strings.Contains(email, "@") || password == "" || !validRole(role) || quota < 0 {
		return entities.User{}, ErrAdminInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return entities.User{}, err
	}
	user := entities.User{ID: uuid.New(), Name: name, Email: email, PasswordHash: string(hash), Role: role, Status: entities.StatusActive, AIDailyQuota: quota, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.repo.Create(&user); err != nil {
		return entities.User{}, err
	}
	return user, nil
}

// Update changes a user's role, status, and/or daily quota.
func (s *UserService) Update(id uuid.UUID, role, status string, quota int) (entities.User, error) {
	user, err := s.repo.ByID(id)
	if err != nil {
		return user, err
	}
	if role != "" {
		if !validRole(role) {
			return entities.User{}, ErrAdminInvalid
		}
		user.Role = role
	}
	if status != "" {
		if status != entities.StatusActive && status != entities.StatusSuspended {
			return entities.User{}, ErrAdminInvalid
		}
		user.Status = status
	}
	if quota >= 0 {
		user.AIDailyQuota = quota
	}
	user.UpdatedAt = time.Now().UTC()
	return user, s.repo.Save(&user)
}

// Delete removes a user.
func (s *UserService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func validRole(role string) bool {
	return role == entities.RoleGeneralUser || role == entities.RoleSuperOwner
}