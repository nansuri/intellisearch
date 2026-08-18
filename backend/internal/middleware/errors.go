package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"intellisearch/internal/contracts"
)

// errorLevel maps an HTTP status to the log severity used when a request
// fails. 5xx are incidents → Error; auth/security rejections (401/403) → Warn;
// routine misses and throttling (404/429) → Info (expected traffic, not an
// incident); every other client error → Warn. This keeps error logs meaningful
// without treating ordinary 4xx traffic as incidents.
func errorLevel(status int) logrus.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return logrus.ErrorLevel
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return logrus.WarnLevel
	case status == http.StatusNotFound || status == http.StatusTooManyRequests:
		return logrus.InfoLevel
	default:
		return logrus.WarnLevel
	}
}

// RespondError writes the standard error envelope and logs the failure at the
// severity matched to its HTTP status. op is a short stable label for the
// operation (never user content), code/message are the typed error code plus
// the sanitized text sent to the browser, and err is the internal cause
// attached to the log line. Extra structured fields carry request context.
func RespondError(c *gin.Context, status int, code, message, op string, err error, fields ...logrus.Fields) {
	entry := logrus.WithFields(logrus.Fields{
		"errorCode": code,
		"route":     c.FullPath(),
		"method":    c.Request.Method,
		"op":        op,
	})
	for _, extra := range fields {
		for key, value := range extra {
			entry = entry.WithField(key, value)
		}
	}
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Log(errorLevel(status), "request failed")
	JSON(c, status, contracts.Fail(code, message))
}