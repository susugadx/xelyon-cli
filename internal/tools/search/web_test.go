package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteWebSearch_UsesExecutionContextConfigForNativeProvider(t *testing.T) {
	query := "runtime-config-openai-web-search-" + t.Name()
	requestCh := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		requestCh <- req

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [
				{
					"type": "message",
					"content": [
						{
							"type": "output_text",
							"text": "Runtime-specific native web search succeeded.",
							"annotations": [
								{"type":"url_citation","title":"OpenAI Blog","url":"https://openai.com/blog"}
							]
						}
					]
				}
			]
		}`))
	}))
	defer server.Close()

	oldURL := os.Getenv("OPENAI_RESPONSES_URL")
	oldKey := os.Getenv("OPENAI_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("OPENAI_RESPONSES_URL", oldURL)
		_ = os.Setenv("OPENAI_API_KEY", oldKey)
	})
	_ = os.Setenv("OPENAI_RESPONSES_URL", server.URL)
	_ = os.Setenv("OPENAI_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{
		DefaultModel:    "gpt-5.2-codex",
		MaxOutputTokens: 4096,
	}
	cfg.OpenAI.ResponsesAPIModels = append(cfg.OpenAI.ResponsesAPIModels, "gpt-5.2-codex")
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "high"

	result := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName: "openai",
		Model:        "",
		Stdout:       io.Discard,
		Stderr:       io.Discard,
		Config:       cfg,
		AutoApprove:  true,
	}, query)

	if !strings.Contains(result, "Runtime-specific native web search succeeded.") {
		t.Fatalf("result should contain native search summary, got %q", result)
	}

	var req map[string]any
	select {
	case req = <-requestCh:
	default:
		t.Fatal("expected native OpenAI web search request to be sent")
	}

	if got := req["model"]; got != "gpt-5.2-codex" {
		t.Fatalf("model = %#v, want %q", got, "gpt-5.2-codex")
	}
	reasoning, ok := req["reasoning"].(map[string]any)
	if !ok {
		t.Fatalf("reasoning = %#v, want object", req["reasoning"])
	}
	if got := reasoning["effort"]; got != "high" {
		t.Fatalf("reasoning.effort = %#v, want %q", got, "high")
	}
}

func TestExecuteWebSearch_ReusesCurrentModelWhenAnthropicSearchSharesClaudeRuntimeIdentity(t *testing.T) {
	query := "runtime-config-anthropic-web-search-" + t.Name()
	requestCh := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		requestCh <- req

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"Anthropic web search reused current model."}]}`))
	}))
	defer server.Close()

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("ANTHROPIC_API_URL", oldURL)
		_ = os.Setenv("ANTHROPIC_API_KEY", oldKey)
	})
	_ = os.Setenv("ANTHROPIC_API_URL", server.URL)
	_ = os.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-default"},
		"claude":    {DefaultModel: "claude-default"},
	})

	result := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName: "claude",
		Model:        "claude-current",
		Stdout:       io.Discard,
		Stderr:       io.Discard,
		Config:       cfg,
		AutoApprove:  true,
	}, query)

	if !strings.Contains(result, "Anthropic web search reused current model.") {
		t.Fatalf("result should contain native search summary, got %q", result)
	}

	var req map[string]any
	select {
	case req = <-requestCh:
	default:
		t.Fatal("expected native Claude web search request to be sent")
	}

	if got := req["model"]; got != "claude-current" {
		t.Fatalf("model = %#v, want %q", got, "claude-current")
	}
	if got := req["model"]; got == "anthropic-default" {
		t.Fatal("search provider with same runtime identity must reuse the current model")
	}
}

