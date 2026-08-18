package middleware

import (
	"intellisearch/internal/contracts"
	"intellisearch/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

const UserIDKey = "userID"
const UserRoleKey = "userRole"

func RequireAuth(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if raw == c.GetHeader("Authorization") || raw == "" {
			RespondError(c, http.StatusUnauthorized, contracts.AUTH01002, "Your session is invalid or has expired.", "missing bearer token", nil)
			c.Abort()
			return
		}
		claims, err := auth.Parse(raw)
		id, parseErr := uuid.Parse(claims.UserID)
		if err != nil || parseErr != nil {
			RespondError(c, http.StatusUnauthorized, contracts.AUTH01002, "Your session is invalid or has expired.", "invalid bearer token", err)
			c.Abort()
			return
		}
		c.Set(UserIDKey, id)
		c.Set(UserRoleKey, claims.Role)
		c.Next()
	}
}
func RequireSuperOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(UserRoleKey) != "super_owner" {
			RespondError(c, http.StatusForbidden, contracts.AUTH02001, "Super Owner access is required.", "super owner only", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
