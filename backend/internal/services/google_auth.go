package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// ErrGoogleUnavailable indicates Google sign-in could not be completed, either
// because credentials are missing or the OAuth exchange failed. The message is
// sanitized by the handler; internal details are logged by the caller.
var ErrGoogleUnavailable = errors.New("google sign-in unavailable")

// googleProfile is the subset of the Google userinfo response we store.
type googleProfile struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// RandomToken returns a cryptographically random hex string (bytes*2 chars).
func RandomToken(bytes int) string {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err == nil {
		return hex.EncodeToString(raw)
	}
	// crypto/rand failure is practically unrecoverable; fall back to a
	// timestamp so account creation never blocks on entropy.
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
