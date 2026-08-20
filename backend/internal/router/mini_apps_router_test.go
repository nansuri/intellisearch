package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMiniAppsCRUDAndPrivacy exercises the mini-app endpoints end to end:
// owner CRUD scoped to the signed-in user, the public launcher listing only
// public apps, and the slug runner enforcing visibility.
func TestMiniAppsCRUDAndPrivacy(t *testing.T) {
	server, _ := adminTestMux(t)
	janeToken := loginToken(t, server, "jane@example.com", "jane-pass")
	ownerSlug := createMiniApp(t, server, janeToken, "Jane Todo", "private")

	// Owner sees the app in their list.
	status, payload := call(t, server, http.MethodGet, "/api/v1/me/mini-apps", janeToken, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(ownerSlug)) {
		t.Fatalf("owner list should include the app: %d %s", status, payload)
	}

	// Private app is not in the public list.
	status, payload = call(t, server, http.MethodGet, "/api/v1/mini-apps", "", nil)
	if status != http.StatusOK || bytes.Contains(payload, []byte(ownerSlug)) {
		t.Fatalf("public list must not leak private apps: %d %s", status, payload)
	}

	// Anonymous run of a private app is denied; the owner can run it.
	status, payload = call(t, server, http.MethodGet, "/api/v1/mini-apps/"+ownerSlug, "", nil)
	if status != http.StatusForbidden || !bytes.Contains(payload, []byte("MINI03001")) {
		t.Fatalf("expected 403 MINI03001 for anonymous run: %d %s", status, payload)
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/mini-apps/"+ownerSlug, janeToken, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte("Jane Todo")) {
		t.Fatalf("owner should run their private app: %d %s", status, payload)
	}

	// Making the app public surfaces it in the launcher (no source) and lets
	// anonymous visitors run it.
	status, payload = call(t, server, http.MethodPatch, "/api/v1/me/mini-apps/"+miniAppIDFromPayload(t, payload), janeToken, []byte(`{"visibility":"public","name":"Jane Todo"}`))
	if status != http.StatusOK {
		t.Fatalf("publish failed: %d %s", status, payload)
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/mini-apps", "", nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(ownerSlug)) {
		t.Fatalf("public app missing from launcher: %d %s", status, payload)
	}
	if html := appHTMLFromPayload(t, payload); strings.Contains(html, "<ul") {
		t.Fatal("launcher must not leak app source")
	}
	status, payload = call(t, server, http.MethodGet, "/api/v1/mini-apps/"+ownerSlug, "", nil)
	if status != http.StatusOK || !strings.Contains(appHTMLFromPayload(t, payload), "<ul") {
		t.Fatalf("anonymous run of public app failed: %d %s", status, payload)
	}

	// Deleting a missing app yields 404; deleting the app succeeds.
	status, payload = call(t, server, http.MethodDelete, "/api/v1/me/mini-apps/"+miniAppIDFromPayload(t, payload), janeToken, nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"deleted":true`)) {
		t.Fatalf("delete failed: %d %s", status, payload)
	}
	status, _ = call(t, server, http.MethodGet, "/api/v1/mini-apps/"+ownerSlug, "", nil)
	if status == http.StatusOK {
		t.Fatal("deleted app should no longer be runnable")
	}
}

// TestMiniAppsApiDocsAndAuth exercises the DB-stored API reference endpoints
// and the auth gate on the /me surface.
func TestMiniAppsApiDocsAndAuth(t *testing.T) {
	server, _ := adminTestMux(t)

	// The API reference and its markdown export are public.
	status, payload := call(t, server, http.MethodGet, "/api/v1/mini-apps/api-docs", "", nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"sections"`)) {
		t.Fatalf("api-docs failed: %d %s", status, payload)
	}
	if !bytes.Contains(payload, []byte("AI (Ask)")) {
		t.Fatalf("api-docs should include the AI section: %s", payload)
	}
	status, md := call(t, server, http.MethodGet, "/api/v1/mini-apps/api-docs/ai.md", "", nil)
	if status != http.StatusOK || !bytes.Contains(md, []byte("# Mini Apps API")) || !bytes.Contains(md, []byte("/api/v1/ask")) {
		t.Fatalf("markdown export failed: %d %s", status, md)
	}

	// The CRUD/generate surface requires a session.
	status, _ = call(t, server, http.MethodGet, "/api/v1/me/mini-apps", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous /me/mini-apps should be 401, got %d", status)
	}
	status, _ = call(t, server, http.MethodPost, "/api/v1/me/mini-apps/generate", "", []byte(`{"prompt":"a counter"}`))
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous generate should be 401, got %d", status)
	}
	// Signed-in (any role, incl. general users) can generate → routes through
	// the AI handler fake and creates a private draft.
	janeToken := loginToken(t, server, "jane@example.com", "jane-pass")
	status, payload = call(t, server, http.MethodPost, "/api/v1/me/mini-apps/generate", janeToken, []byte(`{"prompt":"a counter"}`))
	if status != http.StatusOK || !bytes.Contains(payload, []byte(`"Fake app"`)) || !bytes.Contains(payload, []byte(`"private"`)) {
		t.Fatalf("generate failed: %d %s", status, payload)
	}
}

func createMiniApp(t *testing.T, server *httptest.Server, token, name, visibility string) string {
	t.Helper()
	body := `{"name":"` + name + `","icon":"📝","html":"<ul id=\"t\"></ul>","css":"ul{color:blue}","js":"","visibility":"` + visibility + `"}`
	status, payload := call(t, server, http.MethodPost, "/api/v1/me/mini-apps", token, []byte(body))
	if status != http.StatusOK || !bytes.Contains(payload, []byte(name)) {
		t.Fatalf("create mini app failed: %d %s", status, payload)
	}
	return miniAppSlugFromPayload(t, payload)
}

func miniAppSlugFromPayload(t *testing.T, payload []byte) string {
	t.Helper()
	var envelope struct {
		Data struct {
			Slug string `json:"slug"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Data.Slug == "" {
		t.Fatalf("invalid create payload: %s", payload)
	}
	return envelope.Data.Slug
}

func miniAppIDFromPayload(t *testing.T, payload []byte) string {
	t.Helper()
	var envelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Data.ID == "" {
		t.Fatalf("invalid app payload: %s", payload)
	}
	return envelope.Data.ID
}

func appHTMLFromPayload(t *testing.T, payload []byte) string {
	t.Helper()
	var envelope struct {
		Data struct {
			HTML string `json:"html"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("invalid app payload: %s", payload)
	}
	return envelope.Data.HTML
}