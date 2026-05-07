package kimi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnose_FailsForMissingKeyAndInvalidAPIURL(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "")
	t.Setenv(kimiAPIURLEnv, "://bad")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want true: %#v", report.Checks)
	}
	if !hasKimiDiagnosticCheck(report, "auth", DiagnosticStatusFail) {
		t.Fatalf("missing auth failure: %#v", report.Checks)
	}
	if !hasKimiDiagnosticCheck(report, "api_url", DiagnosticStatusFail) {
		t.Fatalf("missing api_url failure: %#v", report.Checks)
	}
}

func TestDiagnose_ReportsRegistrationModelUnsupportedAndPromptCacheKey(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, "")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Model != defaultKimiModel {
		t.Fatalf("Model = %q, want %q", report.Model, defaultKimiModel)
	}
	if !report.PromptCacheKeyPresent {
		t.Fatal("PromptCacheKeyPresent = false, want true")
	}
	for _, unsupported := range report.UnsupportedFeatures {
		if unsupported == "image input" {
			t.Fatalf("UnsupportedFeatures = %v, want image input removed", report.UnsupportedFeatures)
		}
		if unsupported == "built-in web_search" {
			t.Fatalf("UnsupportedFeatures = %v, want built-in web_search removed", report.UnsupportedFeatures)
		}
	}
	for _, want := range []struct {
		name   string
		status DiagnosticStatus
	}{
		{"provider_registration", DiagnosticStatusOK},
		{"model", DiagnosticStatusOK},
		{"image_input", DiagnosticStatusOK},
		{"unsupported_features", DiagnosticStatusInfo},
		{"prompt_cache_key", DiagnosticStatusOK},
	} {
		if !hasKimiDiagnosticCheck(report, want.name, want.status) {
			t.Fatalf("missing %s/%s check: %#v", want.name, want.status, report.Checks)
		}
	}
}

func TestDiagnose_SmokeRecordsUsageAndPromptCacheKey(t *testing.T) {
	var captured []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		captured = append(captured, body)
		writeKimiDiagnosticSSE(t, w, `{"choices":[{"delta":{"content":"ok"}}]}`, `{"choices":[{"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":7,"completion_tokens":3,"cached_tokens":2}}]}`)
	}))
	defer server.Close()

	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          config.DefaultConfig(),
		RunSmoke:        true,
		MaxOutputTokens: 8,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.Ran {
		t.Fatalf("Smoke = %#v, want ran smoke", report.Smoke)
	}
	if !report.Smoke.UsageObserved || report.Smoke.CachedInputTokens != 6 {
		t.Fatalf("Smoke usage = observed %t cached %d, want observed cached=6", report.Smoke.UsageObserved, report.Smoke.CachedInputTokens)
	}
	if len(captured) != 3 {
		t.Fatalf("captured request count = %d, want 3", len(captured))
	}
	firstKey, _ := captured[0]["prompt_cache_key"].(string)
	secondKey, _ := captured[1]["prompt_cache_key"].(string)
	if firstKey == "" || firstKey != secondKey {
		t.Fatalf("prompt_cache_key first=%q second=%q, want non-empty equal keys", firstKey, secondKey)
	}
	if captured[0]["max_completion_tokens"] != float64(8) {
		t.Fatalf("max_completion_tokens = %#v, want 8", captured[0]["max_completion_tokens"])
	}
	thinking, ok := captured[0]["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("first thinking = %#v, want disabled", captured[0]["thinking"])
	}
	thinking, ok = captured[2]["thinking"].(map[string]any)
	if !ok || thinking["type"] != "enabled" || thinking["keep"] != "all" {
		t.Fatalf("third thinking = %#v, want enabled keep=all", captured[2]["thinking"])
	}
}

