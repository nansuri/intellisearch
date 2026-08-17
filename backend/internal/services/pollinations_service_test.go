package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPollinationsAccountParsesBalanceProfileKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/account/balance":
			_, _ = w.Write([]byte(`{"balance": 42.5}`))
		case "/account/profile":
			_, _ = w.Write([]byte(`{"githubUsername":"dev","image":null,"communityEndpointsAllowed":false,"name":"Dev","email":"dev@example.com"}`))
		case "/account/key":
			_, _ = w.Write([]byte(`{"valid":true,"type":"secret","name":"mykey","expiresAt":null,"expiresIn":null,"permissions":{"models":null,"account":["usage"]},"pollenBudget":null,"rateLimitEnabled":false,"userId":"u1"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	service := NewPollinationsService("https://media.pollinations.ai")
	balance, profile, key, err := service.Account(context.Background(), server.URL, "sk-test")
	if err != nil {
		t.Fatal(err)
	}
	if balance != 42.5 {
		t.Fatalf("unexpected balance %v", balance)
	}
	if profile == nil || profile.GithubUsername != "dev" || profile.Name == nil || *profile.Name != "Dev" {
		t.Fatalf("unexpected profile %#v", profile)
	}
	if key == nil || !key.Valid || key.Type != "secret" || len(key.Permissions.Account) != 1 || key.Permissions.Account[0] != "usage" {
		t.Fatalf("unexpected key info %#v", key)
	}
}

func TestPollinationsAccountUnauthorizedKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	service := NewPollinationsService("https://media.pollinations.ai")
	if _, _, _, err := service.Account(context.Background(), server.URL, "sk-bad"); err != ErrPollinationsUnauthorized {
		t.Fatalf("expected ErrPollinationsUnauthorized, got %v", err)
	}
}

func TestPollinationsUsageAndDailyUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/account/usage":
			_, _ = w.Write([]byte(`{"usage":[{"timestamp":"2026-08-17 10:00:00","type":"generate.text","model":"openai","api_key":"main","meter_source":"tier","input_text_tokens":100,"output_text_tokens":50,"cost_usd":0.0012,"response_time_ms":340}],"count":1}`))
		case "/account/usage/daily":
			_, _ = w.Write([]byte(`{"usage":[{"date":"2026-08-17","api_key":"main","model":"openai","meter_source":"tier","requests":5,"cost_usd":0.006}],"count":1}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	service := NewPollinationsService("https://media.pollinations.ai")
	usage, err := service.Usage(context.Background(), server.URL, "sk-test", 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(usage) != 1 || usage[0].Model == nil || *usage[0].Model != "openai" || usage[0].InputTextTokens != 100 || usage[0].CostUSD != 0.0012 {
		t.Fatalf("unexpected usage %#v", usage)
	}
	daily, err := service.DailyUsage(context.Background(), server.URL, "sk-test", 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(daily) != 1 || daily[0].Date != "2026-08-17" || daily[0].Requests != 5 {
		t.Fatalf("unexpected daily usage %#v", daily)
	}
}

func TestPollinationsModelsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"openai","object":"model","created":1700000000},{"id":"flux","object":"model","created":1700000001}]}`))
	}))
	t.Cleanup(server.Close)
	service := NewPollinationsService("https://media.pollinations.ai")
	models, err := service.Models(context.Background(), server.URL, "sk-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "openai" || models[1].ID != "flux" {
		t.Fatalf("unexpected models %#v", models)
	}
}

func TestPollinationsUploadMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("expected multipart body: %v", err)
		}
		if r.MultipartForm.File["file"] == nil || len(r.MultipartForm.File["file"]) != 1 {
			t.Fatal("expected a file part")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"abc","url":"https://media.pollinations.ai/abc","contentType":"image/png","size":1234}`))
	}))
	t.Cleanup(server.Close)
	service := NewPollinationsService(server.URL)
	result, err := service.Upload(context.Background(), "sk-test", "chart.png", "image/png", []byte("PNG DATA"))
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "abc" || result.URL != "https://media.pollinations.ai/abc" || result.Size != 1234 {
		t.Fatalf("unexpected upload result %#v", result)
	}
}

func TestPollinationsUploadRejectsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	service := NewPollinationsService(server.URL)
	if _, err := service.Upload(context.Background(), "sk-bad", "a.png", "image/png", []byte("x")); err != ErrPollinationsUnauthorized {
		t.Fatalf("expected ErrPollinationsUnauthorized, got %v", err)
	}
}
