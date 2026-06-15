package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"

	// ツール登録のための blank import
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

func TestNewClaudeProvider(t *testing.T) {
	apiKey := "test-api-key"
	provider := New(apiKey)

	if provider == nil {
		t.Fatal("New() returned nil")
	}
}

func TestClaudeProvider_Name(t *testing.T) {
	provider := New("test-key")

	name := provider.Name()
	if name != "Claude" {
		t.Errorf("Name() = %v, want 'Claude'", name)
	}
}

func TestClaudeProvider_SupportsImages(t *testing.T) {
	provider := New("test-key")

	supports := provider.SupportsImages()
	if !supports {
		t.Error("SupportsImages() = false, want true (Claude supports images)")
	}
}

func TestNewClaudeProvider_URLOverride(t *testing.T) {
	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)

	t.Run("DefaultURL", func(t *testing.T) {
		os.Unsetenv("ANTHROPIC_API_URL")
		p := New("test-key")
		if p.APIURL != defaultClaudeURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL, defaultClaudeURL)
		}
	})

	t.Run("CustomURL", func(t *testing.T) {
		customURL := "https://custom.anthropic.api.com/v1"
		os.Setenv("ANTHROPIC_API_URL", customURL)
		p := New("test-key")
		if p.APIURL != customURL {
			t.Errorf("apiURL = %q, want %q", p.APIURL, customURL)
		}
	})
}

