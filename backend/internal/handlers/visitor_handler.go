package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/repositories"
)

// VisitorHandler owns the public, best-effort visit-tracking endpoint used by
// the sign-in page: it records that an (anonymous) visitor opened the register
// tab so the Owner Control Panel can measure registration interest.
type VisitorHandler struct {
	registerVisit *repositories.RegisterVisitRepository
}

func NewVisitorHandler(registerVisit *repositories.RegisterVisitRepository) *VisitorHandler {
	return &VisitorHandler{registerVisit: registerVisit}
}

// TrackRegisterVisit records a unique visitor who opened the register page and
// replies with whether this was a new record. Reusing the same identity
// (httpOnly cookie or X-Visitor-ID header, same as the anonymous AI guest
// token) makes replays no-ops, so the count reflects real visitors. The call is
// always best-effort: a tracking failure must never break the register page.
func (h *VisitorHandler) TrackRegisterVisit(c *gin.Context) {
	raw := c.GetHeader("X-Visitor-ID")
	if raw == "" {
		// Ignore the cookie read error: an absent cookie just means "new visitor".
		raw, _ = c.Cookie(visitorCookie)
	}
	visitorID := uuid.Nil
	if parsed, err := uuid.Parse(raw); err == nil {
		visitorID = parsed
	}
	if visitorID == uuid.Nil {
		visitorID = uuid.New()
	}
	ip := c.ClientIP()
	if ip == "" {
		ip = "unknown"
	}
	_, created, err := h.registerVisit.Claim(visitorID, repositories.HashIP(ip))
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, contracts.STTS01002, "The visit could not be recorded.", "register-page visit claim failed", err)
		return
	}
	// Mirror the same identity cookie the AI handler issues so a visitor who
	// later uses the AI service shares one token across both features.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(visitorCookie, visitorID.String(), 31536000, "/", "", false, true)
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"recorded": true, "new": created}))
}
