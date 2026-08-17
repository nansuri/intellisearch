package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ollamaMock spins up a fake Ollama server answering /api/tags, /api/version,
// and /api/ps with the given payloads.
func ollamaMock(t *testing.T, tags, version, ps string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tags))
		case "/api/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(version))
		case "/api/ps":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(ps))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOllamaModelsListsAvailableModels(t *testing.T) {
	server := ollamaMock(t,
		`{"models":[{"name":"llama3.2:latest","size":2019393183,"details":{"parameter_size":"3.2B","quantization_level":"Q4_K_M"}},{"name":"qwen2.5:7b","size":4688953604,"details":{"parameter_size":"7.6B","quantization_level":"Q4_K_M"}}]}`,
		`{"version":"0.5.7"}`,
		`{"models":[]}`,
	)
	service := NewOllamaService()
	models, err := service.Models(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].Name != "llama3.2:latest" || models[0].ParameterSize != "3.2B" || models[0].Quantization != "Q4_K_M" {
		t.Fatalf("unexpected models %#v", models)
	}
}

func TestOllamaHealthReportsVersionAndRunningStats(t *testing.T) {
	server := ollamaMock(t,
		`{"models":[]}`,
		`{"version":"0.5.7"}`,
		`{"models":[{"name":"llama3.2:latest","size":2019393183,"size_vram":1616000000,"cpu":"99%","gpu":"0%","memory":"1.6GB/3.8GB"}]}`,
	)
	service := NewOllamaService()
	health, err := service.Health(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if health.Version != "0.5.7" || len(health.RunningModels) != 1 {
		t.Fatalf("unexpected health %#v", health)
	}
	running := health.RunningModels[0]
	if running.Name != "llama3.2:latest" || running.CPU != "99%" || running.GPU != "0%" || running.Memory != "1.6GB/3.8GB" || running.SizeVram != 1616000000 {
		t.Fatalf("unexpected running model %#v", running)
	}
}

func TestOllamaUnreachableServer(t *testing.T) {
	service := NewOllamaService()
	if _, err := service.Models(context.Background(), "http://127.0.0.1:1"); err == nil {
		t.Fatal("expected error for unreachable server")
	}
	if _, err := service.Health(context.Background(), "http://127.0.0.1:1"); err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

func TestOllamaRejectsInvalidURL(t *testing.T) {
	service := NewOllamaService()
	for _, baseURL := range []string{"", "ftp://ollama.local", "localhost:11434", "not a url"} {
		if _, err := service.Models(context.Background(), baseURL); !errors.Is(err, ErrInvalidOllamaURL) {
			t.Fatalf("expected ErrInvalidOllamaURL for %q, got %v", baseURL, err)
		}
	}
}
