package kimi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestChatWithTools_RequestShape(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("KIMI_API_URL", server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	got, err := New("test-key").ChatWithTools(ctx, "System prompt", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("ChatWithTools() = %q, want ok", got)
	}

	if captured["model"] != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6", captured["model"])
	}
	messages, ok := captured["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v, want system + user", captured["messages"])
	}
	systemMessage, ok := messages[0].(map[string]any)
	if !ok || systemMessage["role"] != "system" || systemMessage["content"] != "System prompt" {
		t.Fatalf("system message = %#v, want text-only system content", messages[0])
	}
	if _, ok := systemMessage["content"].(string); !ok {
		t.Fatalf("system content type = %T, want string", systemMessage["content"])
	}
	userMessage, ok := messages[1].(map[string]any)
	if !ok || userMessage["role"] != "user" || userMessage["content"] != "hello" {
		t.Fatalf("user message = %#v, want text-only user content", messages[1])
	}
	if _, ok := userMessage["content"].(string); !ok {
		t.Fatalf("user content type = %T, want string", userMessage["content"])
	}
	if captured["max_completion_tokens"] != float64(32768) {
		t.Fatalf("max_completion_tokens = %#v, want 32768", captured["max_completion_tokens"])
	}
	if _, ok := captured["max_tokens"]; ok {
		t.Fatal("max_tokens should be omitted")
	}
	for _, field := range []string{"temperature", "top_p", "n", "presence_penalty", "frequency_penalty", "prompt_cache_retention"} {
		if _, ok := captured[field]; ok {
			t.Fatalf("%s should be omitted", field)
		}
	}
	if captured["stream"] != true {
		t.Fatalf("stream = %#v, want true", captured["stream"])
	}
	streamOptions, ok := captured["stream_options"].(map[string]any)
	if !ok || streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %#v, want include_usage=true", captured["stream_options"])
	}
	if key, ok := captured["prompt_cache_key"].(string); !ok || key == "" {
		t.Fatalf("prompt_cache_key = %#v, want non-empty string", captured["prompt_cache_key"])
	}
	thinking, ok := captured["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want disabled", captured["thinking"])
	}
}

func TestChatWithTools_ThinkingAndToolChoicePolicy(t *testing.T) {
	tests := []struct {
		name           string
		model          string
		thinking       bool
		wantThinking   string
		wantKeep       string
		wantNoKeep     bool
		wantNoThinking bool
		wantAutoTool   bool
	}{
		{
			name:         "k2.6 thinking off allows forced tool choice",
			model:        "kimi-k2.6",
			thinking:     false,
			wantThinking: "disabled",
		},
		{
			name:         "k2.6 thinking on rounds forced tool choice to auto",
			model:        "kimi-k2.6",
			thinking:     true,
			wantThinking: "enabled",
			wantKeep:     "all",
			wantAutoTool: true,
		},
		{
			name:         "k2.5 thinking on sends enabled thinking",
			model:        "kimi-k2.5",
			thinking:     true,
			wantThinking: "enabled",
			wantNoKeep:   true,
			wantAutoTool: true,
		},
		{
			name:           "forced thinking model never sends disabled",
			model:          "kimi-k2-thinking",
			thinking:       false,
			wantNoThinking: true,
			wantAutoTool:   true,
		},
		{
			name:           "k2.7 code never sends disabled and rounds forced tool choice to auto",
			model:          "kimi-k2.7-code",
			thinking:       false,
			wantNoThinking: true,
			wantAutoTool:   true,
		},
		{
			name:         "forced thinking model can send explicit enabled",
			model:        "kimi-k2-thinking",
			thinking:     true,
			wantThinking: "enabled",
			wantNoKeep:   true,
			wantAutoTool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured map[string]any
			server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
			})
			t.Setenv("KIMI_API_URL", server.URL)
			t.Setenv("KIMI_FUNCTION_CALLING", "")

			p := newKimiReadFileToolProviderForTest()
			ctx, _, _ := newKimiTestContext(t, tt.thinking)
			if _, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hi"}}, tt.model); err != nil {
				t.Fatalf("ChatWithTools() error = %v", err)
			}

			if tt.wantNoThinking {
				if _, ok := captured["thinking"]; ok {
					t.Fatalf("thinking = %#v, want absent", captured["thinking"])
				}
			} else {
				thinking, ok := captured["thinking"].(map[string]any)
				if !ok || thinking["type"] != tt.wantThinking {
					t.Fatalf("thinking = %#v, want type %q", captured["thinking"], tt.wantThinking)
				}
				if tt.wantKeep != "" && thinking["keep"] != tt.wantKeep {
					t.Fatalf("thinking.keep = %#v, want %q", thinking["keep"], tt.wantKeep)
				}
				if tt.wantNoKeep {
					if _, ok := thinking["keep"]; ok {
						t.Fatalf("thinking.keep = %#v, want absent", thinking["keep"])
					}
				}
			}

			if tt.wantAutoTool {
				if captured["tool_choice"] != "auto" {
					t.Fatalf("tool_choice = %#v, want auto", captured["tool_choice"])
				}
				return
			}
			toolChoice, ok := captured["tool_choice"].(map[string]any)
			if !ok {
				t.Fatalf("tool_choice = %#v, want forced function object", captured["tool_choice"])
			}
			function, ok := toolChoice["function"].(map[string]any)
			if !ok || function["name"] != "read_file" {
				t.Fatalf("tool_choice.function = %#v, want read_file", toolChoice["function"])
			}
		})
	}
}

