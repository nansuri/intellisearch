package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrInvalidOllamaURL = errors.New("invalid ollama base URL")

// OllamaModel is one entry from Ollama's GET /api/tags.
type OllamaModel struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	ParameterSize string `json:"parameterSize,omitempty"`
	Quantization string `json:"quantization,omitempty"`
}

// OllamaRunningModel is one entry from Ollama's GET /api/ps (a loaded model).
type OllamaRunningModel struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	SizeVram int64  `json:"sizeVram"`
	CPU     string `json:"cpu,omitempty"`
	GPU     string `json:"gpu,omitempty"`
	Memory  string `json:"memory,omitempty"`
}

// OllamaHealth is the version plus the currently loaded models with their
// runtime utilization.
type OllamaHealth struct {
	Version       string               `json:"version"`
	RunningModels []OllamaRunningModel `json:"runningModels"`
}

// OllamaService talks to an Ollama server on behalf of the Owner Control
// Panel. The browser never calls Ollama directly; these requests go through the
// Go API (admin-only routes). Only http/https base URLs are accepted.
type OllamaService struct {
	client *http.Client
}

func NewOllamaService() *OllamaService {
	return &OllamaService{client: &http.Client{Timeout: 5 * time.Second}}
}

// Models lists the models available on the Ollama server at baseURL.
func (s *OllamaService) Models(ctx context.Context, baseURL string) ([]OllamaModel, error) {
	endpoint, err := ollamaEndpoint(baseURL, "/api/tags")
	if err != nil {
		return nil, err
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
			Details struct {
				ParameterSize    string `json:"parameter_size"`
				QuantizationLevel string `json:"quantization_level"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := s.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	models := make([]OllamaModel, 0, len(payload.Models))
	for _, m := range payload.Models {
		models = append(models, OllamaModel{Name: m.Name, Size: m.Size, ParameterSize: m.Details.ParameterSize, Quantization: m.Details.QuantizationLevel})
	}
	return models, nil
}

// Health reports the server version and the models currently loaded with CPU,
// GPU, and memory utilization (GET /api/ps). An unreachable server is an error.
func (s *OllamaService) Health(ctx context.Context, baseURL string) (OllamaHealth, error) {
	versionEndpoint, err := ollamaEndpoint(baseURL, "/api/version")
	if err != nil {
		return OllamaHealth{}, err
	}
	var version struct {
		Version string `json:"version"`
	}
	if err := s.getJSON(ctx, versionEndpoint, &version); err != nil {
		return OllamaHealth{}, err
	}
	health := OllamaHealth{Version: version.Version, RunningModels: []OllamaRunningModel{}}

	psEndpoint, psErr := ollamaEndpoint(baseURL, "/api/ps")
	if psErr != nil {
		return OllamaHealth{}, psErr
	}
	var ps struct {
		Models []struct {
			Name      string `json:"name"`
			Size      int64  `json:"size"`
			SizeVRAM  int64  `json:"size_vram"`
			CPU       string `json:"cpu"`
			GPU       string `json:"gpu"`
			Memory    string `json:"memory"`
		} `json:"models"`
	}
	// /api/ps is best-effort: a running server should always answer, but a
	// failure here must not mask the version/health result.
	if err := s.getJSON(ctx, psEndpoint, &ps); err == nil {
		for _, m := range ps.Models {
			health.RunningModels = append(health.RunningModels, OllamaRunningModel{Name: m.Name, Size: m.Size, SizeVram: m.SizeVRAM, CPU: m.CPU, GPU: m.GPU, Memory: m.Memory})
		}
	}
	return health, nil
}

func (s *OllamaService) getJSON(ctx context.Context, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		io.Copy(io.Discard, response.Body)
		return fmt.Errorf("ollama returned status %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<20))
	return decoder.Decode(target)
}

// ollamaEndpoint validates the base URL and appends the API path.
func ollamaEndpoint(baseURL, path string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", ErrInvalidOllamaURL
	}
	return strings.TrimRight(parsed.String(), "/") + path, nil
}