// mockAPIServer creates a test HTTP server
func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// claudeStreamingHandler はClaude形式のストリーミングハンドラー
func claudeStreamingHandler(texts []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for _, text := range texts {
			event := StreamEvent{
				Type:  "content_block_delta",
				Delta: &Delta{Type: "text_delta", Text: text},
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}

		// 終了イベント
		stopEvent := StreamEvent{Type: "message_stop"}
		data, _ := json.Marshal(stopEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// claudeToolUseStreamingHandler は Tool Use を含むストリーミングハンドラー
func claudeToolUseStreamingHandler(toolID, toolName string, inputChunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// content_block_start (tool_use)
		startEvent := StreamEvent{
			Type:  "content_block_start",
			Index: 0,
			ContentBlock: &ContentBlock{
				Type: "tool_use",
				ID:   toolID,
				Name: toolName,
			},
		}
		data, _ := json.Marshal(startEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// content_block_delta (input_json_delta)
		for _, chunk := range inputChunks {
			deltaEvent := StreamEvent{
				Type:  "content_block_delta",
				Index: 0,
				Delta: &Delta{Type: "input_json_delta", PartialJSON: chunk},
			}
			data, _ := json.Marshal(deltaEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}

		// content_block_stop
		stopBlockEvent := StreamEvent{
			Type:  "content_block_stop",
			Index: 0,
		}
		data, _ = json.Marshal(stopBlockEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// message_delta (stop_reason)
		msgDeltaEvent := StreamEvent{
			Type:  "message_delta",
			Delta: &Delta{StopReason: "tool_use"},
		}
		data, _ = json.Marshal(msgDeltaEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// message_stop
		stopEvent := StreamEvent{Type: "message_stop"}
		data, _ = json.Marshal(stopEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// errorHandler returns a handler that responds with error
func errorHandler(statusCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"%s"}}`, message)
	}
}

// rateLimitHandler returns a handler that responds with rate limit error
func rateLimitHandler(retryAfter string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}
}

const (
	testSharedClaudeAliasModel   = "shared-custom"
	testAnthropicAliasMaxTokens  = 1024
	testCanonicalClaudeMaxTokens = 1536
)

func newClaudeAliasSizingConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {
			DefaultModel:     testSharedClaudeAliasModel,
			MaxOutputTokens:  testAnthropicAliasMaxTokens,
			AnthropicVersion: "2099-01-01",
			AnthropicBeta:    []string{"alias-beta"},
		},
		"claude": {
			DefaultModel:     testSharedClaudeAliasModel,
			MaxOutputTokens:  testCanonicalClaudeMaxTokens,
			AnthropicVersion: "2024-01-01",
			AnthropicBeta:    []string{"canonical-beta"},
		},
	})
	return cfg
}

func captureClaudeRequestForProvider(t *testing.T, cfg *config.Config, providerKey, model string) (Request, http.Header) {
	t.Helper()

	var reqBody Request
	var headers http.Header
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Response{
			Content: []Content{{Type: "text", Text: "ok"}},
		})
	})

	t.Setenv("ANTHROPIC_API_URL", server.URL)

	p := newProvider("test-key", providerKey)
	ctx := config.WithContext(context.Background(), cfg)
	_, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Hello"}}, model)
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	return reqBody, headers
}

func captureClaudeRequest(t *testing.T, cfg *config.Config, model string) (Request, http.Header) {
	t.Helper()
	return captureClaudeRequestForProvider(t, cfg, "claude", model)
}

func captureClaudeRawRequest(t *testing.T, cfg *config.Config, model string) (map[string]any, http.Header) {
	t.Helper()

	var reqBody map[string]any
	var headers http.Header
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Response{
			Content: []Content{{Type: "text", Text: "ok"}},
		})
	})

	t.Setenv("ANTHROPIC_API_URL", server.URL)

	p := newProvider("test-key", "claude")
	ctx := config.WithContext(context.Background(), cfg)
	_, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Hello"}}, model)
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	return reqBody, headers
}

func captureClaudeImageRequestForProvider(t *testing.T, cfg *config.Config, providerKey, model string) (MultimodalRequest, http.Header) {
	t.Helper()

	var reqBody MultimodalRequest
	var headers http.Header
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Response{
			Content: []Content{{Type: "text", Text: "ok"}},
		})
	})

	t.Setenv("ANTHROPIC_API_URL", server.URL)

	p := newProvider("test-key", providerKey)
	ctx := config.WithContext(context.Background(), cfg)
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}
	if _, err := p.ChatWithImage(ctx, "System", nil, "Describe this", image, model); err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}

	return reqBody, headers
}

func headerHasBetaValue(header http.Header, value string) bool {
	for _, raw := range strings.Split(header.Get("anthropic-beta"), ",") {
		if strings.TrimSpace(raw) == value {
			return true
		}
	}
	return false
}

func TestChatWithTools_SetsTopLevelCacheControlWhenPromptCacheEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	reqBody, _ := captureClaudeRequest(t, cfg, "claude-opus-4-6")
	if reqBody.CacheControl == nil {
		t.Fatal("expected top-level cache_control when prompt cache is enabled")
	}
	if reqBody.CacheControl.Type != "ephemeral" {
		t.Fatalf("cache_control.type = %q, want %q", reqBody.CacheControl.Type, "ephemeral")
	}
}

func TestChatWithTools_OmitsTopLevelCacheControlWhenPromptCacheDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = false

	reqBody, _ := captureClaudeRequest(t, cfg, "claude-opus-4-6")
	if reqBody.CacheControl != nil {
		t.Fatalf("expected no top-level cache_control when prompt cache is disabled, got %+v", reqBody.CacheControl)
	}
}

func TestClearToolUses_EditsInRequest(t *testing.T) {
	cfg := config.DefaultConfig()

	reqBody, _ := captureClaudeRequest(t, cfg, "claude-sonnet-4-6")
	if reqBody.ContextManagement == nil {
		t.Fatal("ContextManagement should be set")
	}
	if len(reqBody.ContextManagement.Edits) != 2 {
		t.Fatalf("len(ContextManagement.Edits) = %d, want 2", len(reqBody.ContextManagement.Edits))
	}
	if reqBody.ContextManagement.Edits[0].Type != clearToolUsesEditType {
		t.Errorf("Edits[0].Type = %q, want %q", reqBody.ContextManagement.Edits[0].Type, clearToolUsesEditType)
	}
	if reqBody.ContextManagement.Edits[1].Type != compactEditType {
		t.Errorf("Edits[1].Type = %q, want %q", reqBody.ContextManagement.Edits[1].Type, compactEditType)
	}
}

func TestClearToolUses_Disabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClearToolUses = false

	reqBody, _ := captureClaudeRequest(t, cfg, "claude-sonnet-4-6")
	if reqBody.ContextManagement == nil {
		t.Fatal("ContextManagement should be set")
	}
	if len(reqBody.ContextManagement.Edits) != 1 {
		t.Fatalf("len(ContextManagement.Edits) = %d, want 1", len(reqBody.ContextManagement.Edits))
	}
	if reqBody.ContextManagement.Edits[0].Type != compactEditType {
		t.Errorf("Edits[0].Type = %q, want %q", reqBody.ContextManagement.Edits[0].Type, compactEditType)
	}
}

func TestClearToolUses_IndependentFromCompaction(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false

	reqBody, headers := captureClaudeRequest(t, cfg, "claude-3-5-sonnet")
	if reqBody.ContextManagement == nil {
		t.Fatal("ContextManagement should be set when clear_tool_uses is enabled")
	}
	if len(reqBody.ContextManagement.Edits) != 1 {
		t.Fatalf("len(ContextManagement.Edits) = %d, want 1", len(reqBody.ContextManagement.Edits))
	}
	if reqBody.ContextManagement.Edits[0].Type != clearToolUsesEditType {
		t.Errorf("Edits[0].Type = %q, want %q", reqBody.ContextManagement.Edits[0].Type, clearToolUsesEditType)
	}
	if !headerHasBetaValue(headers, contextManagementBetaHeader) {
		t.Errorf("anthropic-beta should include %q, got %q", contextManagementBetaHeader, headers.Get("anthropic-beta"))
	}
	if headerHasBetaValue(headers, compactBetaHeader) {
		t.Errorf("anthropic-beta should not include %q when compaction is disabled, got %q", compactBetaHeader, headers.Get("anthropic-beta"))
	}
}

func TestClearToolUses_TriggerOrder(t *testing.T) {
	cfg := config.DefaultConfig()

	reqBody, _ := captureClaudeRequest(t, cfg, "claude-sonnet-4-6")
	if reqBody.ContextManagement == nil || len(reqBody.ContextManagement.Edits) != 2 {
		t.Fatal("ContextManagement.Edits should contain clear_tool_uses and compact")
	}

	clearTrigger := reqBody.ContextManagement.Edits[0].Trigger
	compactTrigger := reqBody.ContextManagement.Edits[1].Trigger
	if clearTrigger == nil || compactTrigger == nil {
		t.Fatal("both edits should include trigger")
	}
	if clearTrigger.Value >= compactTrigger.Value {
		t.Errorf("clear trigger = %d, compact trigger = %d, want clear < compact", clearTrigger.Value, compactTrigger.Value)
	}
}

func TestClearToolUses_BetaHeader(t *testing.T) {
	cfg := config.DefaultConfig()

	_, headers := captureClaudeRequest(t, cfg, "claude-sonnet-4-6")
	if !headerHasBetaValue(headers, contextManagementBetaHeader) {
		t.Errorf("anthropic-beta should include %q, got %q", contextManagementBetaHeader, headers.Get("anthropic-beta"))
	}
	if !headerHasBetaValue(headers, compactBetaHeader) {
		t.Errorf("anthropic-beta should include %q, got %q", compactBetaHeader, headers.Get("anthropic-beta"))
	}
}

func TestClaudeCompaction_Opus47Request(t *testing.T) {
	cfg := config.DefaultConfig()

	reqBody, headers := captureClaudeRequest(t, cfg, "claude-opus-4-7")
	if reqBody.ContextManagement == nil {
		t.Fatal("ContextManagement should be set for Opus 4.7")
	}
	if len(reqBody.ContextManagement.Edits) != 2 {
		t.Fatalf("len(ContextManagement.Edits) = %d, want 2", len(reqBody.ContextManagement.Edits))
	}
	if reqBody.ContextManagement.Edits[0].Type != clearToolUsesEditType {
		t.Fatalf("Edits[0].Type = %q, want %q", reqBody.ContextManagement.Edits[0].Type, clearToolUsesEditType)
	}
	if reqBody.ContextManagement.Edits[1].Type != compactEditType {
		t.Fatalf("Edits[1].Type = %q, want %q", reqBody.ContextManagement.Edits[1].Type, compactEditType)
	}
	if !headerHasBetaValue(headers, compactBetaHeader) {
		t.Fatalf("anthropic-beta should include %q, got %q", compactBetaHeader, headers.Get("anthropic-beta"))
	}
}

func TestClaudeCompaction_DisabledKeepsClearToolUsesForOpus47(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false

	reqBody, headers := captureClaudeRequest(t, cfg, "claude-opus-4-7")
	if reqBody.ContextManagement == nil {
		t.Fatal("ContextManagement should be set for clear_tool_uses")
	}
	if len(reqBody.ContextManagement.Edits) != 1 {
		t.Fatalf("len(ContextManagement.Edits) = %d, want 1", len(reqBody.ContextManagement.Edits))
	}
	if reqBody.ContextManagement.Edits[0].Type != clearToolUsesEditType {
		t.Fatalf("Edits[0].Type = %q, want %q", reqBody.ContextManagement.Edits[0].Type, clearToolUsesEditType)
	}
	if headerHasBetaValue(headers, compactBetaHeader) {
		t.Fatalf("anthropic-beta should not include %q when compaction is disabled, got %q", compactBetaHeader, headers.Get("anthropic-beta"))
	}
}

func TestClaudeOpus47CatalogModelDrivesRequestFeatures(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		DefaultModel:    "corp-claude-opus47",
		CatalogModel:    "claude-opus-4-7",
		MaxOutputTokens: 64000,
	})
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"

	reqBody, headers := captureClaudeRawRequest(t, cfg, "corp-claude-opus47")
	if reqBody["model"] != "corp-claude-opus47" {
		t.Fatalf("model = %v, want raw request model", reqBody["model"])
	}
	if reqBody["max_tokens"] != float64(128000) {
		t.Fatalf("max_tokens = %v, want 128000 from catalog_model", reqBody["max_tokens"])
	}
	thinking, ok := reqBody["thinking"].(map[string]any)
	if !ok || thinking["type"] != "adaptive" {
		t.Fatalf("thinking = %+v, want adaptive via catalog_model", reqBody["thinking"])
	}
	if _, ok := thinking["budget_tokens"]; ok {
		t.Fatalf("thinking.budget_tokens should be omitted for catalog_model adaptive thinking, got %+v", thinking)
	}
	outputConfig, ok := reqBody["output_config"].(map[string]any)
	if !ok || outputConfig["effort"] != "xhigh" {
		t.Fatalf("output_config = %+v, want effort=xhigh via catalog_model", reqBody["output_config"])
	}
	if !headerHasBetaValue(headers, compactBetaHeader) {
		t.Fatalf("anthropic-beta should include %q via catalog_model, got %q", compactBetaHeader, headers.Get("anthropic-beta"))
	}
}

func TestClaudeOpus47CatalogModelDrivesImageRequestFeatures(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		DefaultModel:    "corp-claude-opus47",
		CatalogModel:    "claude-opus-4-7",
		MaxOutputTokens: 64000,
	})
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"

	reqBody, headers := captureClaudeImageRequestForProvider(t, cfg, "claude", "corp-claude-opus47")
	if reqBody.Model != "corp-claude-opus47" {
		t.Fatalf("Model = %q, want raw request model", reqBody.Model)
	}
	if reqBody.MaxTokens != 128000 {
		t.Fatalf("MaxTokens = %d, want 128000 from catalog_model", reqBody.MaxTokens)
	}
	if reqBody.Thinking == nil || reqBody.Thinking.Type != "adaptive" {
		t.Fatalf("Thinking = %+v, want adaptive via catalog_model", reqBody.Thinking)
	}
	if reqBody.Thinking.BudgetTokens != 0 {
		t.Fatalf("Thinking.BudgetTokens = %d, want omitted for catalog_model adaptive thinking", reqBody.Thinking.BudgetTokens)
	}
	if reqBody.OutputConfig == nil || reqBody.OutputConfig.Effort != "xhigh" {
		t.Fatalf("OutputConfig = %+v, want effort=xhigh via catalog_model", reqBody.OutputConfig)
	}
	if !headerHasBetaValue(headers, compactBetaHeader) {
		t.Fatalf("anthropic-beta should include %q via catalog_model, got %q", compactBetaHeader, headers.Get("anthropic-beta"))
	}
}

func TestClaudeProvider_UsesAnthropicAliasProviderConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("anthropic", config.ProviderModelConfig{
		DefaultModel:     "anthropic-custom",
		AnthropicVersion: "2099-01-01",
		AnthropicBeta:    []string{"alias-beta"},
	})

	reqBody, headers := captureClaudeRequest(t, cfg, "claude-sonnet-4-6")
	if reqBody.Model != "claude-sonnet-4-6" {
		t.Fatalf("Model = %q, want %q", reqBody.Model, "claude-sonnet-4-6")
	}
	if got := headers.Get("anthropic-version"); got != "2099-01-01" {
		t.Fatalf("anthropic-version = %q, want %q", got, "2099-01-01")
	}
	if !headerHasBetaValue(headers, "alias-beta") {
		t.Fatalf("anthropic-beta should include %q, got %q", "alias-beta", headers.Get("anthropic-beta"))
	}
}

func TestClaudeProvider_PrefersAnthropicOwnerHeadersForAnthropicSelectedModel(t *testing.T) {
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

	reqBody, headers := captureClaudeRequest(t, cfg, "anthropic-custom")
	if reqBody.Model != "anthropic-custom" {
		t.Fatalf("Model = %q, want %q", reqBody.Model, "anthropic-custom")
	}
	if got := headers.Get("anthropic-version"); got != "2099-01-01" {
		t.Fatalf("anthropic-version = %q, want %q", got, "2099-01-01")
	}
	if !headerHasBetaValue(headers, "alias-beta") {
		t.Fatalf("anthropic-beta should include %q, got %q", "alias-beta", headers.Get("anthropic-beta"))
	}
	if headerHasBetaValue(headers, "canonical-beta") {
		t.Fatalf("anthropic-beta should not include %q, got %q", "canonical-beta", headers.Get("anthropic-beta"))
	}
}

func TestClaudeProvider_PrefersRequestedAnthropicAliasWhenBothAliasEntriesShareSameModel(t *testing.T) {
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

	reqBody, headers := captureClaudeRequestForProvider(t, cfg, "anthropic", "shared-custom")
	if reqBody.Model != "shared-custom" {
		t.Fatalf("Model = %q, want %q", reqBody.Model, "shared-custom")
	}
	if got := headers.Get("anthropic-version"); got != "2099-01-01" {
		t.Fatalf("anthropic-version = %q, want %q", got, "2099-01-01")
	}
	if !headerHasBetaValue(headers, "alias-beta") {
		t.Fatalf("anthropic-beta should include %q, got %q", "alias-beta", headers.Get("anthropic-beta"))
	}
	if headerHasBetaValue(headers, "canonical-beta") {
		t.Fatalf("anthropic-beta should not include %q, got %q", "canonical-beta", headers.Get("anthropic-beta"))
	}
}

func TestClaudeProvider_AnthropicAliasRequestSizingUsesAnthropicMaxTokens(t *testing.T) {
	cfg := newClaudeAliasSizingConfig()

	reqBody, headers := captureClaudeRequestForProvider(t, cfg, "anthropic", testSharedClaudeAliasModel)
	if reqBody.MaxTokens != testAnthropicAliasMaxTokens {
		t.Fatalf("MaxTokens = %d, want %d", reqBody.MaxTokens, testAnthropicAliasMaxTokens)
	}
	if got := headers.Get("anthropic-version"); got != "2099-01-01" {
		t.Fatalf("anthropic-version = %q, want %q", got, "2099-01-01")
	}
	if !headerHasBetaValue(headers, "alias-beta") {
		t.Fatalf("anthropic-beta should include %q, got %q", "alias-beta", headers.Get("anthropic-beta"))
	}
	if headerHasBetaValue(headers, "canonical-beta") {
		t.Fatalf("anthropic-beta should not include %q, got %q", "canonical-beta", headers.Get("anthropic-beta"))
	}
}

func TestClaudeProvider_ClaudeAliasRequestSizingUsesClaudeMaxTokens(t *testing.T) {
	cfg := newClaudeAliasSizingConfig()

	reqBody, headers := captureClaudeRequestForProvider(t, cfg, "claude", testSharedClaudeAliasModel)
	if reqBody.MaxTokens != testCanonicalClaudeMaxTokens {
		t.Fatalf("MaxTokens = %d, want %d", reqBody.MaxTokens, testCanonicalClaudeMaxTokens)
	}
	if got := headers.Get("anthropic-version"); got != "2024-01-01" {
		t.Fatalf("anthropic-version = %q, want %q", got, "2024-01-01")
	}
	if !headerHasBetaValue(headers, "canonical-beta") {
		t.Fatalf("anthropic-beta should include %q, got %q", "canonical-beta", headers.Get("anthropic-beta"))
	}
	if headerHasBetaValue(headers, "alias-beta") {
		t.Fatalf("anthropic-beta should not include %q, got %q", "alias-beta", headers.Get("anthropic-beta"))
	}
}

func TestClaudeProvider_AnthropicAliasImageRequestSizingUsesAnthropicMaxTokens(t *testing.T) {
	cfg := newClaudeAliasSizingConfig()

	reqBody, headers := captureClaudeImageRequestForProvider(t, cfg, "anthropic", testSharedClaudeAliasModel)
	if reqBody.MaxTokens != testAnthropicAliasMaxTokens {
		t.Fatalf("MaxTokens = %d, want %d", reqBody.MaxTokens, testAnthropicAliasMaxTokens)
	}
	if got := headers.Get("anthropic-version"); got != "2099-01-01" {
		t.Fatalf("anthropic-version = %q, want %q", got, "2099-01-01")
	}
	if !headerHasBetaValue(headers, "alias-beta") {
		t.Fatalf("anthropic-beta should include %q, got %q", "alias-beta", headers.Get("anthropic-beta"))
	}
	if headerHasBetaValue(headers, "canonical-beta") {
		t.Fatalf("anthropic-beta should not include %q, got %q", "canonical-beta", headers.Get("anthropic-beta"))
	}
}

func TestClearToolUses_ClearToolInputs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClearToolInputs = true

	reqBody, _ := captureClaudeRequest(t, cfg, "claude-sonnet-4-6")
	if reqBody.ContextManagement == nil || len(reqBody.ContextManagement.Edits) == 0 {
		t.Fatal("ContextManagement.Edits should be set")
	}

	edit := reqBody.ContextManagement.Edits[0]
	if edit.Type != clearToolUsesEditType {
		t.Fatalf("Edits[0].Type = %q, want %q", edit.Type, clearToolUsesEditType)
	}
	if edit.ClearToolInputs == nil || !*edit.ClearToolInputs {
		t.Fatal("clear_tool_inputs should be present and true")
	}
}

func TestContextManagement_TriggerMinimum(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false
	cfg.Compression.ClearToolUsesTrigger = 1

	reqBody, _ := captureClaudeRequest(t, cfg, "claude-3-5-sonnet")
	if reqBody.ContextManagement == nil || len(reqBody.ContextManagement.Edits) != 1 {
		t.Fatal("ContextManagement should contain one clear_tool_uses edit")
	}
	if got := reqBody.ContextManagement.Edits[0].Trigger.Value; got != minimumContextEditTrigger {
		t.Errorf("clear_tool_uses trigger = %d, want %d", got, minimumContextEditTrigger)
	}

	cfg = config.DefaultConfig()
	cfg.Compression.CompactionTrigger = 1
	reqBody, _ = captureClaudeRequest(t, cfg, "claude-sonnet-4-6")
	if reqBody.ContextManagement == nil || len(reqBody.ContextManagement.Edits) != 2 {
		t.Fatal("ContextManagement should contain clear_tool_uses and compact edits")
	}
	if got := reqBody.ContextManagement.Edits[1].Trigger.Value; got != minimumContextEditTrigger {
		t.Errorf("compaction trigger = %d, want %d", got, minimumContextEditTrigger)
	}
}

func TestClaudeProvider_ChatWithTools_NonStreaming(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %q, want 'test-key'", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("anthropic-version = %q, want '2023-06-01'", r.Header.Get("anthropic-version"))
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "claude-sonnet-4-6" {
			t.Errorf("Model = %q, want 'claude-sonnet-4-6'", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "Test response from Claude"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	result, err := p.ChatWithTools(context.Background(), "System prompt", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Test response from Claude" {
		t.Errorf("ChatWithTools() = %q, want 'Test response from Claude'", result)
	}
}

func TestClaudeProvider_ChatWithTools_Streaming(t *testing.T) {
	server := mockAPIServer(t, claudeStreamingHandler([]string{"Hello", " from", " Claude"}))

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if result != "Hello from Claude" {
		t.Errorf("ChatWithTools() = %q, want 'Hello from Claude'", result)
	}
}

func TestClaudeProvider_ChatWithTools_APIError(t *testing.T) {
	server := mockAPIServer(t, errorHandler(500, "Internal Server Error"))

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for 500 status")
	}
}

func TestClaudeProvider_ChatWithTools_RateLimit(t *testing.T) {
	server := mockAPIServer(t, rateLimitHandler("1"))

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hi"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "")
	if err == nil {
		t.Error("ChatWithTools() should return error for rate limit")
	}
}

func TestClaudeProvider_ChatWithImage_NoImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "No image response"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Hello", nil, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "No image response" {
		t.Errorf("ChatWithImage() = %q, want 'No image response'", result)
	}
}

func TestClaudeProvider_ChatWithImage_WithImage(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// リクエストボディにimageソースがあることを確認
		var req MultimodalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "Image analysis complete"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	result, err := p.ChatWithImage(context.Background(), "System", nil, "Describe this", image, "")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image analysis complete" {
		t.Errorf("ChatWithImage() = %q, want 'Image analysis complete'", result)
	}
}

func TestClaudeProvider_ChatWithImage_ClearToolUsesOnly(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false

	var reqBody MultimodalRequest
	var headers http.Header
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Response{
			Content: []Content{{Type: "text", Text: "Image analysis complete"}},
		})
	})

	t.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	ctx := config.WithContext(context.Background(), cfg)
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png"}

	result, err := p.ChatWithImage(ctx, "System", nil, "Describe this", image, "claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if result != "Image analysis complete" {
		t.Errorf("ChatWithImage() = %q, want %q", result, "Image analysis complete")
	}
	if reqBody.ContextManagement == nil || len(reqBody.ContextManagement.Edits) != 1 {
		t.Fatal("ContextManagement should contain only clear_tool_uses")
	}
	if reqBody.ContextManagement.Edits[0].Type != clearToolUsesEditType {
		t.Errorf("Edits[0].Type = %q, want %q", reqBody.ContextManagement.Edits[0].Type, clearToolUsesEditType)
	}
	if !headerHasBetaValue(headers, contextManagementBetaHeader) {
		t.Errorf("anthropic-beta should include %q, got %q", contextManagementBetaHeader, headers.Get("anthropic-beta"))
	}
	if headerHasBetaValue(headers, compactBetaHeader) {
		t.Errorf("anthropic-beta should not include %q when compaction is disabled, got %q", compactBetaHeader, headers.Get("anthropic-beta"))
	}

	messages := reqBody.Messages
	if len(messages) == 0 {
		t.Fatal("Messages should not be empty")
	}

	lastMsg, ok := messages[len(messages)-1].(map[string]interface{})
	if !ok {
		t.Fatalf("Last message should be a map, got %T", messages[len(messages)-1])
	}
	content, ok := lastMsg["content"].([]interface{})
	if !ok {
		t.Fatalf("Last message content should be an array, got %T", lastMsg["content"])
	}
	if len(content) != 2 {
		t.Errorf("Content length = %d, want 2 (image + text)", len(content))
	}
}

func TestLevelToBudgetTokens(t *testing.T) {
	tests := []struct {
		level string
		want  int
	}{
		{"low", 5000},
		{"medium", 10000},
		{"high", 20000},
		{"xhigh", 40000},
		{"unknown", 10000}, // default
		{"", 10000},        // default
		{"invalid", 10000}, // default
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := LevelToBudgetTokens(tt.level)
			if got != tt.want {
				t.Errorf("LevelToBudgetTokens(%q) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

func TestIsAdaptiveThinkingModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"claude-opus-4-8", true},
		{"claude-opus-4.8", true},
		{"claude-opus-4-7", true},
		{"claude-opus-4.7", true},
		{"claude-fable-5", true},
		{"claude-opus-4-6", true},
		{"claude-sonnet-4-6", true},
		{"claude-opus-4.6", true},
		{"claude-sonnet-4.6", true},
		{"claude-opus-4-5-20251101", false},
		{"claude-sonnet-4-5-20250929", false},
		{"claude-sonnet-4-20250514", false},
		{"claude-opus-4-20250918", false},
		{"claude-sonnet-3-7-20250219", false},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := IsAdaptiveThinkingModel(tt.model); got != tt.want {
				t.Errorf("IsAdaptiveThinkingModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestLevelToEffort(t *testing.T) {
	tests := []struct {
		level string
		model string
		want  string
	}{
		{"low", "claude-opus-4-6", "low"},
		{"medium", "claude-opus-4-6", "medium"},
		{"high", "claude-opus-4-6", "high"},
		{"xhigh", "claude-opus-4-8", "xhigh"},
		{"xhigh", "claude-opus-4.8", "xhigh"},
		{"xhigh", "claude-opus-4-7", "xhigh"},
		{"xhigh", "claude-opus-4.7", "xhigh"},
		{"xhigh", "claude-fable-5", "xhigh"},
		{"xhigh", "claude-opus-4-6", "max"},
		{"xhigh", "claude-sonnet-4-6", "high"},
		{"", "claude-opus-4-6", "medium"},
		{"invalid", "claude-opus-4-6", "medium"},
	}
	for _, tt := range tests {
		t.Run(tt.level+"_"+tt.model, func(t *testing.T) {
			if got := levelToEffort(tt.level, tt.model); got != tt.want {
				t.Errorf("levelToEffort(%q, %q) = %q, want %q", tt.level, tt.model, got, tt.want)
			}
		})
	}
}

func TestIsCompactionSupportedModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"claude-fable-5", true},
		{"claude-opus-4-8", true},
		{"claude-opus-4.8", true},
		{"claude-opus-4-7", true},
		{"claude-opus-4.7", true},
		{"Claude-Opus-4-7", true},
		{"claude-opus-4-6", true},
		{"claude-opus-4-5", true},
		{"claude-sonnet-4-6", true},
		{"claude-3-5-sonnet", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := IsCompactionSupportedModel(tt.model); got != tt.want {
				t.Errorf("IsCompactionSupportedModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

// Tool Use Tests

func TestClaudeProvider_ChatWithTools_ToolUse(t *testing.T) {
	// Disable function calling env var for consistent testing
	originalEnv := os.Getenv("CLAUDE_FUNCTION_CALLING")
	defer os.Setenv("CLAUDE_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("CLAUDE_FUNCTION_CALLING")

	inputChunks := []string{
		`{"pa`,
		`th":"`,
		`/test.txt"`,
		`}`,
	}
	server := mockAPIServer(t, claudeToolUseStreamingHandler("toolu_01ABC123", "read_file", inputChunks))

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read test.txt"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// Should contain tool JSON
	if result == "" {
		t.Error("ChatWithTools() returned empty result, expected tool JSON")
	}
	if !contains(result, "read_file") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'read_file'", result)
	}
	if !contains(result, "toolu_01ABC123") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'toolu_01ABC123'", result)
	}
}

func TestClaudeProvider_ChatWithTools_NonStreaming_ToolUse(t *testing.T) {
	originalEnv := os.Getenv("CLAUDE_FUNCTION_CALLING")
	defer os.Setenv("CLAUDE_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("CLAUDE_FUNCTION_CALLING")

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{
				{Type: "text", Text: "I'll read that file."},
				{
					Type:  "tool_use",
					ID:    "toolu_01XYZ789",
					Name:  "read_file",
					Input: map[string]interface{}{"path": "/readme.md"},
				},
			},
			StopReason: "tool_use",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Read readme.md"}}

	result, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	if !contains(result, "I'll read that file.") {
		t.Errorf("ChatWithTools() = %q, expected to contain text", result)
	}
	if !contains(result, "read_file") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'read_file'", result)
	}
	if !contains(result, "toolu_01XYZ789") {
		t.Errorf("ChatWithTools() = %q, expected to contain 'toolu_01XYZ789'", result)
	}
}

func TestSetMCPTools(t *testing.T) {
	p := New("test-key")

	tools := []api.ToolDefinition{
		{Name: "custom_tool", Description: "A custom tool"},
	}
	p.SetMCPTools(tools)

	if len(p.mcpTools) != 1 {
		t.Errorf("mcpTools length = %d, want 1", len(p.mcpTools))
	}
	if p.mcpTools[0].Name != "custom_tool" {
		t.Errorf("mcpTools[0].Name = %q, want 'custom_tool'", p.mcpTools[0].Name)
	}
}

func TestClaudeProvider_ChatWithTools_FunctionCallingDisabled(t *testing.T) {
	originalEnv := os.Getenv("CLAUDE_FUNCTION_CALLING")
	defer os.Setenv("CLAUDE_FUNCTION_CALLING", originalEnv)
	os.Setenv("CLAUDE_FUNCTION_CALLING", "0")

	var requestBody Request
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "No tools"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// When CLAUDE_FUNCTION_CALLING=0, Tools should not be included
	if len(requestBody.Tools) > 0 {
		t.Errorf("Tools should be empty when CLAUDE_FUNCTION_CALLING=0, got %d tools", len(requestBody.Tools))
	}
}

func TestClaudeProvider_ChatWithTools_ToolUseDisabledOmitsTools(t *testing.T) {
	t.Setenv("CLAUDE_FUNCTION_CALLING", "1")

	var requestBody map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Response{Content: []Content{{Type: "text", Text: "No tools"}}}
		_ = json.NewEncoder(w).Encode(resp)
	})

	t.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{Name: "custom_lookup", Description: "custom lookup"}})
	ctx := api.WithToolUseDisabled(context.Background())
	if _, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Hello"}}, "claude-sonnet-4-6"); err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if _, ok := requestBody["tools"]; ok {
		t.Fatalf("tools should be omitted when tool use is disabled: %#v", requestBody["tools"])
	}
}