func TestKimiThinkingConfig_UsesCatalogModelPayloadShape(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	cfg.SetProviderModelConfig("moonshot", config.ProviderModelConfig{
		DefaultModel: "corp-kimi",
		CatalogModel: "kimi-k2.5",
	})
	ctx := config.WithContext(context.Background(), cfg)

	extraFields, thinkingActive, _ := kimiThinkingConfig(ctx, "moonshot", "corp-kimi")
	if !thinkingActive {
		t.Fatal("thinkingActive = false, want true")
	}
	thinking, ok := extraFields["thinking"].(map[string]any)
	if !ok || thinking["type"] != "enabled" {
		t.Fatalf("thinking = %#v, want enabled", extraFields["thinking"])
	}
	if _, ok := thinking["keep"]; ok {
		t.Fatalf("thinking.keep = %#v, want absent for kimi-k2.5 catalog model", thinking["keep"])
	}
}

func TestKimiThinkingConfig_K27CatalogModelOmitsThinkingField(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = false
	cfg.SetProviderModelConfig("moonshot", config.ProviderModelConfig{
		DefaultModel: "corp-kimi-code",
		CatalogModel: "kimi-k2.7-code",
	})
	ctx := config.WithContext(context.Background(), cfg)

	extraFields, thinkingActive, spinnerSuffix := kimiThinkingConfig(ctx, "moonshot", "corp-kimi-code")
	if !thinkingActive {
		t.Fatal("thinkingActive = false, want true")
	}
	if spinnerSuffix != "Reasoner" {
		t.Fatalf("spinnerSuffix = %q, want Reasoner", spinnerSuffix)
	}
	if extraFields != nil {
		t.Fatalf("extraFields = %#v, want nil for kimi-k2.7-code catalog model", extraFields)
	}
}

func TestChatWithTools_FunctionCallingDisabledOmitsTools(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("KIMI_API_URL", server.URL)

	p := newKimiReadFileToolProviderForTest()
	ctx, _, _ := newKimiTestContext(t, false)
	if _, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hi"}}, "kimi-k2.6"); err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	assertKimiToolPayloadOmitted(t, captured)
}

func TestChatWithTools_RequestToolUseDisabledOmitsTools(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("KIMI_API_URL", server.URL)

	p := newKimiReadFileToolProviderForTest()
	ctx, _, _ := newKimiTestContext(t, false)
	ctx = api.WithToolUseDisabled(ctx)
	if _, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hi"}}, "kimi-k2.6"); err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	assertKimiToolPayloadOmitted(t, captured)
}

