package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/services"
)

const oauthStateCookie = "intellisearch_oauth_state"

type AuthHandler struct{ service *services.AuthService }

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}
func (h *AuthHandler) Login(c *gin.Context) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.AUTH01001, "Enter your email and password.", "parse login request", err)
		return
	}
	token, user, err := h.service.Login(request.Email, request.Password)
	if err != nil {
		middleware.RespondError(c, http.StatusUnauthorized, contracts.AUTH01001, "Invalid email or password.", "login rejected", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"token": token, "user": user}))
}

// Register creates a new account and returns a signed JWT plus the profile, so
// the user lands signed in. Google SSO registration happens through the same
// OAuth flow as GoogleStart/GoogleCallback (find-or-create).
func (h *AuthHandler) Register(c *gin.Context) {
	var request struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.AUTH01004, "Enter your name, email, and a password of at least 8 characters.", "parse register request", err)
		return
	}
	token, user, err := h.service.Register(request.Name, request.Email, request.Password)
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			middleware.RespondError(c, http.StatusConflict, contracts.AUTH01005, "An account with that email already exists — try signing in.", "register email taken", err)
			return
		}
		if errors.Is(err, services.ErrInvalidRegistration) {
			middleware.RespondError(c, http.StatusBadRequest, contracts.AUTH01004, "Enter your name, a valid email, and a password of at least 8 characters.", "invalid registration", err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, contracts.AUTH01004, "Your account could not be created. Please try again.", "register failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"token": token, "user": user}))
}

// GoogleStart kicks off the OAuth authorization-code flow: it stores a random
// CSRF state in an httpOnly cookie and redirects the browser to Google.
func (h *AuthHandler) GoogleStart(c *gin.Context) {
	if !h.service.GoogleConfigured() {
		middleware.RespondError(c, http.StatusServiceUnavailable, contracts.AUTH01003, "Google sign-in is not configured on this deployment.", "google not configured", nil)
		return
	}
	state := services.RandomToken(24)
	h.setOAuthStateCookie(c, state, 600)
	c.Redirect(http.StatusFound, h.service.GoogleAuthURL(state))
}

// GoogleCallback exchanges the authorization code for an app JWT and sends the
// browser back to the frontend with the token in the query string.
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	expected, err := c.Cookie(oauthStateCookie)
	if err != nil || expected == "" || state == "" || state != expected {
		logrus.WithField("stateMismatch", state != expected).Warn("google oauth state mismatch")
		h.redirectAuthError(c, "Sign-in expired or failed. Please try again.")
		return
	}
	h.setOAuthStateCookie(c, "", -1)

	token, user, err := h.service.GoogleCallback(code, state, expected)
	if err != nil {
		logrus.WithError(err).Error("google oauth callback failed")
		h.redirectAuthError(c, "Google sign-in failed. Please try again.")
		return
	}
	location := h.service.FrontendOrigin() + "/auth/callback?token=" + url.QueryEscape(token) + "&name=" + url.QueryEscape(user.Name)
	c.Redirect(http.StatusFound, location)
}

// setOAuthStateCookie writes the CSRF state cookie, marking it Secure when the
// configured frontend origin is served over HTTPS (nginx terminates TLS, so the
// request itself may still be plain HTTP inside the stack).
func (h *AuthHandler) setOAuthStateCookie(c *gin.Context, value string, maxAge int) {
	secure := strings.HasPrefix(h.service.FrontendOrigin(), "https://")
	c.SetCookie(oauthStateCookie, value, maxAge, "/", "", secure, true)
}

// redirectAuthError sends the browser back to the login page with a friendly
// error instead of rendering an envelope (the browser expects a redirect here).
func (h *AuthHandler) redirectAuthError(c *gin.Context, message string) {
	location := h.service.FrontendOrigin() + "/login?error=" + url.QueryEscape(message)
	c.Redirect(http.StatusFound, location)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"loggedOut": true}))
}
