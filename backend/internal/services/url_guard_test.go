package services

import "testing"

func TestValidateExternalURL(t *testing.T) {
	allowed := []string{"https://example.com", "http://8.8.8.8/docs"}
	for _, raw := range allowed {
		if _, err := ValidateExternalURL(raw); err != nil {
			t.Fatalf("expected %s to pass: %v", raw, err)
		}
	}
	blocked := []string{"file:///etc/passwd", "http://localhost", "http://127.0.0.1", "http://10.0.0.1", "http://[::1]", "http://service.internal"}
	for _, raw := range blocked {
		if _, err := ValidateExternalURL(raw); err == nil {
			t.Fatalf("expected %s to be blocked", raw)
		}
	}
}
