package kimi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newKimiTestContext(t *testing.T, thinking bool) (context.Context, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = thinking
	cfg.Thinking.Level = "high"

	var out bytes.Buffer
	var errOut bytes.Buffer
	runtime := ui.NewRuntime(strings.NewReader(""), &out, &errOut)
	ctx := ui.WithRuntime(context.Background(), runtime)
	ctx = config.WithContext(ctx, cfg)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	return ctx, &out, &errOut
}

func mockKimiAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func kimiStreamingHandler(chunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		for _, chunk := range chunks {
			_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}
}

func newKimiSSEResponse(chunks ...string) *http.Response {
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

func TestNewAndURLOverride(t *testing.T) {
	t.Setenv("KIMI_API_URL", "")
	if got := New("test-key").APIURL(); got != defaultKimiURL {
		t.Fatalf("APIURL() = %q, want %q", got, defaultKimiURL)
	}

	t.Setenv("KIMI_API_URL", "https://proxy.example/v1/chat/completions")
	if got := New("test-key").APIURL(); got != "https://proxy.example/v1/chat/completions" {
		t.Fatalf("APIURL() = %q, want custom URL", got)
	}
}

func TestProviderRegistrationAndMoonshotAlias(t *testing.T) {
	t.Setenv("MOONSHOT_API_KEY", "")
	if _, err := api.NewProvider("kimi"); err == nil || err.Error() != "MOONSHOT_API_KEY not set" {
		t.Fatalf("NewProvider(kimi) error = %v, want MOONSHOT_API_KEY not set", err)
	}

	t.Setenv("MOONSHOT_API_KEY", "test-key")
	for _, tt := range []struct {
		providerName  string
		wantConfigKey string
	}{
		{providerName: "kimi", wantConfigKey: "kimi"},
		{providerName: "moonshot", wantConfigKey: "moonshot"},
	} {
		providerName := tt.providerName
		t.Run(providerName, func(t *testing.T) {
			p, err := api.NewProvider(providerName)
			if err != nil {
				t.Fatalf("NewProvider(%q) error = %v", providerName, err)
			}
			if p.Name() != "Kimi" {
				t.Fatalf("Name() = %q, want Kimi", p.Name())
			}
			aware, ok := p.(interface{ ProviderConfigKey() string })
			if !ok {
				t.Fatalf("NewProvider(%q) does not expose ProviderConfigKey", providerName)
			}
			if got := aware.ProviderConfigKey(); got != tt.wantConfigKey {
				t.Fatalf("ProviderConfigKey() = %q, want %q", got, tt.wantConfigKey)
			}
		})
	}
}

func TestProviderCapabilities(t *testing.T) {
	p := New("test-key")
	if p.SupportsImages() {
		t.Fatal("SupportsImages() = true, want false")
	}

	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	if p.IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() = true, want false when KIMI_FUNCTION_CALLING=0")
	}
	t.Setenv("KIMI_FUNCTION_CALLING", "")
	if !p.IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() = false, want true by default")
	}
}

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

			p := New("test-key")
			p.SetMCPTools([]api.ToolDefinition{{Name: "read_file", Description: "read", Parameters: map[string]any{"type": "object"}}})
			p.SetToolChoice("read_file")
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

	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{Name: "read_file"}})
	p.SetToolChoice("read_file")
	ctx, _, _ := newKimiTestContext(t, false)
	if _, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "hi"}}, "kimi-k2.6"); err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if _, ok := captured["tools"]; ok {
		t.Fatalf("tools = %#v, want absent", captured["tools"])
	}
	if _, ok := captured["tool_choice"]; ok {
		t.Fatalf("tool_choice = %#v, want absent", captured["tool_choice"])
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

func TestHandleStreamingResponse_ContentReasoningToolCallsAndUsage(t *testing.T) {
	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	resp := newKimiSSEResponse(
		`{"choices":[{"delta":{"reasoning_content":"Think"}}]}`,
		`{"choices":[{"delta":{"reasoning_content":" deeply"}}]}`,
		`{"choices":[{"delta":{"content":"Answer "}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"main.go\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls","usage":{"prompt_tokens":12,"completion_tokens":8,"cached_tokens":5}}]}`,
	)
	defer resp.Body.Close()

	ctx, out, _ := newKimiTestContext(t, true)
	got, err := p.handleStreamingResponse(ctx, resp, ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if !strings.Contains(got, "Answer ") || !strings.Contains(got, `"tool":"read_file"`) || !strings.Contains(got, `"path":"main.go"`) {
		t.Fatalf("handleStreamingResponse() = %q, want text and tool JSON", got)
	}
	if p.LastReasoningContent() != "Think deeply" {
		t.Fatalf("LastReasoningContent() = %q, want Think deeply", p.LastReasoningContent())
	}
	if !strings.Contains(out.String(), "Think deeply") {
		t.Fatalf("reasoning output = %q, want streamed reasoning", out.String())
	}
	if gotUsage.InputTokens != 12 || gotUsage.OutputTokens != 8 || gotUsage.CachedInputTokens != 5 || gotUsage.ThinkingTokens != 0 {
		t.Fatalf("usage = %+v, want input=12 output=8 cached=5 thinking=0", gotUsage)
	}
}

func TestChatWithImage_WarnsAndFallsBackToText(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	var captured openaicompat.ChatCompletionsRequest
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("KIMI_API_URL", server.URL)

	ctx, out, _ := newKimiTestContext(t, false)
	got, err := New("test-key").ChatWithImage(ctx, "System", nil, "describe image", &api.ImageData{Base64: "abc", MediaType: "image/png"}, "kimi-k2.6")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("ChatWithImage() = %q, want ok", got)
	}
	if !strings.Contains(out.String(), "Kimi does not support image input") {
		t.Fatalf("warning output = %q, want image warning", out.String())
	}
	if len(captured.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(captured.Messages))
	}
	if captured.Messages[1].Role != "user" || captured.Messages[1].Content != "describe image" {
		t.Fatalf("last message = %#v, want text-only user message", captured.Messages[1])
	}
}
