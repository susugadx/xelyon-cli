package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newDeepSeekTestContext(t *testing.T, thinking bool) (context.Context, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	return newDeepSeekTestContextWithLevel(t, thinking, "high")
}

func newDeepSeekTestContextWithLevel(t *testing.T, thinking bool, level string) (context.Context, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = thinking
	cfg.Thinking.Level = level
	return newDeepSeekTestContextWithConfig(t, cfg)
}

func newDeepSeekTestContextWithConfig(t *testing.T, cfg *config.Config) (context.Context, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	var errOut bytes.Buffer
	runtime := ui.NewRuntime(strings.NewReader(""), &out, &errOut)
	ctx := ui.WithRuntime(context.Background(), runtime)
	ctx = config.WithContext(ctx, cfg)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	return ctx, &out, &errOut
}

func newDeepSeekSSEResponse(chunks ...string) *http.Response {
	var body strings.Builder
	for _, chunk := range chunks {
		body.WriteString("data: ")
		body.WriteString(chunk)
		body.WriteString("\n\n")
	}
	body.WriteString("data: [DONE]\n\n")
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body.String())),
	}
}

func TestProvider_IsFunctionCallingEnabled_RespectsEnv(t *testing.T) {
	t.Setenv("DEEPSEEK_FUNCTION_CALLING", "0")
	if New("test-key").IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() should be false when DEEPSEEK_FUNCTION_CALLING=0")
	}

	t.Setenv("DEEPSEEK_FUNCTION_CALLING", "1")
	if !New("test-key").IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() should be true when DEEPSEEK_FUNCTION_CALLING=1")
	}
}

func TestProvider_ChatWithTools_DeepSeekThinkingRequest(t *testing.T) {
	tests := []struct {
		name                string
		thinking            bool
		level               string
		model               string
		wantModel           string
		wantThinking        string
		wantReasoningEffort string
	}{
		{
			name:         "think off flash disables thinking",
			thinking:     false,
			level:        "high",
			model:        "deepseek-v4-flash",
			wantModel:    "deepseek-v4-flash",
			wantThinking: "disabled",
		},
		{
			name:                "think on high flash enables thinking",
			thinking:            true,
			level:               "high",
			model:               "deepseek-v4-flash",
			wantModel:           "deepseek-v4-flash",
			wantThinking:        "enabled",
			wantReasoningEffort: "high",
		},
		{
			name:                "think on xhigh flash maps effort to max",
			thinking:            true,
			level:               "xhigh",
			model:               "deepseek-v4-flash",
			wantModel:           "deepseek-v4-flash",
			wantThinking:        "enabled",
			wantReasoningEffort: "max",
		},
		{
			name:                "think on high pro keeps pro",
			thinking:            true,
			level:               "high",
			model:               "deepseek-v4-pro",
			wantModel:           "deepseek-v4-pro",
			wantThinking:        "enabled",
			wantReasoningEffort: "high",
		},
		{
			name:         "think off pro keeps pro",
			thinking:     false,
			level:        "high",
			model:        "deepseek-v4-pro",
			wantModel:    "deepseek-v4-pro",
			wantThinking: "disabled",
		},
		{
			name:         "legacy chat think off maps to flash disabled",
			thinking:     false,
			level:        "high",
			model:        "deepseek-chat",
			wantModel:    "deepseek-v4-flash",
			wantThinking: "disabled",
		},
		{
			name:                "legacy chat think on maps to flash enabled",
			thinking:            true,
			level:               "high",
			model:               "deepseek-chat",
			wantModel:           "deepseek-v4-flash",
			wantThinking:        "enabled",
			wantReasoningEffort: "high",
		},
		{
			name:                "legacy reasoner think on maps to flash enabled",
			thinking:            true,
			level:               "high",
			model:               "deepseek-reasoner",
			wantModel:           "deepseek-v4-flash",
			wantThinking:        "enabled",
			wantReasoningEffort: "high",
		},
		{
			name:         "legacy reasoner think off maps to flash disabled",
			thinking:     false,
			level:        "high",
			model:        "deepseek-reasoner",
			wantModel:    "deepseek-v4-flash",
			wantThinking: "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured map[string]any
			server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				streamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
			})
			t.Setenv("DEEPSEEK_API_URL", server.URL)

			p := New("test-key")
			ctx, _, _ := newDeepSeekTestContextWithLevel(t, tt.thinking, tt.level)
			_, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, tt.model)
			if err != nil {
				t.Fatalf("ChatWithTools() error = %v", err)
			}
			if captured["model"] != tt.wantModel {
				t.Fatalf("request model = %q, want %q", captured["model"], tt.wantModel)
			}
			thinking, ok := captured["thinking"].(map[string]any)
			if !ok {
				t.Fatalf("request thinking = %#v, want object", captured["thinking"])
			}
			if thinking["type"] != tt.wantThinking {
				t.Fatalf("thinking.type = %q, want %q", thinking["type"], tt.wantThinking)
			}
			gotReasoning, hasReasoning := captured["reasoning_effort"]
			if tt.wantReasoningEffort == "" {
				if hasReasoning {
					t.Fatalf("reasoning_effort = %q, want absent", gotReasoning)
				}
				return
			}
			if !hasReasoning || gotReasoning != tt.wantReasoningEffort {
				t.Fatalf("reasoning_effort = %q, want %q", gotReasoning, tt.wantReasoningEffort)
			}
		})
	}
}

