package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ollamaprovider "github.com/susugadx/xelyon-cli/internal/api/providers/ollama"
)

func TestRunOllamaDoctorInvocation_JSONReportsExplicitModelAndCatalogModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	server := newOllamaDoctorCommandTestServer(t, []string{"qwen2.5-coder:7b"}, nil)
	defer server.Close()
	t.Setenv("OLLAMA_BASE_URL", server.URL)

	cmd, out := newDoctorSubcommandTest(t, newOllamaDoctorCommand)

	doctorOllamaModelFlag = "qwen2.5-coder:7b"
	doctorCatalogModelFlag = "qwen2.5-coder:7b"
	doctorJSONFlag = true

	if err := runOllamaDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOllamaDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.Provider != "ollama" {
		t.Fatalf("provider = %q, want ollama", report.Provider)
	}
	if report.APIURL != server.URL {
		t.Fatalf("api_url = %q, want fake server URL", report.APIURL)
	}
	if report.Model != "qwen2.5-coder:7b" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "qwen2.5-coder:7b" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != "ollama_chat" {
		t.Fatalf("route = %q, want ollama_chat", report.Route)
	}
	for _, name := range []string{"auth", "endpoint", "installed_model", "catalog_policy"} {
		requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, name), "ok")
	}
	requireDoctorJSONCheckDetailContains(t, requireDoctorJSONCheck(t, report.Checks, "catalog_policy"), "pricing=input $0.00/M")
}

func TestRunOllamaDoctorInvocation_PrintRequestJSONDoesNotSendNetwork(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected network request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	t.Setenv("OLLAMA_BASE_URL", server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	cmd, out := newDoctorSubcommandTest(t, newOllamaDoctorCommand)

	doctorOllamaModelFlag = "qwen2.5-coder:7b"
	doctorCatalogModelFlag = "qwen2.5-coder:7b"
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runOllamaDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOllamaDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}
	if requests != 0 {
		t.Fatalf("network requests = %d, want 0", requests)
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
	requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, "auth"), "ok")
	requireDoctorJSONRequestPreviewCount(t, report.RequestPreview, 1)
	request := requireDoctorJSONRequestPreviewAt(t, report.RequestPreview, 0, "tool")
	if request.Name != "tool" || !request.ToolPayload {
		t.Fatalf("preview request = %#v, want tool payload", request)
	}
	requireDoctorJSONRequestPreviewHeader(t, request, "Content-Type", "application/json")
	body := requireDoctorJSONRequestPreviewBody[struct {
		Model   string `json:"model"`
		Stream  bool   `json:"stream"`
		Options struct {
			NumPredict int `json:"num_predict"`
		} `json:"options"`
		Tools      []any  `json:"tools"`
		ToolChoice string `json:"tool_choice"`
	}](t, request)
	if body.Model != "qwen2.5-coder:7b" || !body.Stream || body.Options.NumPredict != 64 || len(body.Tools) != 1 || body.ToolChoice != "xelyon_ollama_doctor_probe" {
		t.Fatalf("preview body = %#v, want diagnostic tool body", body)
	}
}

func TestRootCommand_OllamaDoctorCommandParsesFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	server := newOllamaDoctorCommandTestServer(t, []string{"qwen2.5-coder:7b"}, nil)
	defer server.Close()
	t.Setenv("OLLAMA_BASE_URL", server.URL)

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "ollama", "--model", "qwen2.5-coder:7b", "--catalog-model", "qwen2.5-coder:7b", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "qwen2.5-coder:7b"`) {
		t.Fatalf("output = %q, want parsed Ollama model", out.String())
	}
	if !strings.Contains(out.String(), `"catalog_model": "qwen2.5-coder:7b"`) {
		t.Fatalf("output = %q, want parsed Ollama catalog model", out.String())
	}
}

func TestRootCommand_OllamaDoctorHelpShowsMinimalFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "ollama", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	for _, want := range []string{"--model", "--catalog-model", "--smoke", "--tool-smoke", "--print-request", "--timeout", "--json", "Diagnose Ollama provider configuration"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want Ollama doctor help substring %q", out.String(), want)
		}
	}
	for _, unwanted := range []string{"--capabilities", "--require-capability", "--retention-smoke", "--image-smoke", "--web-search-smoke", "--thinking-smoke", "--print-config"} {
		if strings.Contains(out.String(), unwanted) {
			t.Fatalf("output = %q, should not contain %s", out.String(), unwanted)
		}
	}
}

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