func TestChatWithTools_DoesNotIncludeBuiltinWebSearch(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("KIMI_API_URL", server.URL)

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{Name: "read_file", Description: "read", Parameters: map[string]any{"type": "object"}}})
	ctx, _, _ := newKimiTestContext(t, false)
	if _, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hi"}}, "kimi-k2.6"); err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}

	toolsPayload, ok := captured["tools"].([]any)
	if !ok || len(toolsPayload) != 1 {
		t.Fatalf("tools = %#v, want only regular function tool", captured["tools"])
	}
	tool, ok := toolsPayload[0].(map[string]any)
	if !ok {
		t.Fatalf("tool payload = %#v, want object", toolsPayload[0])
	}
	if tool["type"] == "builtin_function" {
		t.Fatalf("tools = %#v, want ChatWithTools to keep %s out of regular tool payloads", captured["tools"], kimiWebSearchToolName)
	}
	function, ok := tool["function"].(map[string]any)
	if !ok || function["name"] == kimiWebSearchToolName {
		t.Fatalf("tool.function = %#v, want no %s", tool["function"], kimiWebSearchToolName)
	}
}

func TestChatWithTools_MoonshotAliasUsesAliasModelConfig(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("KIMI_API_URL", server.URL)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("moonshot", config.ProviderModelConfig{
		DefaultModel:    "moonshot-custom",
		MaxOutputTokens: 12345,
	})
	ctx, _, _ := newKimiTestContext(t, false)
	ctx = config.WithContext(ctx, cfg)

	p := newProvider("test-key", "moonshot")
	got, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hi"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("ChatWithTools() = %q, want ok", got)
	}
	if captured["model"] != "moonshot-custom" {
		t.Fatalf("model = %q, want moonshot-custom", captured["model"])
	}
	if captured["max_completion_tokens"] != float64(12345) {
		t.Fatalf("max_completion_tokens = %#v, want 12345", captured["max_completion_tokens"])
	}
}

