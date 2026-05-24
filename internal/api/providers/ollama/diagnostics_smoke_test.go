package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnoseOllama_TextSmokeObservesUsageAndZeroCost(t *testing.T) {
	var received OllamaRequest
	server := newOllamaDiagnosticServer(t, []string{"qwen2.5-coder:7b"}, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOllamaDiagnosticJSONL(w,
			OllamaStreamResponse{Message: OllamaMessageContent{Content: "xelyon ollama doctor ok"}},
			OllamaStreamResponse{Done: true, PromptEvalCount: 10, EvalCount: 5},
		)
	})
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "qwen2.5-coder:7b",
		CatalogModel: "qwen2.5-coder:7b",
		RunSmoke:     true,
		TextSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if received.Model != "qwen2.5-coder:7b" || received.Options == nil || received.Options.NumPredict != defaultOllamaDiagnosticSmokeMaxOutputTokens {
		t.Fatalf("request = %#v, want smoke model and num_predict override", received)
	}
	if len(received.Tools) != 0 || received.ToolChoice != "" {
		t.Fatalf("text smoke request should not include tools: %#v", received)
	}
	if report.Smoke == nil || !report.Smoke.UsageObserved {
		t.Fatalf("Smoke = %#v, want observed usage", report.Smoke)
	}
	if report.Smoke.Usage.InputTokens != 10 || report.Smoke.Usage.OutputTokens != 5 {
		t.Fatalf("Smoke usage = %+v, want normalized usage", report.Smoke.Usage)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD != 0 {
		t.Fatalf("Smoke cost = %+v, want local zero cost", report.Smoke.Cost)
	}
	for _, name := range []string{"smoke", "usage", "cost"} {
		check, ok := ollamaDiagnosticCheckByName(report, name)
		if !ok || check.Status != DiagnosticStatusOK {
			t.Fatalf("%s check = %#v, %v; want ok", name, check, ok)
		}
	}
}

func TestDiagnoseOllama_ToolSmokeRequiresToolCall(t *testing.T) {
	var received OllamaRequest
	server := newOllamaDiagnosticServer(t, []string{"qwen2.5-coder:7b"}, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOllamaDiagnosticJSONL(w,
			OllamaStreamResponse{
				Message: OllamaMessageContent{ToolCalls: []api.OpenAIToolCall{{
					ID:   "call_1",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      ollamaDiagnosticSmokeToolName,
						Arguments: `{"value":"ollama-tool-ok"}`,
					},
				}}},
			},
			OllamaStreamResponse{Done: true, PromptEvalCount: 12, EvalCount: 3},
		)
	})
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "qwen2.5-coder:7b",
		CatalogModel: "qwen2.5-coder:7b",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if len(received.Tools) != 1 || received.ToolChoice != ollamaDiagnosticSmokeToolName {
		t.Fatalf("request = %#v, want forced diagnostic tool payload", received)
	}
	check, ok := ollamaDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusOK {
		t.Fatalf("tool_smoke check = %#v, %v; want ok", check, ok)
	}
}

func TestDiagnoseOllama_ToolSmokeFailsWithoutToolCall(t *testing.T) {
	server := newOllamaDiagnosticServer(t, []string{"qwen2.5-coder:7b"}, func(w http.ResponseWriter, r *http.Request) {
		writeOllamaDiagnosticJSONL(w,
			OllamaStreamResponse{Message: OllamaMessageContent{Content: "plain text"}},
			OllamaStreamResponse{Done: true},
		)
	})
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "qwen2.5-coder:7b",
		CatalogModel: "qwen2.5-coder:7b",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want tool smoke failure: %#v", report.Checks)
	}
	check, ok := ollamaDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusFail {
		t.Fatalf("tool_smoke check = %#v, %v; want fail", check, ok)
	}
	smoke, ok := ollamaDiagnosticCheckByName(report, "smoke")
	if !ok || smoke.Status != DiagnosticStatusFail || !strings.Contains(smoke.Message, "function calling smoke was not accepted") {
		t.Fatalf("smoke check = %#v, %v; want common feature classification", smoke, ok)
	}
}

func TestDiagnoseOllama_FunctionCallingDisabledSkipsToolAndRunsTextFallback(t *testing.T) {
	requests := 0
	var received map[string]any
	server := newOllamaDiagnosticServer(t, []string{"qwen2.5-coder:7b"}, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeOllamaDiagnosticJSONL(w,
			OllamaStreamResponse{Message: OllamaMessageContent{Content: "fallback ok"}},
			OllamaStreamResponse{Done: true, PromptEvalCount: 8, EvalCount: 2},
		)
	})
	defer server.Close()

	t.Setenv(ollamaBaseURLEnv, server.URL)
	t.Setenv("OLLAMA_FUNCTION_CALLING", "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "qwen2.5-coder:7b",
		CatalogModel: "qwen2.5-coder:7b",
		RunSmoke:     true,
		ToolSmoke:    true,
	})
	if report.HasFailures() {
		t.Fatalf("Diagnose() has failures: %#v", report.Checks)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want only text fallback network request", requests)
	}
	if _, ok := received["tools"]; ok {
		t.Fatalf("fallback request should not include tools: %#v", received["tools"])
	}
	check, ok := ollamaDiagnosticCheckByName(report, "tool_smoke")
	if !ok || check.Status != DiagnosticStatusWarn {
		t.Fatalf("tool_smoke check = %#v, %v; want warn skip", check, ok)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 || !report.Smoke.Requests[1].Skipped {
		t.Fatalf("Smoke requests = %#v, want text fallback plus skipped tool", report.Smoke)
	}
}
