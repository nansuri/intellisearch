package services

import (
	"intellisearch/internal/config"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"strings"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Claims struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
type AuthService struct {
	users  *repositories.UserRepository
	secret []byte
	ttl    time.Duration
}

func NewAuthService(users *repositories.UserRepository, cfg config.Config) *AuthService {
	ttl := time.Duration(cfg.JWTTTLHours) * time.Hour
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &AuthService{users: users, secret: []byte(cfg.JWTSecret), ttl: ttl}
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
