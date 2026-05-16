package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestBuildMessagesRequest_UsesRuntimeFeaturePolicy(t *testing.T) {
	t.Setenv(claudeFunctionCallEnv, "1")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		DefaultModel:    "corp-claude-opus47",
		CatalogModel:    "claude-opus-4-7",
		MaxOutputTokens: 64000,
	})
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"

	p := New("test-key")
	ctx := api.WithToolUseDisabled(config.WithContext(context.Background(), cfg))
	built := p.buildMessagesRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hello"}}, "corp-claude-opus47")

	if built.Model != "corp-claude-opus47" || built.Request.Model != "corp-claude-opus47" {
		t.Fatalf("model = %q/%q, want corp-claude-opus47", built.Model, built.Request.Model)
	}
	if built.Request.MaxTokens != 128000 {
		t.Fatalf("MaxTokens = %d, want 128000 from catalog_model", built.Request.MaxTokens)
	}
	if built.Request.Thinking == nil || built.Request.Thinking.Type != "adaptive" {
		t.Fatalf("Thinking = %+v, want adaptive", built.Request.Thinking)
	}
	if built.Request.OutputConfig == nil || built.Request.OutputConfig.Effort != "xhigh" {
		t.Fatalf("OutputConfig = %+v, want effort=xhigh", built.Request.OutputConfig)
	}
	if built.Request.ContextManagement == nil {
		t.Fatal("ContextManagement = nil, want runtime context management payload")
	}
	if len(built.Request.Tools) != 0 {
		t.Fatalf("Tools = %d, want omitted when tool use is disabled by context", len(built.Request.Tools))
	}
}

func TestBuildMessagesRequest_UsesForcedToolChoice(t *testing.T) {
	t.Setenv(claudeFunctionCallEnv, "1")

	p := New("test-key")
	p.SetToolChoice("custom_lookup")
	ctx := api.WithToolDefinitions(context.Background(), []api.ToolDefinition{{
		Name:        "custom_lookup",
		Description: "lookup",
		Parameters:  map[string]interface{}{"type": "object"},
	}})

	built := p.buildMessagesRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hello"}}, defaultClaudeModel)
	if len(built.Request.Tools) != 1 || built.Request.Tools[0].Name != "custom_lookup" {
		t.Fatalf("Tools = %+v, want custom_lookup tool", built.Request.Tools)
	}
	requireClaudeToolChoice(t, built.Request.ToolChoice, "custom_lookup")

	p.ClearToolChoice()
	built = p.buildMessagesRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hello"}}, defaultClaudeModel)
	if built.Request.ToolChoice != nil {
		t.Fatalf("ToolChoice = %+v, want nil after ClearToolChoice", built.Request.ToolChoice)
	}
}

func TestAnthropicHeaders_MergeConfiguredAndContextManagementBetas(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		DefaultModel:     "corp-claude",
		AnthropicVersion: "2099-01-01",
		AnthropicBeta:    []string{"configured-beta"},
	})

	p := New("test-key")
	ctx := config.WithContext(context.Background(), cfg)
	contextManagement := &ContextManagement{Edits: []ContextEdit{
		{Type: clearToolUsesEditType},
		{Type: compactEditType},
	}}
	headers := p.anthropicHeaders(ctx, "corp-claude", contextManagement)

	if got := headers.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := headers.Get("x-api-key"); got != "test-key" {
		t.Fatalf("x-api-key = %q, want test-key", got)
	}
	if got := headers.Get("anthropic-version"); got != "2099-01-01" {
		t.Fatalf("anthropic-version = %q, want 2099-01-01", got)
	}
	beta := headers.Get("anthropic-beta")
	for _, want := range []string{"configured-beta", contextManagementBetaHeader, compactBetaHeader} {
		if !strings.Contains(beta, want) {
			t.Fatalf("anthropic-beta = %q, want to include %q", beta, want)
		}
	}
}

func TestBuildWebSearchRequest_UsesSharedHeaderPolicy(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		DefaultModel:     "corp-claude-web",
		MaxOutputTokens:  1024,
		AnthropicVersion: "2099-01-01",
		AnthropicBeta:    []string{"configured-beta"},
	})

	p := New("test-key")
	ctx := config.WithContext(context.Background(), cfg)
	built := p.buildWebSearchRequest(ctx, "anthropic web search", "corp-claude-web")
	if built.Request.MaxTokens != 1024 {
		t.Fatalf("MaxTokens = %d, want provider max 1024", built.Request.MaxTokens)
	}
	if built.Request.Stream {
		t.Fatal("Stream = true, want false for native web search")
	}
	if len(built.Request.Tools) != 1 || built.Request.Tools[0].Type != "web_search_20250305" {
		t.Fatalf("Tools = %+v, want one web_search_20250305 tool", built.Request.Tools)
	}

	headers := p.anthropicHeaders(ctx, built.Model, nil, webSearchBetaHeader)
	beta := headers.Get("anthropic-beta")
	for _, want := range []string{webSearchBetaHeader, "configured-beta"} {
		if !strings.Contains(beta, want) {
			t.Fatalf("anthropic-beta = %q, want to include %q", beta, want)
		}
	}
}
