package services

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"intellisearch/internal/config"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRegistration = errors.New("invalid registration data")
	ErrEmailTaken          = errors.New("email already registered")
)

type Claims struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
type AuthService struct {
	users              *repositories.UserRepository
	queueConfig        *repositories.QueueConfigRepository
	secret             []byte
	ttl                time.Duration
	googleClientID     string
	googleClientSecret string
	googleRedirectURL  string
	frontendOrigin     string
	httpClient         *http.Client
	googleTokenURL     string
	googleUserInfoURL  string
}

func NewAuthService(users *repositories.UserRepository, queueConfig *repositories.QueueConfigRepository, cfg config.Config) *AuthService {
	ttl := time.Duration(cfg.JWTTTLHours) * time.Hour
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &AuthService{
		users:              users,
		queueConfig:        queueConfig,
		secret:             []byte(cfg.JWTSecret),
		ttl:                ttl,
		googleClientID:     cfg.GoogleClientID,
		googleClientSecret: cfg.GoogleClientSecret,
		googleRedirectURL:  cfg.GoogleRedirectURL,
		frontendOrigin:     cfg.FrontendOrigin,
		httpClient:         &http.Client{Timeout: 10 * time.Second},
		googleTokenURL:     "https://oauth2.googleapis.com/token",
		googleUserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
	}
}

// defaultDailyQuota returns the admin-configurable daily AI-usage quota for
// newly registered accounts, falling back to 3 when the config row is missing
// (e.g. before seeding). 0 means unlimited, matching per-user semantics.
func (s *AuthService) defaultDailyQuota() int {
	if s.queueConfig != nil {
		if config, err := s.queueConfig.Get(); err == nil {
			return config.DefaultDailyQuota
		}
	}
	return 3
}

// GoogleConfigured reports whether Google SSO credentials are present.
func (s *AuthService) GoogleConfigured() bool {
	return s.googleClientID != "" && s.googleClientSecret != "" && s.googleRedirectURL != ""
}

// FrontendOrigin returns the configured origin used to redirect the browser
// back after OAuth completes.
func (s *AuthService) FrontendOrigin() string {
	if s.frontendOrigin == "" {
		return "http://localhost:5173"
	}
	return s.frontendOrigin
}

// GoogleAuthURL builds the Google OAuth authorization URL for the given CSRF
// state value. The state round-trips through Google so the callback can verify
// the flow started on our side.
func (s *AuthService) GoogleAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", s.googleClientID)
	params.Set("redirect_uri", s.googleRedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("access_type", "online")
	params.Set("prompt", "select_account")
	params.Set("state", state)
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

// GoogleCallback exchanges the authorization code for a Google identity,
// finds or creates the matching user, and issues an app JWT.
func (s *AuthService) GoogleCallback(code, state, expectedState string) (string, entities.User, error) {
	if !s.GoogleConfigured() {
		return "", entities.User{}, ErrGoogleUnavailable
	}
	if state == "" || state != expectedState {
		return "", entities.User{}, ErrGoogleUnavailable
	}
	profile, err := s.fetchGoogleProfile(code)
	if err != nil {
		return "", entities.User{}, err
	}
	email := strings.ToLower(strings.TrimSpace(profile.Email))
	if email == "" {
		return "", entities.User{}, ErrGoogleUnavailable
	}
	user, err := s.users.ByEmail(email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		name := strings.TrimSpace(profile.Name)
		if name == "" {
			name = email
		}
		// New Google users have no password; a random hash keeps the column
		// non-empty while making password login impossible for the account.
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(RandomToken(32)), bcrypt.DefaultCost)
		if hashErr != nil {
			return "", entities.User{}, hashErr
		}
		user = entities.User{
			ID: uuid.New(), Name: name, Email: email, PasswordHash: string(hash),
			Role: entities.RoleGeneralUser, Status: entities.StatusActive,
			AIDailyQuota: s.defaultDailyQuota(),
			CreatedAt:    time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		if profile.Picture != "" {
			avatar := profile.Picture
			user.AvatarURL = &avatar
		}
		if err := s.users.Create(&user); err != nil {
			return "", entities.User{}, err
		}
	} else if err != nil {
		return "", entities.User{}, err
	} else if user.Status != entities.StatusActive {
		return "", entities.User{}, ErrInvalidCredentials
	}
	now := time.Now().UTC()
	user.LastLoginAt = &now
	if err := s.users.Save(&user); err != nil {
		return "", entities.User{}, err
	}
	claims := Claims{UserID: user.ID.String(), Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)), IssuedAt: jwt.NewNumericDate(now)}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	return token, user, err
}