func TestDiagnose_ImageSmokeBuildsMultimodalPayload(t *testing.T) {
	var captured []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		captured = append(captured, body)
		writeKimiDiagnosticSSE(t, w, `{"choices":[{"delta":{"content":"image ok"}}]}`, `{"choices":[{"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":9,"completion_tokens":3,"cached_tokens":1}}]}`)
	}))
	defer server.Close()

	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("KIMI_FUNCTION_CALLING", "1")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          config.DefaultConfig(),
		RunSmoke:        true,
		ImageSmoke:      true,
		MaxOutputTokens: 8,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.Ran || !report.Smoke.ImagePayload {
		t.Fatalf("Smoke = %#v, want image payload smoke", report.Smoke)
	}
	if !report.Smoke.UsageObserved || report.Smoke.CachedInputTokens != 1 {
		t.Fatalf("Smoke usage = observed %t cached %d, want observed cached=1", report.Smoke.UsageObserved, report.Smoke.CachedInputTokens)
	}
	if !hasKimiDiagnosticCheck(report, "image_smoke", DiagnosticStatusOK) {
		t.Fatalf("missing image_smoke OK check: %#v", report.Checks)
	}
	if len(captured) != 1 {
		t.Fatalf("captured request count = %d, want only image smoke request", len(captured))
	}
	if _, ok := captured[0]["tools"]; ok {
		t.Fatalf("tools = %#v, want absent for image smoke", captured[0]["tools"])
	}
	if key, _ := captured[0]["prompt_cache_key"].(string); key == "" {
		t.Fatalf("prompt_cache_key = %#v, want non-empty", captured[0]["prompt_cache_key"])
	}
	if captured[0]["max_completion_tokens"] != float64(8) {
		t.Fatalf("max_completion_tokens = %#v, want 8", captured[0]["max_completion_tokens"])
	}
	messages, ok := captured[0]["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v, want system + multimodal user", captured[0]["messages"])
	}
	user, ok := messages[1].(map[string]any)
	if !ok || user["role"] != "user" {
		t.Fatalf("user message = %#v, want role user", messages[1])
	}
	content, ok := user["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content = %#v, want text + image parts", user["content"])
	}
	imagePart, ok := content[1].(map[string]any)
	if !ok || imagePart["type"] != "image_url" {
		t.Fatalf("image part = %#v, want image_url", content[1])
	}
	imageURL, ok := imagePart["image_url"].(map[string]any)
	if !ok || imageURL["url"] != "data:image/png;base64,"+kimiDiagnosticPNGBase64 {
		t.Fatalf("image_url = %#v, want diagnostic PNG data URL", imagePart["image_url"])
	}
	if report.Smoke.Requests[0].Content != "image ok" {
		t.Fatalf("image smoke content = %q, want image ok", report.Smoke.Requests[0].Content)
	}
}

