package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func captureWebSearchRequestForProvider(t *testing.T, cfg *config.Config, providerKey, model string) (webSearchRequest, http.Header) {
	t.Helper()

	var reqBody webSearchRequest
	var headers http.Header
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
	})

	t.Setenv("ANTHROPIC_API_URL", server.URL)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	ctx := config.WithContext(context.Background(), cfg)
	if _, err := webSearchWithContextForProvider(ctx, providerKey, "anthropic web search tool", model); err != nil {
		t.Fatalf("webSearchWithContextForProvider() error = %v", err)
	}

	return reqBody, headers
}

func TestWebSearch_Success(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}
		if got := r.Header.Get("anthropic-beta"); !strings.Contains(got, webSearchBetaHeader) {
			t.Fatalf("anthropic-beta = %q, want %q", got, webSearchBetaHeader)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		tools, ok := req["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("tools = %#v, want 1 tool", req["tools"])
		}
		tool, ok := tools[0].(map[string]any)
		if !ok || tool["type"] != "web_search_20250305" {
			t.Fatalf("tool = %#v, want type=web_search_20250305", tools[0])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"content": [
				{
					"type": "text",
					"text": "Anthropic shipped a web search tool.",
					"citations": [
						{"type": "web_search_result_location", "title": "Anthropic Docs", "url": "https://docs.anthropic.com/en/docs/build-with-claude/tool-use/web-search-tool"}
					]
				}
			]
		}`))
	})

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_URL", server.URL)
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Setenv("ANTHROPIC_API_URL", oldURL)
	defer os.Setenv("ANTHROPIC_API_KEY", oldKey)

	ctx := config.WithContext(context.Background(), config.DefaultConfig())
	result, err := WebSearchWithContext(ctx, "anthropic web search tool", "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
	if !strings.Contains(result, "Summary:") {
		t.Fatalf("result should contain Summary, got %q", result)
	}
	if !strings.Contains(result, "https://docs.anthropic.com/en/docs/build-with-claude/tool-use/web-search-tool") {
		t.Fatalf("result should contain citation URL, got %q", result)
	}
}

func TestWebSearch_UsesAnthropicAliasProviderConfig(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("anthropic-version"); got != "2099-01-01" {
			t.Fatalf("anthropic-version = %q, want %q", got, "2099-01-01")
		}
		if got := r.Header.Get("anthropic-beta"); !strings.Contains(got, "alias-beta") {
			t.Fatalf("anthropic-beta = %q, want to include %q", got, "alias-beta")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
	})

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_URL", server.URL)
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Setenv("ANTHROPIC_API_URL", oldURL)
	defer os.Setenv("ANTHROPIC_API_KEY", oldKey)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("anthropic", config.ProviderModelConfig{
		AnthropicVersion: "2099-01-01",
		AnthropicBeta:    []string{"alias-beta"},
	})
	ctx := config.WithContext(context.Background(), cfg)

	if _, err := WebSearchWithContext(ctx, "anthropic web search tool", "claude-sonnet-4-6"); err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
}

func TestWebSearch_PrefersAnthropicOwnerHeadersForAnthropicSelectedModel(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("anthropic-version"); got != "2099-01-01" {
			t.Fatalf("anthropic-version = %q, want %q", got, "2099-01-01")
		}
		if got := r.Header.Get("anthropic-beta"); !strings.Contains(got, "alias-beta") {
			t.Fatalf("anthropic-beta = %q, want to include %q", got, "alias-beta")
		}
		if got := r.Header.Get("anthropic-beta"); strings.Contains(got, "canonical-beta") {
			t.Fatalf("anthropic-beta = %q, should not include %q", got, "canonical-beta")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
	})

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_URL", server.URL)
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Setenv("ANTHROPIC_API_URL", oldURL)
	defer os.Setenv("ANTHROPIC_API_KEY", oldKey)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
provider_models:
  anthropic:
    default_model: anthropic-custom
    anthropic_version: 2099-01-01
    anthropic_beta:
      - alias-beta
  claude:
    default_model: claude-custom
    anthropic_version: 2024-01-01
    anthropic_beta:
      - canonical-beta
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx := config.WithContext(context.Background(), cfg)

	if _, err := WebSearchWithContext(ctx, "anthropic web search tool", "anthropic-custom"); err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
}

func TestWebSearch_PrefersRequestedAnthropicAliasWhenBothAliasEntriesShareSameModel(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("anthropic-version"); got != "2099-01-01" {
			t.Fatalf("anthropic-version = %q, want %q", got, "2099-01-01")
		}
		if got := r.Header.Get("anthropic-beta"); !strings.Contains(got, "alias-beta") {
			t.Fatalf("anthropic-beta = %q, want to include %q", got, "alias-beta")
		}
		if got := r.Header.Get("anthropic-beta"); strings.Contains(got, "canonical-beta") {
			t.Fatalf("anthropic-beta = %q, should not include %q", got, "canonical-beta")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
	})

	oldURL := os.Getenv("ANTHROPIC_API_URL")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_URL", server.URL)
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Setenv("ANTHROPIC_API_URL", oldURL)
	defer os.Setenv("ANTHROPIC_API_KEY", oldKey)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
provider_models:
  anthropic:
    default_model: shared-custom
    anthropic_version: 2099-01-01
    anthropic_beta:
      - alias-beta
  claude:
    default_model: shared-custom
    anthropic_version: 2024-01-01
    anthropic_beta:
      - canonical-beta
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx := config.WithContext(context.Background(), cfg)

	if _, err := webSearchWithContextForProvider(ctx, "anthropic", "anthropic web search tool", "shared-custom"); err != nil {
		t.Fatalf("webSearchWithContextForProvider() error = %v", err)
	}
}

func TestWebSearch_AnthropicAliasRequestSizingUsesAnthropicMaxTokens(t *testing.T) {
	cfg := newClaudeAliasSizingConfig()

	reqBody, headers := captureWebSearchRequestForProvider(t, cfg, "anthropic", testSharedClaudeAliasModel)
	if reqBody.MaxTokens != testAnthropicAliasMaxTokens {
		t.Fatalf("MaxTokens = %d, want %d", reqBody.MaxTokens, testAnthropicAliasMaxTokens)
	}
	if got := headers.Get("anthropic-version"); got != "2099-01-01" {
		t.Fatalf("anthropic-version = %q, want %q", got, "2099-01-01")
	}
	if !strings.Contains(headers.Get("anthropic-beta"), "alias-beta") {
		t.Fatalf("anthropic-beta = %q, want to include %q", headers.Get("anthropic-beta"), "alias-beta")
	}
	if strings.Contains(headers.Get("anthropic-beta"), "canonical-beta") {
		t.Fatalf("anthropic-beta = %q, should not include %q", headers.Get("anthropic-beta"), "canonical-beta")
	}
}

func TestWebSearch_ClaudeAliasRequestSizingUsesClaudeMaxTokens(t *testing.T) {
	cfg := newClaudeAliasSizingConfig()

	reqBody, headers := captureWebSearchRequestForProvider(t, cfg, "claude", testSharedClaudeAliasModel)
	if reqBody.MaxTokens != testCanonicalClaudeMaxTokens {
		t.Fatalf("MaxTokens = %d, want %d", reqBody.MaxTokens, testCanonicalClaudeMaxTokens)
	}
	if got := headers.Get("anthropic-version"); got != "2024-01-01" {
		t.Fatalf("anthropic-version = %q, want %q", got, "2024-01-01")
	}
	if !strings.Contains(headers.Get("anthropic-beta"), "canonical-beta") {
		t.Fatalf("anthropic-beta = %q, want to include %q", headers.Get("anthropic-beta"), "canonical-beta")
	}
	if strings.Contains(headers.Get("anthropic-beta"), "alias-beta") {
		t.Fatalf("anthropic-beta = %q, should not include %q", headers.Get("anthropic-beta"), "alias-beta")
	}
}
