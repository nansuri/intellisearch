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
	return NewAuthService(repositories.NewUserRepository(newTestDB(t)), cfg)
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