func TestDiagnose_WebSearchSmokeBuildsBuiltinPayload(t *testing.T) {
	var captured []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		captured = append(captured, body)
		switch len(captured) {
		case 1:
			writeKimiDiagnosticSSE(
				t,
				w,
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_web","type":"builtin_function","function":{"name":"$web_search","arguments":"{\"query\":\"doctor\"}"}}]}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"cached_tokens":1}}`,
			)
		case 2:
			writeKimiDiagnosticSSE(
				t,
				w,
				`{"choices":[{"delta":{"content":"web search ok"}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":4,"cached_tokens":2}}`,
			)
		default:
			t.Fatalf("unexpected request count %d", len(captured))
		}
	}))
	defer server.Close()

	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          config.DefaultConfig(),
		RunSmoke:        true,
		WebSearchSmoke:  true,
		MaxOutputTokens: 8,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.Ran || !report.Smoke.WebSearchPayload {
		t.Fatalf("Smoke = %#v, want web search payload smoke", report.Smoke)
	}
	if report.Smoke.Content != "web search ok" {
		t.Fatalf("Smoke content = %q, want web search ok", report.Smoke.Content)
	}
	if !report.Smoke.UsageObserved || report.Smoke.CachedInputTokens != 3 {
		t.Fatalf("Smoke usage = observed %t cached %d, want observed cached=3", report.Smoke.UsageObserved, report.Smoke.CachedInputTokens)
	}
	if !hasKimiDiagnosticCheck(report, "web_search_smoke", DiagnosticStatusOK) {
		t.Fatalf("missing web_search_smoke OK check: %#v", report.Checks)
	}
	if len(captured) != 2 {
		t.Fatalf("captured request count = %d, want 2", len(captured))
	}
	first := captured[0]
	if first["max_completion_tokens"] != float64(8) {
		t.Fatalf("max_completion_tokens = %#v, want 8", first["max_completion_tokens"])
	}
	thinking, ok := first["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want disabled", first["thinking"])
	}
	tools, ok := first["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want one builtin tool", first["tools"])
	}
	tool, ok := tools[0].(map[string]any)
	if !ok || tool["type"] != "builtin_function" {
		t.Fatalf("tool = %#v, want builtin_function", tools[0])
	}
	function, ok := tool["function"].(map[string]any)
	if !ok || function["name"] != "$web_search" {
		t.Fatalf("tool.function = %#v, want $web_search", tool["function"])
	}
	if _, ok := first["tool_choice"]; ok {
		t.Fatalf("tool_choice = %#v, want omitted", first["tool_choice"])
	}
	if key, _ := first["prompt_cache_key"].(string); key == "" {
		t.Fatalf("prompt_cache_key = %#v, want non-empty", first["prompt_cache_key"])
	}
	secondMessages, ok := captured[1]["messages"].([]any)
	if !ok || len(secondMessages) != 4 {
		t.Fatalf("second messages = %#v, want tool loop replay", captured[1]["messages"])
	}
	toolMessage, ok := secondMessages[3].(map[string]any)
	if !ok || toolMessage["role"] != "tool" || toolMessage["name"] != "$web_search" || toolMessage["content"] != `{"query":"doctor"}` {
		t.Fatalf("tool replay message = %#v, want exact Kimi web_search tool result", secondMessages[3])
	}
}

func TestDiagnose_ImageAndWebSearchSmokeDoesNotCompareCrossPayloadCacheKeys(t *testing.T) {
	var captured []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		captured = append(captured, body)
		switch len(captured) {
		case 1:
			writeKimiDiagnosticSSE(t, w, `{"choices":[{"delta":{"content":"image ok"}}]}`, `{"choices":[{"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":9,"completion_tokens":3,"cached_tokens":1}}]}`)
		case 2:
			writeKimiDiagnosticSSE(
				t,
				w,
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_web","type":"builtin_function","function":{"name":"$web_search","arguments":"{\"query\":\"doctor\"}"}}]}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"cached_tokens":1}}`,
			)
		case 3:
			writeKimiDiagnosticSSE(
				t,
				w,
				`{"choices":[{"delta":{"content":"web search ok"}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":4,"cached_tokens":2}}`,
			)
		default:
			t.Fatalf("unexpected request count %d", len(captured))
		}
	}))
	defer server.Close()

	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          config.DefaultConfig(),
		RunSmoke:        true,
		ImageSmoke:      true,
		WebSearchSmoke:  true,
		MaxOutputTokens: 8,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for image+web smoke without text cache pair: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.ImagePayload || !report.Smoke.WebSearchPayload {
		t.Fatalf("Smoke = %#v, want image and web search payload smoke", report.Smoke)
	}
	if len(captured) != 3 {
		t.Fatalf("captured request count = %d, want image + web search tool loop", len(captured))
	}
	imageKey, _ := captured[0]["prompt_cache_key"].(string)
	webKey, _ := captured[1]["prompt_cache_key"].(string)
	if imageKey == "" || webKey == "" || imageKey == webKey {
		t.Fatalf("prompt_cache_key image=%q web=%q, want different non-empty keys", imageKey, webKey)
	}
}

