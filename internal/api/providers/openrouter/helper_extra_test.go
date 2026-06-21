package openrouter

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

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
	if !p.SupportsClaudeCompactionWithContext(ctx, "anthropic/claude-sonnet-4.6") {
		t.Fatal("SupportsClaudeCompactionWithContext() = false, want true for supported context model")
	}
	if p.SupportsClaudeCompactionWithContext(ctx, "openai/gpt-5") {
		t.Fatal("SupportsClaudeCompactionWithContext() = true, want false for non-Claude model")
	}

	defaultCfg := config.DefaultConfig()
	defaultCfg.Compression.ClaudeCompaction = true
	pm := defaultCfg.ProviderModels["openrouter"]
	pm.DefaultModel = "anthropic/claude-sonnet-4.6"
	defaultCfg.ProviderModels["openrouter"] = pm
	p.SetRuntimeConfig(defaultCfg)
	if !p.SupportsClaudeCompaction() {
		t.Fatal("SupportsClaudeCompaction() = false, want true for supported runtime model")
	}
}

func TestSetToolChoiceAndClear(t *testing.T) {
	p := New("test-key")
	p.SetToolChoice("read_file")
	if p.toolChoice == nil || *p.toolChoice != "read_file" {
		t.Fatalf("toolChoice = %#v, want read_file", p.toolChoice)
	}

	p.ClearToolChoice()
	if p.toolChoice != nil {
		t.Fatalf("toolChoice = %#v, want nil after ClearToolChoice", p.toolChoice)
	}
}

func TestHandleNonStreamingResponse_UsageCallback(t *testing.T) {
	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	runtime := uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	ctx := uiruntime.WithRuntime(context.Background(), runtime)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{
			"choices":[{"message":{"content":"hello"}}],
			"usage":{"prompt_tokens":12,"completion_tokens":5,"prompt_tokens_details":{"cached_tokens":4},"completion_tokens_details":{"reasoning_tokens":2}}
		}`)),
	}

	result, err := p.handleNonStreamingResponse(ctx, resp, uiruntime.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleNonStreamingResponse() error = %v", err)
	}
	if result != "hello" {
		t.Fatalf("handleNonStreamingResponse() = %q, want %q", result, "hello")
	}
	if gotUsage.InputTokens != 12 || gotUsage.OutputTokens != 3 || gotUsage.CachedInputTokens != 4 || gotUsage.ThinkingTokens != 2 {
		t.Fatalf("usage callback = %+v, want input=12 output=3 cached=4 thinking=2", gotUsage)
	}
}

func TestHandleClaudeStreamingResponse_CompactionAndUsage(t *testing.T) {
	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	runtime := uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	ctx := uiruntime.WithRuntime(context.Background(), runtime)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	body := strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":10,"cache_read_input_tokens":2,"cache_creation_input_tokens":1}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"compaction"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"summary"}}`,
		`data: {"type":"content_block_stop","index":0}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Hello"}}`,
		`data: {"type":"message_delta","usage":{"output_tokens":7}}`,
		`data: {"type":"message_stop"}`,
	}, "\n\n") + "\n\n"

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(body)),
	}

	result, err := p.handleClaudeStreamingResponse(ctx, resp, uiruntime.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleClaudeStreamingResponse() error = %v", err)
	}
	if !strings.Contains(result, "[COMPACTION]\nsummary\n[/COMPACTION]\nHello") {
		t.Fatalf("unexpected streamed result: %q", result)
	}
	if gotUsage.InputTokens != 13 || gotUsage.CachedInputTokens != 2 || gotUsage.CacheCreationTokens != 1 || gotUsage.OutputTokens != 7 {
		t.Fatalf("usage callback = %+v, want input=13 cached=2 creation=1 output=7", gotUsage)
	}
}