func TestClaudeProvider_ChatWithTools_FunctionCallingEnabled(t *testing.T) {
	originalEnv := os.Getenv("CLAUDE_FUNCTION_CALLING")
	defer os.Setenv("CLAUDE_FUNCTION_CALLING", originalEnv)
	os.Unsetenv("CLAUDE_FUNCTION_CALLING")

	var requestBody Request
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Content: []Content{{Type: "text", Text: "With tools"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	originalURL := os.Getenv("ANTHROPIC_API_URL")
	defer os.Setenv("ANTHROPIC_API_URL", originalURL)
	os.Setenv("ANTHROPIC_API_URL", server.URL)

	p := New("test-key")
	history := []api.Message{{Role: "user", Content: "Hello"}}

	_, err := p.ChatWithTools(context.Background(), "System", history, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	// When CLAUDE_FUNCTION_CALLING is not "0", Tools should be included
	if len(requestBody.Tools) == 0 {
		t.Error("Tools should not be empty when CLAUDE_FUNCTION_CALLING is not disabled")
	}
}

func TestGetClaudeToolDefinitions(t *testing.T) {
	tools := GetClaudeToolDefinitions()

	if len(tools) == 0 {
		t.Error("GetClaudeToolDefinitions() returned empty slice")
	}

	// read_file ツールが含まれていることを確認
	found := false
	for _, tool := range tools {
		if tool.Name == "read_file" {
			found = true
			if tool.InputSchema == nil {
				t.Error("read_file tool should have InputSchema")
			}
			break
		}
	}
	if !found {
		t.Error("GetClaudeToolDefinitions() should contain 'read_file' tool")
	}
}

func TestConvertOpenAIToolToClaude(t *testing.T) {
	openaiTool := api.ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]interface{}{"type": "string"},
			},
		},
	}

	claudeTool := ConvertOpenAIToolToClaude(openaiTool)

	if claudeTool.Name != "test_tool" {
		t.Errorf("Name = %q, want 'test_tool'", claudeTool.Name)
	}
	if claudeTool.Description != "A test tool" {
		t.Errorf("Description = %q, want 'A test tool'", claudeTool.Description)
	}
	if claudeTool.InputSchema == nil {
		t.Error("InputSchema should not be nil")
	}
}