func TestDiagnose_SmokeFailsForEmptyNonToolContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeKimiDiagnosticSSE(t, w, `{"choices":[{"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":7,"completion_tokens":0}}]}`)
	}))
	defer server.Close()

	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:   config.DefaultConfig(),
		RunSmoke: true,
	})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want true for empty smoke content: %#v", report.Checks)
	}
	if !hasKimiDiagnosticCheck(report, "smoke", DiagnosticStatusFail) {
		t.Fatalf("missing smoke failure: %#v", report.Checks)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) == 0 || strings.TrimSpace(report.Smoke.Requests[0].Content) != "" {
		t.Fatalf("Smoke = %#v, want empty first request content recorded", report.Smoke)
	}
}

func TestDiagnose_ToolSmokeIncludesDummyToolPayload(t *testing.T) {
	var toolRequest map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["tools"]; ok {
			toolRequest = body
			writeKimiDiagnosticSSE(
				t,
				w,
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_probe","type":"function","function":{"name":"xelyon_kimi_doctor_probe","arguments":"{\"value\":\"kimi-tool-ok\"}"}}]}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"tool_calls","usage":{"prompt_tokens":8,"completion_tokens":4,"cached_tokens":0}}]}`,
			)
			return
		}
		writeKimiDiagnosticSSE(t, w, `{"choices":[{"delta":{"content":"ok"}}]}`, `{"choices":[{"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":7,"completion_tokens":3}}]}`)
	}))
	defer server.Close()

	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("KIMI_FUNCTION_CALLING", "1")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:    config.DefaultConfig(),
		RunSmoke:  true,
		ToolSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.ToolPayload {
		t.Fatalf("Smoke = %#v, want tool payload smoke", report.Smoke)
	}
	if !hasKimiDiagnosticCheck(report, "tool_smoke", DiagnosticStatusOK) {
		t.Fatalf("missing tool_smoke OK check: %#v", report.Checks)
	}
	tools, ok := toolRequest["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want one diagnostic tool", toolRequest["tools"])
	}
	toolChoice, ok := toolRequest["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice = %#v, want forced function choice", toolRequest["tool_choice"])
	}
	function, ok := toolChoice["function"].(map[string]any)
	if !ok || function["name"] != diagnosticSmokeToolName {
		t.Fatalf("tool_choice.function = %#v, want %s", toolChoice["function"], diagnosticSmokeToolName)
	}
}

func TestDiagnose_SmokeTransientErrorWarns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"temporary"}`))
	}))
	defer server.Close()

	t.Setenv(kimiAPIKeyEnv, "moonshot-key")
	t.Setenv(kimiAPIURLEnv, server.URL+"/v1/chat/completions")
	t.Setenv("XELYON_MODEL", "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:   config.DefaultConfig(),
		RunSmoke: true,
	})
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for transient smoke warning: %#v", report.Checks)
	}
	if !hasKimiDiagnosticCheck(report, "smoke", DiagnosticStatusWarn) {
		t.Fatalf("missing transient smoke warning: %#v", report.Checks)
	}
}

func writeKimiDiagnosticSSE(t *testing.T, w http.ResponseWriter, chunks ...string) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	for _, chunk := range chunks {
		if _, err := fmt.Fprintf(w, "data: %s\n\n", chunk); err != nil {
			t.Fatalf("write chunk: %v", err)
		}
	}
	if _, err := fmt.Fprint(w, "data: [DONE]\n\n"); err != nil {
		t.Fatalf("write done: %v", err)
	}
}

func hasKimiDiagnosticCheck(report DiagnosticReport, name string, status DiagnosticStatus) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}

func TestIsTransientKimiSmokeError(t *testing.T) {
	for _, errText := range []string{
		"rate limit exceeded (429)",
		"API error (503): temporary",
		"request timeout",
		"context deadline exceeded",
	} {
		if !isTransientKimiSmokeError(fmt.Errorf("%s", errText)) {
			t.Fatalf("isTransientKimiSmokeError(%q) = false, want true", errText)
		}
	}
	if isTransientKimiSmokeError(fmt.Errorf("API error (401): unauthorized")) {
		t.Fatal("isTransientKimiSmokeError(401) = true, want false")
	}
	if strings.TrimSpace(diagnosticSmokeToolName) == "" {
		t.Fatal("diagnosticSmokeToolName is empty")
	}
}