func TestExecuteWebSearch_ReusesCurrentModelWhenClaudeSearchSharesAnthropicRuntimeIdentity(t *testing.T) {
	query := "runtime-config-claude-web-search-" + t.Name()
	requestCh := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		requestCh <- req

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"Claude web search reused current model."}]}`))
	}))
	defer server.Close()

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("ANTHROPIC_API_URL", oldURL)
		_ = os.Setenv("ANTHROPIC_API_KEY", oldKey)
	})
	_ = os.Setenv("ANTHROPIC_API_URL", server.URL)
	_ = os.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = "claude"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-default"},
		"claude":    {DefaultModel: "claude-default"},
	})

	result := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName: "anthropic",
		Model:        "anthropic-current",
		Stdout:       io.Discard,
		Stderr:       io.Discard,
		Config:       cfg,
		AutoApprove:  true,
	}, query)

	if !strings.Contains(result, "Claude web search reused current model.") {
		t.Fatalf("result should contain native search summary, got %q", result)
	}

	var req map[string]any
	select {
	case req = <-requestCh:
	default:
		t.Fatal("expected native Claude web search request to be sent")
	}

	if got := req["model"]; got != "anthropic-current" {
		t.Fatalf("model = %#v, want %q", got, "anthropic-current")
	}
	if got := req["model"]; got == "claude-default" {
		t.Fatal("search provider with same runtime identity must reuse the current model")
	}
}

func TestExecuteWebSearch_DoesNotReuseCurrentModelAcrossDifferentRuntimeProviders(t *testing.T) {
	query := "runtime-config-anthropic-web-search-negative-" + t.Name()
	requestCh := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		requestCh <- req

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"Anthropic web search used configured default model."}]}`))
	}))
	defer server.Close()

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("ANTHROPIC_API_URL", oldURL)
		_ = os.Setenv("ANTHROPIC_API_KEY", oldKey)
	})
	_ = os.Setenv("ANTHROPIC_API_URL", server.URL)
	_ = os.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-default"},
	})

	result := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName: "openai",
		Model:        "gpt-current",
		Stdout:       io.Discard,
		Stderr:       io.Discard,
		Config:       cfg,
		AutoApprove:  true,
	}, query)

	if !strings.Contains(result, "Anthropic web search used configured default model.") {
		t.Fatalf("result should contain native search summary, got %q", result)
	}

	var req map[string]any
	select {
	case req = <-requestCh:
	default:
		t.Fatal("expected native Claude web search request to be sent")
	}

	if got := req["model"]; got != "anthropic-default" {
		t.Fatalf("model = %#v, want %q", got, "anthropic-default")
	}
	if got := req["model"]; got == "gpt-current" {
		t.Fatal("different runtime providers must not reuse the current model")
	}
}

func TestExecuteWebSearch_UsesKimiNativeSearchProviderAndUsageCallback(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "kimi-native-web-search-" + t.Name()
	requestCh := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q, want Bearer test-key", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		requestCh <- req

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Kimi native web search succeeded.\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":13,\"completion_tokens\":4,\"cached_tokens\":2}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	oldURL := os.Getenv("KIMI_API_URL")
	oldKey := os.Getenv("MOONSHOT_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("KIMI_API_URL", oldURL)
		_ = os.Setenv("MOONSHOT_API_KEY", oldKey)
	})
	_ = os.Setenv("KIMI_API_URL", server.URL)
	_ = os.Setenv("MOONSHOT_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = "kimi"
	cfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{
		DefaultModel:    "kimi-web-search-default",
		MaxOutputTokens: 88,
	})

	var gotUsage api.Usage
	var gotUsageProvider string
	var gotUsageModel string
	result := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName: "deepseek",
		Model:        "deepseek-v4-flash",
		Stdout:       io.Discard,
		Stderr:       io.Discard,
		Config:       cfg,
		AutoApprove:  true,
		UsageAttribution: func(provider, model string, usage api.Usage) {
			gotUsageProvider = provider
			gotUsageModel = model
			gotUsage = usage
		},
	}, query)

	if !strings.Contains(result, "Kimi native web search succeeded.") {
		t.Fatalf("result should contain native Kimi search summary, got %q", result)
	}

	var req map[string]any
	select {
	case req = <-requestCh:
	default:
		t.Fatal("expected native Kimi web search request to be sent")
	}
	if got := req["model"]; got != "kimi-web-search-default" {
		t.Fatalf("model = %#v, want Kimi configured default model", got)
	}
	if req["max_completion_tokens"] != float64(88) {
		t.Fatalf("max_completion_tokens = %#v, want 88", req["max_completion_tokens"])
	}
	if _, ok := req["max_tokens"]; ok {
		t.Fatal("max_tokens should be omitted")
	}
	thinking, ok := req["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want disabled", req["thinking"])
	}
	toolsPayload, ok := req["tools"].([]any)
	if !ok || len(toolsPayload) != 1 {
		t.Fatalf("tools = %#v, want Kimi builtin web_search tool", req["tools"])
	}
	tool, ok := toolsPayload[0].(map[string]any)
	if !ok || tool["type"] != "builtin_function" {
		t.Fatalf("tool = %#v, want builtin_function", toolsPayload[0])
	}
	function, ok := tool["function"].(map[string]any)
	if !ok || function["name"] != "$web_search" {
		t.Fatalf("tool.function = %#v, want $web_search", tool["function"])
	}
	if _, ok := req["tool_choice"]; ok {
		t.Fatalf("tool_choice = %#v, want omitted", req["tool_choice"])
	}
	if gotUsage.InputTokens != 13 || gotUsage.OutputTokens != 4 || gotUsage.CachedInputTokens != 2 {
		t.Fatalf("usage = %+v, want input=13 output=4 cached=2", gotUsage)
	}
	if gotUsageProvider != "kimi" || gotUsageModel != "kimi-web-search-default" {
		t.Fatalf("usage owner = %s/%s, want kimi/kimi-web-search-default", gotUsageProvider, gotUsageModel)
	}
}

