package handlers

import (
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
		middleware.JSON(c, http.StatusBadRequest, contracts.Fail(contracts.AUTH01001, "Enter your email and password."))
		return
	}
	token, user, err := h.service.Login(request.Email, request.Password)
	if err != nil {
		middleware.JSON(c, http.StatusUnauthorized, contracts.Fail(contracts.AUTH01001, "Invalid email or password."))
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"token": token, "user": user}))
}

// GoogleStart kicks off the OAuth authorization-code flow: it stores a random
// CSRF state in an httpOnly cookie and redirects the browser to Google.
func (h *AuthHandler) GoogleStart(c *gin.Context) {
	if !h.service.GoogleConfigured() {
		middleware.JSON(c, http.StatusServiceUnavailable, contracts.Fail(contracts.AUTH01003, "Google sign-in is not configured on this deployment."))
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
