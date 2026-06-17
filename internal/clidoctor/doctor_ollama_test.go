package clidoctor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ollamaprovider "github.com/susugadx/xelyon-cli/internal/api/providers/ollama"
)

func TestRenderOllamaDoctorTextIncludesRequestPreviewAndSmokeObservability(t *testing.T) {
	report := ollamaprovider.DiagnosticReport{
		Provider:           "ollama",
		APIURL:             "http://localhost:11434",
		Model:              "qwen2.5-coder:7b",
		ModelSource:        "test",
		CatalogModel:       "qwen2.5-coder:7b",
		CatalogModelSource: "test",
		Route:              ollamaprovider.DiagnosticRouteOllamaChat,
		RouteReason:        "Ollama provider uses the local /api/chat JSONL stream endpoint",
		Checks: []ollamaprovider.DiagnosticCheck{
			{Name: "smoke", Status: ollamaprovider.DiagnosticStatusOK, Message: "live Ollama smoke request succeeded"},
		},
		RequestPreview: &ollamaprovider.DiagnosticRequestPreview{
			Requests: []ollamaprovider.DiagnosticRequestPreviewRequest{{
				Name:    "text",
				Route:   ollamaprovider.DiagnosticRouteOllamaChat,
				Method:  "POST",
				URL:     "http://localhost:11434/api/chat",
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    map[string]any{"model": "qwen2.5-coder:7b", "stream": true, "options": map[string]any{"num_predict": 64}},
			}},
		},
		Smoke: &ollamaprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         ollamaprovider.DiagnosticRouteOllamaChat,
			Content:       "xelyon ollama doctor ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage: ollamaprovider.DiagnosticSmokeUsage{
				InputTokens:  10,
				OutputTokens: 4,
			},
			Cost: ollamaprovider.DiagnosticSmokeCost{
				USD: 0,
			},
		},
	}

	var out bytes.Buffer
	renderOllamaDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Route reason: Ollama provider uses the local /api/chat JSONL stream endpoint",
		"Request preview:",
		`"Content-Type": "application/json"`,
		"Smoke route: ollama_chat",
		"Smoke usage: input=10 cached=0 output=4 reasoning=0 cache_creation=0",
		"Smoke cost estimate: $0.00000000 USD",
	})
}

func newOllamaDoctorCommandTestServer(t *testing.T, models []string, chat http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			type ollamaTagsResponse struct {
				Models []struct {
					Name string `json:"name"`
				} `json:"models"`
			}
			response := ollamaTagsResponse{}
			for _, model := range models {
				response.Models = append(response.Models, struct {
					Name string `json:"name"`
				}{Name: model})
			}
			_ = json.NewEncoder(w).Encode(response)
		case "/api/chat":
			if chat == nil {
				t.Fatalf("unexpected chat request: %s %s", r.Method, r.URL.Path)
			}
			chat(w, r)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
}