func TestSearchWithCache_DoesNotCacheIncompleteKimiWebSearch(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "kimi-incomplete-web-search-cache-" + t.Name()
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "text/event-stream")
		if requestCount == 1 {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"partial Kimi native web search result\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}],\"usage\":{\"prompt_tokens\":13,\"completion_tokens\":4}}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Complete Kimi native web search result.\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":13,\"completion_tokens\":4}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	oldURL := os.Getenv("KIMI_API_URL")
	oldKey := os.Getenv("MOONSHOT_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("KIMI_API_URL", oldURL)
		_ = os.Setenv("MOONSHOT_API_KEY", oldKey)
	})
	_ = os.Setenv("KIMI_API_URL", server.URL)
	_ = os.Setenv("MOONSHOT_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.WebSearch.CacheEnabled = true
	cfg.WebSearch.CacheSize = 50
	cfg.WebSearch.Provider = "kimi"
	cfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{
		DefaultModel:    "kimi-k2.6",
		MaxOutputTokens: 88,
	})
	ctx := config.WithContext(context.Background(), cfg)

	_, cached, err := searchWithCache(ctx, cfg, "kimi", query, "kimi-k2.6")
	if err == nil {
		t.Fatal("first searchWithCache() error = nil, want incomplete finish_reason error")
	}
	if cached {
		t.Fatal("first searchWithCache() cached = true, want false for failed response")
	}

	result, cached, err := searchWithCache(ctx, cfg, "kimi", query, "kimi-k2.6")
	if err != nil {
		t.Fatalf("second searchWithCache() error = %v", err)
	}
	if cached {
		t.Fatal("second searchWithCache() cached = true, want live request after failed response")
	}
	if !strings.Contains(result, "Complete Kimi native web search result.") {
		t.Fatalf("second searchWithCache() = %q, want complete result", result)
	}
	if requestCount != 2 {
		t.Fatalf("requestCount = %d, want 2 live requests", requestCount)
	}
}

