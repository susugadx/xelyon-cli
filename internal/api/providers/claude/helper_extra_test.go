package claude

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProviderConfigKey_NormalizesAlias(t *testing.T) {
	p := New("test-key")
	if got := p.ProviderConfigKey(); got != "claude" {
		t.Fatalf("ProviderConfigKey() = %q, want %q", got, "claude")
	}

	p.SetProviderConfigKey("Anthropic")
	if got := p.ProviderConfigKey(); got != "anthropic" {
		t.Fatalf("ProviderConfigKey() after SetProviderConfigKey = %q, want %q", got, "anthropic")
	}

	var nilProvider *Provider
	nilProvider.SetProviderConfigKey("claude")
}

func TestSupportsClaudeCompaction_UsesRuntimeAndContextConfig(t *testing.T) {
	p := New("test-key")

	runtimeCfg := config.DefaultConfig()
	runtimeCfg.Compression.ClaudeCompaction = false
	p.SetRuntimeConfig(runtimeCfg)
	if p.SupportsClaudeCompaction() {
		t.Fatal("SupportsClaudeCompaction() = true, want false when runtime config disables compaction")
	}

	ctxCfg := config.DefaultConfig()
	ctxCfg.Compression.ClaudeCompaction = true
	ctx := config.WithContext(context.Background(), ctxCfg)
	if !p.SupportsClaudeCompactionWithContext(ctx, "claude-sonnet-4-6") {
		t.Fatal("SupportsClaudeCompactionWithContext() = false, want true when context config enables supported model")
	}
	if p.SupportsClaudeCompactionWithContext(ctx, "claude-3-5-sonnet") {
		t.Fatal("SupportsClaudeCompactionWithContext() = true, want false for unsupported model")
	}

	ctxCfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		DefaultModel: "corp-claude-opus47",
		CatalogModel: "claude-opus-4-7",
	})
	if !p.SupportsClaudeCompactionWithContext(ctx, "corp-claude-opus47") {
		t.Fatal("SupportsClaudeCompactionWithContext() = false, want true via catalog_model")
	}

	defaultCfg := config.DefaultConfig()
	defaultCfg.Compression.ClaudeCompaction = true
	pm := defaultCfg.ProviderModels["claude"]
	pm.DefaultModel = "claude-opus-4-6"
	defaultCfg.ProviderModels["claude"] = pm
	p.SetRuntimeConfig(defaultCfg)
	if !p.SupportsClaudeCompaction() {
		t.Fatal("SupportsClaudeCompaction() = false, want true for supported runtime model")
	}
}

func TestSetUsageCallback_StoresCallback(t *testing.T) {
	p := New("test-key")

	called := false
	p.SetUsageCallback(func(usage api.Usage) {
		called = usage.InputTokens == 0
	})

	if p.usageCallback == nil {
		t.Fatal("usageCallback should be stored")
	}
	p.usageCallback(api.Usage{})
	if !called {
		t.Fatal("stored usage callback was not invoked")
	}
}

func TestSetMessageCacheBreakpointsWithEnabled_Wrapper(t *testing.T) {
	messages := []AnthropicMessage{
		{
			Role: "user",
			Content: []AnthropicContentBlock{
				{Type: "tool_result", ToolUseID: "tool-1", Content: strings.Repeat("A", 600)},
			},
		},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "second"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "third"}}},
		{Role: "user", Content: []AnthropicContentBlock{{Type: "text", Text: "fourth"}}},
	}

	SetMessageCacheBreakpointsWithEnabled(messages, true)
	if messages[0].Content[0].CacheControl == nil {
		t.Fatal("SetMessageCacheBreakpointsWithEnabled() should set cache breakpoint for stable tool result")
	}

	messages[0].Content[0].CacheControl = nil
	SetMessageCacheBreakpointsWithEnabled(messages, false)
	if messages[0].Content[0].CacheControl != nil {
		t.Fatal("SetMessageCacheBreakpointsWithEnabled() should not mutate messages when disabled")
	}
}

func TestExtractCompaction(t *testing.T) {
	t.Run("valid markers", func(t *testing.T) {
		summary, text := extractCompaction("before\n[COMPACTION]\nsummary\n[/COMPACTION]\nafter")
		if summary != "summary" {
			t.Fatalf("summary = %q, want %q", summary, "summary")
		}
		if text != "before\n\nafter" {
			t.Fatalf("text = %q, want %q", text, "before\\n\\nafter")
		}
	})

	t.Run("missing end marker returns original text", func(t *testing.T) {
		summary, text := extractCompaction("before [COMPACTION] summary only")
		if summary != "" {
			t.Fatalf("summary = %q, want empty", summary)
		}
		if text != "before [COMPACTION] summary only" {
			t.Fatalf("text = %q, want original content", text)
		}
	})
}

func TestValidateAnthropicToolPairs_DebugOutput(t *testing.T) {
	var out bytes.Buffer
	validateAnthropicToolPairs([]AnthropicMessage{
		{
			Role: "assistant",
			Content: []AnthropicContentBlock{
				{Type: "tool_use", ID: "tool-1", Name: "read_file"},
			},
		},
		{
			Role: "user",
			Content: []AnthropicContentBlock{
				{Type: "tool_result", ToolUseID: "tool-1", Content: "ok"},
				{Type: "tool_result", ToolUseID: "tool-2", Content: "mismatch"},
			},
		},
	}, &out)

	got := out.String()
	if !strings.Contains(got, `tool_result.tool_use_id="tool-2"`) {
		t.Fatalf("expected mismatched tool_result debug output, got %q", got)
	}
	if !strings.Contains(got, "2 tool_results vs 1 tool_uses") {
		t.Fatalf("expected count mismatch debug output, got %q", got)
	}
}

func TestGetToolDefinitionNames_ContainsBuiltins(t *testing.T) {
	names := GetToolDefinitionNames()
	if len(names) == 0 {
		t.Fatal("GetToolDefinitionNames() returned no tool definitions")
	}

	foundReadFile := false
	for _, name := range names {
		if name == "read_file" {
			foundReadFile = true
			break
		}
	}
	if !foundReadFile {
		t.Fatalf("expected builtin read_file tool in definitions, got %v", names)
	}
}
