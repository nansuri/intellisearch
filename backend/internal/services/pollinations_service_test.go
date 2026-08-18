package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPollinationsAccountBalanceShapes covers every /account/balance body the
// upstream may return: a bare JSON number, a JSON-quoted number string (with
// format=json), and an object with a "balance" field (defensive).
func TestPollinationsAccountBalanceShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"bare number", "9.5"},
		{"quoted string", `"9.5"`},
		{"object balance", `{"balance": 9.5}`},
		{"object balance quoted", `{"balance": "9.5"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/account/balance":
					_, _ = w.Write([]byte(tc.body))
				case "/account/profile":
					_, _ = w.Write([]byte(`{"githubUsername":"dev","image":null,"communityEndpointsAllowed":false}`))
				case "/account/key":
					_, _ = w.Write([]byte(`{"valid":true,"type":"secret","rateLimitEnabled":false}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer upstream.Close()

			service := NewPollinationsService("https://media.pollinations.ai")
			balance, _, _, err := service.Account(context.Background(), upstream.URL, "sk-test")
			if err != nil {
				t.Fatalf("Account failed: %v", err)
			}
			if balance != 9.5 {
				t.Fatalf("expected balance 9.5, got %v", balance)
			}
		})
	}
}

// TestPollinationsAccountStatusMapping verifies that 401, 403, and 5xx
// upstream statuses map to their distinct sentinel errors.
func TestPollinationsAccountStatusMapping(t *testing.T) {
	cases := []struct {
		name string
		code int
		want error
	}{
		{"unauthorized", http.StatusUnauthorized, ErrPollinationsUnauthorized},
		{"missing scope", http.StatusForbidden, ErrPollinationsForbidden},
		{"upstream error", http.StatusInternalServerError, ErrPollinationsUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(`{"error":"boom"}`))
			}))
			defer upstream.Close()

			service := NewPollinationsService("https://media.pollinations.ai")
			_, _, _, err := service.Account(context.Background(), upstream.URL, "sk-test")
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if tc.want == ErrPollinationsUnavailable && !strings.Contains(err.Error(), "status") {
				t.Fatalf("unavailable error should include upstream status, got %q", err)
			}
			if tc.want != ErrPollinationsUnavailable && err != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}