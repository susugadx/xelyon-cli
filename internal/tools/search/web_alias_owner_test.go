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
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

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
