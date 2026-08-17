package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"intellisearch/internal/config"
	"intellisearch/internal/repositories"
)

// googleMock spins up fake Google token + userinfo endpoints and returns an
// AuthService wired to them plus the servers for cleanup.
func googleMock(t *testing.T, email, name, picture string, status int) (*AuthService, *httptest.Server) {
	t.Helper()
	userinfo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mock-access-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(googleProfile{Email: email, Name: name, Picture: picture})
	}))
	token := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if status >= 400 {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "mock-access-token"})
	}))
	t.Cleanup(func() { userinfo.Close(); token.Close() })

	cfg := config.Config{
		JWTSecret:         "google-test-secret-32-chars-minimum",
		JWTTTLHours:       24,
		GoogleClientID:    "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRedirectURL:  "http://localhost:5173/api/v1/auth/google/callback",
		FrontendOrigin:     "http://localhost:5173",
	}
	db := newTestDB(t)
	service := NewAuthService(repositories.NewUserRepository(db), repositories.NewQueueConfigRepository(db), cfg)
	service.googleTokenURL = token.URL
	service.googleUserInfoURL = userinfo.URL
	return service, userinfo
}

func TestGoogleAuthURL(t *testing.T) {
	service, _ := googleMock(t, "jane@example.com", "Jane", "", http.StatusOK)
	url := service.GoogleAuthURL("state-123")
	for _, want := range []string{"accounts.google.com/o/oauth2/v2/auth", "client_id=client-id", "redirect_uri=", "state=state-123", "response_type=code"} {
		if !strings.Contains(url, want) {
			t.Fatalf("google auth url %q missing %q", url, want)
		}
	}
}

func TestGoogleCallbackCreatesAndReusesUser(t *testing.T) {
	service, _ := googleMock(t, "jane@example.com", "Jane Doe", "https://pics.example/jane.png", http.StatusOK)
	if !service.GoogleConfigured() {
		t.Fatal("expected google configured")
	}
	token, user, err := service.GoogleCallback("code", "state", "state")
	if err != nil {
		t.Fatalf("google callback failed: %v", err)
	}
	if token == "" || user.Email != "jane@example.com" || user.Name != "Jane Doe" || user.Role != "general_user" {
		t.Fatalf("unexpected first sign-in result: token=%q user=%+v", token, user)
	}
	if user.AIDailyQuota != 3 {
		t.Fatalf("new google users must get the default quota (3), got %d", user.AIDailyQuota)
	}
	if user.AvatarURL == nil || *user.AvatarURL != "https://pics.example/jane.png" {
		t.Fatalf("google avatar not stored: %v", user.AvatarURL)
	}

	// A second sign-in reuses the same account.
	token2, user2, err := service.GoogleCallback("code", "state", "state")
	if err != nil {
		t.Fatalf("second google callback failed: %v", err)
	}
	if token2 == "" || user2.ID != user.ID || user2.Role != "general_user" {
		t.Fatalf("expected same user on re-login: %+v vs %+v", user, user2)
	}
}

func TestGoogleCallbackRejectsStateMismatch(t *testing.T) {
	service, _ := googleMock(t, "jane@example.com", "Jane", "", http.StatusOK)
	if _, _, err := service.GoogleCallback("code", "wrong", "expected"); err == nil {
		t.Fatal("expected error on state mismatch")
	}
}

func TestGoogleCallbackHandlesProviderError(t *testing.T) {
	service, _ := googleMock(t, "jane@example.com", "Jane", "", http.StatusBadGateway)
	if _, _, err := service.GoogleCallback("code", "state", "state"); err == nil {
		t.Fatal("expected error when google userinfo fails")
	}
}

func TestGoogleNotConfigured(t *testing.T) {
	cfg := config.Config{JWTSecret: "no-google-secret-32-chars-minimum", JWTTTLHours: 24}
	db := newTestDB(t)
	service := NewAuthService(repositories.NewUserRepository(db), repositories.NewQueueConfigRepository(db), cfg)
	if service.GoogleConfigured() {
		t.Fatal("expected google not configured with empty credentials")
	}
	if _, _, err := service.GoogleCallback("code", "state", "state"); err != ErrGoogleUnavailable {
		t.Fatalf("expected ErrGoogleUnavailable, got %v", err)
	}
}
