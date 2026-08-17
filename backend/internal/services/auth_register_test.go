package services

import (
	"errors"
	"testing"

	"intellisearch/internal/config"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

func newAuthService(t *testing.T) *AuthService {
	t.Helper()
	cfg := config.Config{JWTSecret: "register-test-secret-32-chars-minimum", JWTTTLHours: 24}
	db := newTestDB(t)
	return NewAuthService(repositories.NewUserRepository(db), repositories.NewQueueConfigRepository(db), cfg)
}

// TestRegisterAppliesDefaultDailyQuota verifies newly registered accounts get
// the admin-configurable default quota (falling back to 3 when the config row
// is missing), and that changing the config default changes future sign-ups.
func TestRegisterAppliesDefaultDailyQuota(t *testing.T) {
	// No queue-config row: fallback default of 3.
	service := newAuthService(t)
	_, user, err := service.Register("Quota User", "quota@example.com", "password-123")
	if err != nil {
		t.Fatal(err)
	}
	if user.AIDailyQuota != 3 {
		t.Fatalf("expected fallback default quota 3, got %d", user.AIDailyQuota)
	}

	// A configured default of 5 applies to the next registration.
	db := newTestDB(t)
	if err := db.Create(&entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10, DefaultDailyQuota: 5}).Error; err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{JWTSecret: "register-test-secret-32-chars-minimum", JWTTTLHours: 24}
	configured := NewAuthService(repositories.NewUserRepository(db), repositories.NewQueueConfigRepository(db), cfg)
	_, user, err = configured.Register("Quota User", "quota2@example.com", "password-123")
	if err != nil {
		t.Fatal(err)
	}
	if user.AIDailyQuota != 5 {
		t.Fatalf("expected configured default quota 5, got %d", user.AIDailyQuota)
	}
}

func TestRegisterCreatesGeneralUserAndSignsIn(t *testing.T) {
	service := newAuthService(t)
	token, user, err := service.Register("New User", "New.User@Example.com", "password-123")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || user.Email != "new.user@example.com" || user.Name != "New User" {
		t.Fatalf("unexpected registration result: token=%q user=%+v", token, user)
	}
	if user.Role != entities.RoleGeneralUser || user.Status != entities.StatusActive {
		t.Fatalf("new accounts must be active general users, got %+v", user)
	}
	// The created account can sign in with the password.
	loginToken, loggedIn, err := service.Login("new.user@example.com", "password-123")
	if err != nil || loginToken == "" || loggedIn.ID != user.ID {
		t.Fatalf("registered account could not sign in: err=%v", err)
	}
}

func TestRegisterRejectsWeakPassword(t *testing.T) {
	service := newAuthService(t)
	if _, _, err := service.Register("New User", "a@example.com", "short"); !errors.Is(err, ErrInvalidRegistration) {
		t.Fatalf("expected ErrInvalidRegistration for short password, got %v", err)
	}
}

func TestRegisterRejectsEmptyName(t *testing.T) {
	service := newAuthService(t)
	if _, _, err := service.Register("   ", "a@example.com", "password-123"); !errors.Is(err, ErrInvalidRegistration) {
		t.Fatalf("expected ErrInvalidRegistration for empty name, got %v", err)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	service := newAuthService(t)
	if _, _, err := service.Register("First", "taken@example.com", "password-123"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Register("Second", "TAKEN@example.com", "password-123"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken for duplicate email, got %v", err)
	}
}