func TestConvertToolUseToToolJSON(t *testing.T) {
	input := map[string]interface{}{
		"path": "/test.txt",
	}

	result, err := ConvertToolUseToToolJSON("toolu_01ABC", "read_file", input)
	if err != nil {
		t.Fatalf("ConvertToolUseToToolJSON() error = %v", err)
	}

	if !contains(result, "toolu_01ABC") {
		t.Errorf("result = %q, expected to contain 'toolu_01ABC'", result)
	}
	if !contains(result, "read_file") {
		t.Errorf("result = %q, expected to contain 'read_file'", result)
	}
	if !contains(result, "/test.txt") {
		t.Errorf("result = %q, expected to contain '/test.txt'", result)
	}
}

func TestGetCombinedClaudeTools(t *testing.T) {
	mcpTools := []api.ToolDefinition{
		{Name: "mcp_tool_1", Description: "MCP Tool 1"},
		{Name: "mcp_tool_2", Description: "MCP Tool 2"},
	}

	combined := GetCombinedClaudeToolsWithContext(context.Background(), mcpTools)

	builtInCount := len(GetClaudeToolDefinitions())
	expectedCount := builtInCount + 2

	if len(combined) != expectedCount {
		t.Errorf("GetCombinedClaudeToolsWithContext() returned %d tools, want %d", len(combined), expectedCount)
	}

	// MCP ツールが含まれていることを確認
	found := 0
	for _, tool := range combined {
		if tool.Name == "mcp_tool_1" || tool.Name == "mcp_tool_2" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("Expected 2 MCP tools in combined list, found %d", found)
	}
}

