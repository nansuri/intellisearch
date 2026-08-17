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
			JSON(c, http.StatusUnauthorized, contracts.Fail(contracts.AUTH01002, "Your session is invalid or has expired."))
			c.Abort()
			return
		}
		claims, err := auth.Parse(raw)
		id, parseErr := uuid.Parse(claims.UserID)
		if err != nil || parseErr != nil {
			JSON(c, http.StatusUnauthorized, contracts.Fail(contracts.AUTH01002, "Your session is invalid or has expired."))
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
			JSON(c, http.StatusForbidden, contracts.Fail(contracts.AUTH02001, "Super Owner access is required."))
			c.Abort()
			return
		}
		c.Next()
	}
}