// fetchGoogleProfile exchanges the authorization code for an access token and
// reads the user's Google profile (email, name, picture).
func (s *AuthService) fetchGoogleProfile(code string) (googleProfile, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", s.googleClientID)
	form.Set("client_secret", s.googleClientSecret)
	form.Set("redirect_uri", s.googleRedirectURL)
	form.Set("grant_type", "authorization_code")
	tokenResponse, err := s.httpClient.PostForm(s.googleTokenURL, form)
	if err != nil {
		return googleProfile{}, ErrGoogleUnavailable
	}
	defer tokenResponse.Body.Close()
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResponse.Body).Decode(&token); err != nil || token.AccessToken == "" {
		return googleProfile{}, ErrGoogleUnavailable
	}
	req, err := http.NewRequest(http.MethodGet, s.googleUserInfoURL, nil)
	if err != nil {
		return googleProfile{}, ErrGoogleUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err := s.httpClient.Do(req)
	if err != nil {
		return googleProfile{}, ErrGoogleUnavailable
	}
	defer response.Body.Close()
	var profile googleProfile
	if err := json.NewDecoder(response.Body).Decode(&profile); err != nil {
		return googleProfile{}, ErrGoogleUnavailable
	}
	return profile, nil
}

// Register creates a new general-user account, hashes the password, and issues
// a JWT so the user lands signed in. Google SSO registration uses the same
// find-or-create path as GoogleCallback.
func (s *AuthService) Register(name, email, password string) (string, entities.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" || !strings.Contains(email, "@") || len(password) < 8 {
		return "", entities.User{}, ErrInvalidRegistration
	}
	if _, err := s.users.ByEmail(email); err == nil {
		return "", entities.User{}, ErrEmailTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", entities.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", entities.User{}, err
	}
	now := time.Now().UTC()
	user := entities.User{
		ID: uuid.New(), Name: name, Email: email, PasswordHash: string(hash),
		Role: entities.RoleGeneralUser, Status: entities.StatusActive,
		AIDailyQuota: s.defaultDailyQuota(),
		CreatedAt:    now, UpdatedAt: now,
	}
	if err := s.users.Create(&user); err != nil {
		return "", entities.User{}, err
	}
	user.LastLoginAt = &now
	if err := s.users.Save(&user); err != nil {
		return "", entities.User{}, err
	}
	claims := Claims{UserID: user.ID.String(), Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)), IssuedAt: jwt.NewNumericDate(now)}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	return token, user, err
}

func (s *AuthService) Login(email, password string) (string, entities.User, error) {
	user, err := s.users.ByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil || user.Status != entities.StatusActive {
		return "", entities.User{}, ErrInvalidCredentials
	}
	now := time.Now().UTC()
	user.LastLoginAt = &now
	if err := s.users.Save(&user); err != nil {
		return "", entities.User{}, err
	}
	claims := Claims{UserID: user.ID.String(), Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)), IssuedAt: jwt.NewNumericDate(now)}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	return token, user, err
}
func (s *AuthService) Parse(token string) (Claims, error) {
	claims := Claims{}
	parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidCredentials
		}
		return s.secret, nil
	})
	if err != nil || !parsed.Valid {
		return Claims{}, ErrInvalidCredentials
	}
	return claims, nil
}
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
