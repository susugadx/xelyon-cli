package search

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
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