func TestChatWithTools_ThinkingToolLoopReplayRequestShape(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "")

	const (
		toolCallID     = "call_1"
		toolName       = "read_file"
		toolPath       = "README.md"
		reasoningText  = "調査します"
		toolResultText = "README contents"
	)

	var captured []map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		captured = append(captured, body)

		switch len(captured) {
		case 1:
			kimiStreamingHandler([]string{
				`{"choices":[{"delta":{"reasoning_content":"` + reasoningText + `"}}]}`,
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"` + toolCallID + `","type":"function","function":{"name":"` + toolName + `","arguments":"{\"path\":\"` + toolPath + `\"}"}}]}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
			})(w, r)
		case 2:
			kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"完了しました"}}]}`})(w, r)
		default:
			t.Fatalf("unexpected request count %d", len(captured))
		}
	})
	t.Setenv("KIMI_API_URL", server.URL)

	ctx, _, _ := newKimiTestContext(t, true)
	ctx = api.WithPromptCacheScope(ctx, api.PromptCacheScope{SessionID: "session-loop"})
	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{
		Name:        toolName,
		Description: "Read a file",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []any{"path"},
		},
	}})
	p.SetToolChoice(toolName)

	history := []api.Message{{Role: "user", Content: "README を確認して"}}
	response, err := p.ChatWithTools(ctx, "System prompt", history, "kimi-k2.6")
	if err != nil {
		t.Fatalf("first ChatWithTools() error = %v", err)
	}
	toolCalls := tools.ParseToolCalls(response)
	if len(toolCalls) != 1 {
		t.Fatalf("ParseToolCalls(first response) len = %d, want 1; response=%q", len(toolCalls), response)
	}
	assertParsedKimiReplayToolCall(t, toolCalls[0], toolCallID, toolName, toolPath)

	history = appendKimiToolReplayHistory(history, p.LastReasoningContent(), toolCalls[0], toolResultText)

	got, err := p.ChatWithTools(ctx, "System prompt", history, "kimi-k2.6")
	if err != nil {
		t.Fatalf("second ChatWithTools() error = %v", err)
	}
	if got != "完了しました" {
		t.Fatalf("second ChatWithTools() = %q, want 完了しました", got)
	}

	if len(captured) != 2 {
		t.Fatalf("captured request count = %d, want 2", len(captured))
	}
	assertKimiThinkingToolLoopRequestFields(t, captured[0], "first")
	assertKimiThinkingToolLoopRequestFields(t, captured[1], "second")
	firstCacheKey, _ := captured[0]["prompt_cache_key"].(string)
	secondCacheKey, _ := captured[1]["prompt_cache_key"].(string)
	if firstCacheKey == "" || secondCacheKey == "" || firstCacheKey != secondCacheKey {
		t.Fatalf("prompt_cache_key first=%q second=%q, want non-empty equal keys", firstCacheKey, secondCacheKey)
	}
	if !strings.HasPrefix(firstCacheKey, "xelyon:kimi:v1:") {
		t.Fatalf("prompt_cache_key = %q, want session-aware Kimi key", firstCacheKey)
	}

	assertKimiToolReplayMessages(t, captured[1]["messages"], reasoningText, toolCallID, toolName, toolPath, toolResultText)
}

func assertParsedKimiReplayToolCall(t *testing.T, toolCall *tools.ToolCall, wantID, wantTool, wantPath string) {
	t.Helper()
	if toolCall.ID != wantID || toolCall.Tool != wantTool || toolCall.Args["path"] != wantPath {
		t.Fatalf("parsed tool call = %+v, want %s %s %s", toolCall, wantTool, wantID, wantPath)
	}
}

func appendKimiToolReplayHistory(history []api.Message, reasoningContent string, toolCall *tools.ToolCall, result string) []api.Message {
	history = append(history, api.Message{
		Role:             "assistant",
		ReasoningContent: reasoningContent,
		ToolCalls: []api.OpenAIToolCall{{
			ID:   toolCall.ID,
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      toolCall.Tool,
				Arguments: toolruntime.ArgsToJSON(toolCall.RawArgs),
			},
		}},
	})
	return append(history, toolruntime.BuildToolResultMessage(
		toolCall,
		result,
		toolruntime.FormatTextToolResultContent(toolCall.Tool, result),
	))
}

func assertKimiToolReplayMessages(t *testing.T, rawMessages any, wantReasoning, wantCallID, wantToolName, wantPath, wantResult string) {
	t.Helper()
	messages, ok := rawMessages.([]any)
	if !ok || len(messages) != 4 {
		t.Fatalf("second messages = %#v, want system + user + assistant + tool", rawMessages)
	}
	assistant, ok := messages[2].(map[string]any)
	if !ok || assistant["role"] != "assistant" {
		t.Fatalf("second assistant message = %#v, want role assistant", messages[2])
	}
	if assistant["reasoning_content"] != wantReasoning {
		t.Fatalf("assistant reasoning_content = %#v, want %s", assistant["reasoning_content"], wantReasoning)
	}
	toolCallsPayload, ok := assistant["tool_calls"].([]any)
	if !ok || len(toolCallsPayload) != 1 {
		t.Fatalf("assistant tool_calls = %#v, want one tool call", assistant["tool_calls"])
	}
	toolCall, ok := toolCallsPayload[0].(map[string]any)
	if !ok || toolCall["id"] != wantCallID || toolCall["type"] != "function" {
		t.Fatalf("assistant tool_call = %#v, want %s function", toolCallsPayload[0], wantCallID)
	}
	function, ok := toolCall["function"].(map[string]any)
	if !ok || function["name"] != wantToolName || function["arguments"] != `{"path":"`+wantPath+`"}` {
		t.Fatalf("assistant tool_call.function = %#v, want %s %s arguments", toolCall["function"], wantToolName, wantPath)
	}
	toolResult, ok := messages[3].(map[string]any)
	if !ok || toolResult["role"] != "tool" || toolResult["tool_call_id"] != wantCallID || toolResult["content"] != wantResult {
		t.Fatalf("second tool result message = %#v, want role/tool_call_id/content preserved", messages[3])
	}
	if _, ok := toolResult["tool_name"]; ok {
		t.Fatalf("second tool result message = %#v, want no tool_name in Kimi replay payload", toolResult)
	}
}

func assertKimiThinkingToolLoopRequestFields(t *testing.T, body map[string]any, label string) {
	t.Helper()
	thinking, ok := body["thinking"].(map[string]any)
	if !ok || thinking["type"] != "enabled" || thinking["keep"] != "all" {
		t.Fatalf("%s thinking = %#v, want enabled keep=all", label, body["thinking"])
	}
	if body["tool_choice"] != "auto" {
		t.Fatalf("%s tool_choice = %#v, want auto", label, body["tool_choice"])
	}
}
