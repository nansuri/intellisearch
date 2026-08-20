package services

import (
	"errors"
	"testing"
	"time"

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

// TestSessionTTLResolution verifies the issued JWT lifetime: the
// admin-editable ai_queue_config.session_ttl_hours wins when set (>0), and the
// env JWT_TTL_HOURS fallback applies when it is 0 or the row is missing.
func TestSessionTTLResolution(t *testing.T) {
	ttlOf := func(t *testing.T, service *AuthService) time.Duration {
		t.Helper()
		token, _, err := service.Register("TTL User", "ttl@example.com", "password-123")
		if err != nil {
			t.Fatal(err)
		}
		claims, err := service.Parse(token)
		if err != nil {
			t.Fatalf("parse token: %v", err)
		}
		duration := time.Until(claims.ExpiresAt.Time)
		if duration < 0 {
			t.Fatalf("token already expired: %s", claims.ExpiresAt.Time)
		}
		return duration
	}

	// No queue-config row: the env JWT_TTL_HOURS (24h) applies.
	service := newAuthService(t)
	if got := ttlOf(t, service); got < 23*time.Hour || got > 25*time.Hour {
		t.Fatalf("expected ~24h fallback session, got %s", got)
	}

	// A configured session_ttl_hours of 72 wins over the env value.
	db := newTestDB(t)
	if err := db.Create(&entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10, SessionTTLHours: 72}).Error; err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{JWTSecret: "register-test-secret-32-chars-minimum", JWTTTLHours: 24}
	configured := NewAuthService(repositories.NewUserRepository(db), repositories.NewQueueConfigRepository(db), cfg)
	if got := ttlOf(t, configured); got < 71*time.Hour || got > 73*time.Hour {
		t.Fatalf("expected ~72h admin-configured session, got %s", got)
	}

	// A zero session_ttl_hours means "unset": fall back to the env value. It is
	// written through the admin update path (Save persists zeros; Create would
	// apply the 168 column default).
	db2 := newTestDB(t)
	if err := db2.Create(&entities.AIQueueConfig{ID: 1, MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10, SessionTTLHours: 168}).Error; err != nil {
		t.Fatal(err)
	}
	admin := NewAdminService(repositories.NewProviderRepository(db2), repositories.NewQueueConfigRepository(db2), repositories.NewSiteRepository(db2), "k", t.TempDir())
	if _, err := admin.UpdateQueueConfig(4, 20, 60000, 10, 6, 3, 20, 0); err != nil {
		t.Fatalf("set session ttl to 0: %v", err)
	}
	zeroCfg := NewAuthService(repositories.NewUserRepository(db2), repositories.NewQueueConfigRepository(db2), cfg)
	if got := ttlOf(t, zeroCfg); got < 23*time.Hour || got > 25*time.Hour {
		t.Fatalf("expected 24h env fallback for zero config, got %s", got)
	}
}