func TestGetCombinedClaudeTools_BP2(t *testing.T) {
	// BP#2: ツール定義の末尾に cache_control が設定されること
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	ctx := tools.WithConfig(context.Background(), cfg)
	claudeTools := GetCombinedClaudeToolsWithContext(ctx, nil)
	if len(claudeTools) == 0 {
		t.Fatal("expected at least 1 tool")
	}
	last := claudeTools[len(claudeTools)-1]
	if last.CacheControl == nil {
		t.Error("expected cache_control on last tool (BP#2)")
	}
	if last.CacheControl != nil && last.CacheControl.Type != "ephemeral" {
		t.Errorf("cache_control type = %q, want 'ephemeral'", last.CacheControl.Type)
	}

	// 最後以外には cache_control なし
	for i := 0; i < len(claudeTools)-1; i++ {
		if claudeTools[i].CacheControl != nil {
			t.Errorf("tools[%d] should not have cache_control", i)
		}
	}
}

func TestGetCombinedClaudeTools_BP2Disabled(t *testing.T) {
	// prompt_cache.enabled=false → cache_control なし
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = false

	ctx := tools.WithConfig(context.Background(), cfg)
	claudeTools := GetCombinedClaudeToolsWithContext(ctx, nil)
	if len(claudeTools) == 0 {
		t.Fatal("expected at least 1 tool")
	}
	last := claudeTools[len(claudeTools)-1]
	if last.CacheControl != nil {
		t.Errorf("expected no cache_control when prompt cache disabled, got %+v", last.CacheControl)
	}
}

// Helper function for string contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
