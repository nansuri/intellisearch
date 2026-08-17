package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"intellisearch/internal/contracts"
)

func JSON(c *gin.Context, status int, body contracts.Envelope) { c.JSON(status, body) }

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 && !c.Writer.Written() { JSON(c, http.StatusInternalServerError, contracts.Fail(contracts.AISY01001, "The service is temporarily unavailable.")) }
	}
}