func TestProvider_ChatWithTools_DeepSeekCatalogAliasDrivesThinkingRequest(t *testing.T) {
	tests := []struct {
		name                string
		thinking            bool
		level               string
		model               string
		wantModel           string
		wantThinking        string
		wantReasoningEffort string
	}{
		{
			name:         "default deployment uses catalog model for V4 thinking off",
			thinking:     false,
			level:        "high",
			model:        "",
			wantModel:    "corp-v4-pro",
			wantThinking: "disabled",
		},
		{
			name:                "model override deployment uses catalog model for xhigh",
			thinking:            true,
			level:               "xhigh",
			model:               "corp-v4-flash",
			wantModel:           "corp-v4-flash",
			wantThinking:        "enabled",
			wantReasoningEffort: "max",
		},
		{
			name:                "legacy catalog alias still enables V4 thinking",
			thinking:            true,
			level:               "high",
			model:               "corp-legacy-chat",
			wantModel:           "corp-legacy-chat",
			wantThinking:        "enabled",
			wantReasoningEffort: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured map[string]any
			server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				streamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
			})
			t.Setenv("DEEPSEEK_API_URL", server.URL)

			cfg := config.DefaultConfig()
			cfg.SetProviderModelConfig("deepseek", config.ProviderModelConfig{
				DefaultModel: "corp-v4-pro",
				CatalogModel: "deepseek-v4-pro",
				ModelOverrides: map[string]config.ModelOverride{
					"corp-v4-flash": {
						CatalogModel: "deepseek-v4-flash",
					},
					"corp-legacy-chat": {
						CatalogModel: "deepseek-chat",
					},
				},
			})
			cfg.Thinking.Enabled = tt.thinking
			cfg.Thinking.Level = tt.level

			p := New("test-key")
			ctx, _, _ := newDeepSeekTestContextWithConfig(t, cfg)
			_, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, tt.model)
			if err != nil {
				t.Fatalf("ChatWithTools() error = %v", err)
			}
			if captured["model"] != tt.wantModel {
				t.Fatalf("request model = %q, want %q", captured["model"], tt.wantModel)
			}
			if captured["max_tokens"] != float64(384000) {
				t.Fatalf("max_tokens = %v, want 384000 from catalog_model", captured["max_tokens"])
			}
			thinking, ok := captured["thinking"].(map[string]any)
			if !ok {
				t.Fatalf("request thinking = %#v, want object", captured["thinking"])
			}
			if thinking["type"] != tt.wantThinking {
				t.Fatalf("thinking.type = %q, want %q", thinking["type"], tt.wantThinking)
			}
			gotReasoning, hasReasoning := captured["reasoning_effort"]
			if tt.wantReasoningEffort == "" {
				if hasReasoning {
					t.Fatalf("reasoning_effort = %q, want absent", gotReasoning)
				}
				return
			}
			if !hasReasoning || gotReasoning != tt.wantReasoningEffort {
				t.Fatalf("reasoning_effort = %q, want %q", gotReasoning, tt.wantReasoningEffort)
			}
		})
	}
}

