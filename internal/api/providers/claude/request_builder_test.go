package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
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

func TestBuildRequestFeatures_DefaultsNilConfig(t *testing.T) {
	p := New("test-key")

	features := p.buildRequestFeatures(context.Background(), nil, "System", "claude-opus-4-7", "claude-opus-4-7")

	if features.System == nil {
		t.Fatal("System = nil, want default-config system payload")
	}
	if features.MaxTokens == 0 {
		t.Fatal("MaxTokens = 0, want default-config token policy")
	}
	if features.ContextManagement == nil {
		t.Fatal("ContextManagement = nil, want default-config context management payload")
	}
}

func TestBuildMessagesRequest_ReplaysThinkingForAlwaysOnFable(t *testing.T) {
	p := New("test-key")
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = false
	ctx := api.WithToolUseDisabled(config.WithContext(context.Background(), cfg))

	assistantMessage := api.Message{
		Role: "assistant",
		ToolCalls: []api.OpenAIToolCall{{
			ID:   "toolu_fable",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"/tmp/a.txt"}`,
			},
		}},
	}
	assistantMessage.SetAnthropicThinkingBlocks([]api.AnthropicThinkingBlock{{
		Type:      "thinking",
		Thinking:  "need the file",
		Signature: "sig_fable",
	}})

	built := p.buildMessagesRequest(ctx, "System", []api.Message{
		{Role: "user", Content: "Read a file"},
		assistantMessage,
	}, "claude-fable-5")

	if built.Request.Thinking != nil || built.Request.OutputConfig != nil {
		t.Fatalf("explicit thinking payload = %+v output_config=%+v, want omitted when config disables thinking", built.Request.Thinking, built.Request.OutputConfig)
	}
	if len(built.Request.Messages) != 2 || len(built.Request.Messages[1].Content) == 0 {
		t.Fatalf("messages = %#v, want assistant content with replayed thinking", built.Request.Messages)
	}
	first := built.Request.Messages[1].Content[0]
	if first.Type != "thinking" || first.Signature != "sig_fable" {
		t.Fatalf("first assistant content = %#v, want replayed Fable thinking block", first)
	}
}

func TestBuildMessagesRequest_AddsActiveContextToDynamicSystemSuffix(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true
	p := New("test-key")
	evidence := claudeTestRehydratedEvidence()
	ctx := api.WithActiveContextBlocks(config.WithContext(context.Background(), cfg), []api.ActiveContextBlock{{
		Name:    "provider_history_rehydrated_evidence",
		Content: evidence,
	}})

	built := p.buildMessagesRequest(ctx, "Static"+api.SystemPromptCacheBoundary+"Dynamic", []api.Message{{Role: "user", Content: "Hello"}}, defaultClaudeModel)

	systemBlocks, ok := built.Request.System.([]api.SystemBlock)
	if !ok || len(systemBlocks) != 2 {
		t.Fatalf("System = %#v, want static/dynamic system blocks", built.Request.System)
	}
	if systemBlocks[0].Text != "Static" {
		t.Fatalf("System[0].Text = %q, want static system prompt", systemBlocks[0].Text)
	}
	wantDynamic := "Dynamic\n\n" + evidence
	if systemBlocks[1].Text != wantDynamic {
		t.Fatalf("System[1].Text = %q, want active context appended to dynamic suffix", systemBlocks[1].Text)
	}
	if systemBlocks[1].CacheControl == nil {
		t.Fatal("System[1].CacheControl = nil, want dynamic cache boundary preserved")
	}
}

func claudeTestRehydratedEvidence() string {
	return taskstate.RenderRehydratedEvidenceBlock(taskstate.RehydratedEvidenceBlock{Items: []taskstate.RehydratedEvidenceItem{{
		Path:       "README.md",
		StartLine:  1,
		EndLine:    2,
		Source:     "read_file",
		Reason:     taskstate.RehydratePlanReasonOmittedProviderHistory,
		ToolCallID: "call_read",
		Content:    "line one\nline two",
	}}})
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

func TestBuildWebSearchRequest_InheritsThinkingPolicy(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		configure        func(*config.Config)
		wantType         string
		wantEffort       string
		wantBudgetTokens int
		wantMaxTokens    int
	}{
		{
			name:          "disabled omits thinking for adaptive model",
			model:         "claude-opus-4-7",
			wantMaxTokens: 2048,
		},
		{
			name:  "adaptive model sends output effort",
			model: "claude-opus-4-7",
			configure: func(cfg *config.Config) {
				cfg.Thinking.Enabled = true
				cfg.Thinking.Level = "xhigh"
			},
			wantType:      "adaptive",
			wantEffort:    "xhigh",
			wantMaxTokens: 2048,
		},
		{
			name:  "legacy thinking model sends budget tokens",
			model: "claude-3-5-sonnet",
			configure: func(cfg *config.Config) {
				cfg.Thinking.Enabled = true
				cfg.Thinking.Level = "high"
			},
			wantType:         "enabled",
			wantBudgetTokens: LevelToBudgetTokens("high"),
			wantMaxTokens:    LevelToBudgetTokens("high") + 2048,
		},
	}

	p := New("test-key")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			if tt.configure != nil {
				tt.configure(cfg)
			}
			built := p.buildWebSearchRequest(config.WithContext(context.Background(), cfg), "anthropic web search", tt.model)
			if built.Request.MaxTokens != tt.wantMaxTokens {
				t.Fatalf("MaxTokens = %d, want %d", built.Request.MaxTokens, tt.wantMaxTokens)
			}
			if tt.wantType == "" {
				if built.Request.Thinking != nil || built.Request.OutputConfig != nil {
					t.Fatalf("Thinking = %+v OutputConfig = %+v, want omitted", built.Request.Thinking, built.Request.OutputConfig)
				}
				return
			}
			if built.Request.Thinking == nil || built.Request.Thinking.Type != tt.wantType {
				t.Fatalf("Thinking = %+v, want type %q", built.Request.Thinking, tt.wantType)
			}
			if got := built.Request.Thinking.BudgetTokens; got != tt.wantBudgetTokens {
				t.Fatalf("BudgetTokens = %d, want %d", got, tt.wantBudgetTokens)
			}
			if tt.wantEffort == "" {
				if built.Request.OutputConfig != nil {
					t.Fatalf("OutputConfig = %+v, want nil", built.Request.OutputConfig)
				}
				return
			}
			if built.Request.OutputConfig == nil || built.Request.OutputConfig.Effort != tt.wantEffort {
				t.Fatalf("OutputConfig = %+v, want effort %q", built.Request.OutputConfig, tt.wantEffort)
			}
		})
	}
}

func TestBuildWebSearchRequest_LegacyThinkingKeepsConfiguredVisibleOutputBudget(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "high"
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		ModelOverrides: map[string]config.ModelOverride{
			"corp-legacy-web": {
				CatalogModel:    "claude-3-5-sonnet",
				MaxOutputTokens: 1024,
			},
		},
	})

	p := New("test-key")
	built := p.buildWebSearchRequest(config.WithContext(context.Background(), cfg), "anthropic web search", "corp-legacy-web")
	budget := LevelToBudgetTokens("high")
	if built.Request.Thinking == nil || built.Request.Thinking.BudgetTokens != budget {
		t.Fatalf("Thinking = %+v, want budget %d", built.Request.Thinking, budget)
	}
	if built.Request.MaxTokens != budget+1024 {
		t.Fatalf("MaxTokens = %d, want budget plus configured visible output %d", built.Request.MaxTokens, budget+1024)
	}
}