func TestExecuteWebSearch_PreservesAnthropicOwnerForDefaultClaudeSearchReuse(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "default-claude-search-owner-" + t.Name()
	betaHeaderCh := make(chan string, 1)
	requestCh := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}
		betaHeaderCh <- r.Header.Get("anthropic-beta")

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		requestCh <- req

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"Anthropic-owned default Claude web search succeeded."}]}`))
	}))
	defer server.Close()

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("ANTHROPIC_API_URL", oldURL)
		_ = os.Setenv("ANTHROPIC_API_KEY", oldKey)
	})
	_ = os.Setenv("ANTHROPIC_API_URL", server.URL)
	_ = os.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {
			DefaultModel:  "shared-claude-model",
			AnthropicBeta: []string{"beta-anthropic"},
		},
		"claude": {
			DefaultModel:  "shared-claude-model",
			AnthropicBeta: []string{"beta-claude"},
		},
	})

	result := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		Model:             "shared-claude-model",
		Stdout:            io.Discard,
		Stderr:            io.Discard,
		Config:            cfg,
		AutoApprove:       true,
	}, query)

	if !strings.Contains(result, "Anthropic-owned default Claude web search succeeded.") {
		t.Fatalf("result should contain native search summary, got %q", result)
	}

	var req map[string]any
	select {
	case req = <-requestCh:
	default:
		t.Fatal("expected native Claude web search request to be sent")
	}
	if got := req["model"]; got != "shared-claude-model" {
		t.Fatalf("model = %#v, want %q", got, "shared-claude-model")
	}

	var betaHeader string
	select {
	case betaHeader = <-betaHeaderCh:
	default:
		t.Fatal("expected anthropic-beta header to be sent")
	}
	if !strings.Contains(betaHeader, "beta-anthropic") {
		t.Fatalf("anthropic-beta = %q, want anthropic-owned beta header", betaHeader)
	}
	if strings.Contains(betaHeader, "beta-claude") {
		t.Fatalf("anthropic-beta = %q, must not use claude-owned beta header", betaHeader)
	}
}

func TestExecuteWebSearch_DefaultClaudeSearchDoesNotReuseCacheAcrossAliasOwners(t *testing.T) {
	resetWebSearchCacheForTest()

	query := "default-claude-search-cache-owner-" + t.Name()
	requestCount := 0
	betaHeaders := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		betaHeaders = append(betaHeaders, r.Header.Get("anthropic-beta"))

		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"Anthropic-owned request executed."}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"Claude-owned request executed."}]}`))
	}))
	defer server.Close()

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("ANTHROPIC_API_URL", oldURL)
		_ = os.Setenv("ANTHROPIC_API_KEY", oldKey)
	})
	_ = os.Setenv("ANTHROPIC_API_URL", server.URL)
	_ = os.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {
			DefaultModel:  "shared-claude-model",
			AnthropicBeta: []string{"beta-anthropic"},
		},
		"claude": {
			DefaultModel:  "shared-claude-model",
			AnthropicBeta: []string{"beta-claude"},
		},
	})

	first := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName:      "claude",
		ProviderConfigKey: "anthropic",
		Model:             "shared-claude-model",
		Stdout:            io.Discard,
		Stderr:            io.Discard,
		Config:            cfg,
		AutoApprove:       true,
	}, query)
	if !strings.Contains(first, "Anthropic-owned request executed.") {
		t.Fatalf("first result = %q, want anthropic-owned response", first)
	}

	second := ExecuteWebSearch(tools.ExecutionContext{
		ProviderName:      "claude",
		ProviderConfigKey: "claude",
		Model:             "shared-claude-model",
		Stdout:            io.Discard,
		Stderr:            io.Discard,
		Config:            cfg,
		AutoApprove:       true,
	}, query)
	if !strings.Contains(second, "Claude-owned request executed.") {
		t.Fatalf("second result = %q, want claude-owned response instead of cached anthropic response", second)
	}

	if requestCount != 2 {
		t.Fatalf("requestCount = %d, want 2 separate requests for distinct alias owners", requestCount)
	}
	if len(betaHeaders) != 2 {
		t.Fatalf("len(betaHeaders) = %d, want 2", len(betaHeaders))
	}
	if !strings.Contains(betaHeaders[0], "beta-anthropic") {
		t.Fatalf("first anthropic-beta = %q, want anthropic-owned beta header", betaHeaders[0])
	}
	if !strings.Contains(betaHeaders[1], "beta-claude") {
		t.Fatalf("second anthropic-beta = %q, want claude-owned beta header", betaHeaders[1])
	}
}