func TestProvider_ChatWithTools_ForcedToolChoiceIncludesCustomMCPTool(t *testing.T) {
	var captured openaicompat.ChatCompletionsRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		streamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("DEEPSEEK_API_URL", server.URL)

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{
		Name:        "custom_lookup",
		Description: "custom lookup",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		},
	}})
	p.SetToolChoice("custom_lookup")

	ctx, _, _ := newDeepSeekTestContext(t, false)
	_, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Lookup it"}}, "deepseek-chat")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	foundCustomTool := false
	for _, tool := range captured.Tools {
		if tool.Function != nil && tool.Function.Name == "custom_lookup" {
			foundCustomTool = true
			break
		}
	}
	if !foundCustomTool {
		t.Fatalf("request tools should include custom_lookup: %+v", captured.Tools)
	}

	toolChoice, ok := captured.ToolChoice.(map[string]any)
	if !ok {
		t.Fatalf("toolChoice type = %T, want map[string]any", captured.ToolChoice)
	}
	function, ok := toolChoice["function"].(map[string]any)
	if !ok || function["name"] != "custom_lookup" {
		t.Fatalf("toolChoice function = %#v, want custom_lookup", toolChoice["function"])
	}
}

func TestHandleStreamingResponse_PersistsReasoningAndUsage(t *testing.T) {
	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	ctx, out, _ := newDeepSeekTestContext(t, true)
	resp := newDeepSeekSSEResponse(
		`{"choices":[{"delta":{"reasoning_content":"Think"}}]}`,
		`{"choices":[{"delta":{"reasoning_content":" deeply"}}]}`,
		`{"choices":[{"delta":{"content":"Answer"}}]}`,
		`{"choices":[],"usage":{"prompt_tokens":11,"completion_tokens":7,"prompt_cache_hit_tokens":3,"completion_tokens_details":{"reasoning_tokens":2}}}`,
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
	)
	defer resp.Body.Close()

	got, err := p.handleStreamingResponse(ctx, resp, ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if got != "Answer" {
		t.Fatalf("handleStreamingResponse() = %q, want %q", got, "Answer")
	}
	if p.LastReasoningContent() != "Think deeply" {
		t.Fatalf("LastReasoningContent() = %q, want %q", p.LastReasoningContent(), "Think deeply")
	}
	if gotUsage.InputTokens != 11 ||
		gotUsage.OutputTokens != 5 ||
		gotUsage.ThinkingTokens != 2 ||
		gotUsage.CachedInputTokens != 3 {
		t.Fatalf("usage = %+v, want input=11 output=5 thinking=2 cached=3", gotUsage)
	}
	if !strings.Contains(out.String(), "Think deeply") {
		t.Fatalf("output should include reasoning content, got %q", out.String())
	}
}

func TestHandleStreamingResponse_NullUsageDoesNotEmitZeroUsageOnToolCallsFinish(t *testing.T) {
	p := New("test-key")
	callbackCount := 0
	p.SetUsageCallback(func(api.Usage) {
		callbackCount++
	})

	ctx, _, _ := newDeepSeekTestContext(t, false)
	resp := newDeepSeekSSEResponse(
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"/tmp/demo.txt\"}"}}]}}]}`,
		`{"choices":[],"usage":null}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	)
	defer resp.Body.Close()

	got, err := p.handleStreamingResponse(ctx, resp, ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if !strings.Contains(got, `"tool":"read_file"`) {
		t.Fatalf("handleStreamingResponse() = %q, want tool JSON", got)
	}
	if callbackCount != 0 {
		t.Fatalf("usage callback count = %d, want 0 when usage is null only", callbackCount)
	}
}
