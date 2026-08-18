package middleware

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"testing"
)

func TestErrorLevelMapping(t *testing.T) {
	cases := []struct {
		status int
		want   logrus.Level
	}{
		{http.StatusInternalServerError, logrus.ErrorLevel},
		{http.StatusBadGateway, logrus.ErrorLevel},
		{http.StatusServiceUnavailable, logrus.ErrorLevel},
		{http.StatusGatewayTimeout, logrus.ErrorLevel},
		{http.StatusUnauthorized, logrus.WarnLevel},
		{http.StatusForbidden, logrus.WarnLevel},
		{http.StatusBadRequest, logrus.WarnLevel},
		{http.StatusUnprocessableEntity, logrus.WarnLevel},
		{http.StatusNotFound, logrus.InfoLevel},
		{http.StatusTooManyRequests, logrus.InfoLevel},
	}
	for _, tc := range cases {
		if got := errorLevel(tc.status); got != tc.want {
			t.Errorf("errorLevel(%d) = %v, want %v", tc.status, got, tc.want)
		}
	}
}