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
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
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

const kimiTestPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="

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
	if !p.SupportsImages() {
		t.Fatal("SupportsImages() = false, want true")
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

func TestWebSearchWithContext_BuiltinToolLoopPayloadAndUsage(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	const toolArguments = `{"query":"Moonshot AI","usage":{"total_tokens":4}}`

	var captured []map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		captured = append(captured, body)

		switch len(captured) {
		case 1:
			kimiStreamingHandler([]string{
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_web","type":"builtin_function","function":{"name":"$web_search","arguments":"{\"query\":\"Moonshot AI\",\"usage\":{\"total_tokens\":4}}"}}]}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":11,"completion_tokens":2,"cached_tokens":3}}`,
			})(w, r)
		case 2:
			kimiStreamingHandler([]string{
				`{"choices":[{"delta":{"content":"Summary: Kimi search result."}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":21,"completion_tokens":6,"cached_tokens":5}}`,
			})(w, r)
		default:
			t.Fatalf("unexpected request count %d", len(captured))
		}
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{
		DefaultModel:    "kimi-search-test",
		MaxOutputTokens: 123,
	})
	ctx := config.WithContext(context.Background(), cfg)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	ctx = api.WithPromptCacheScope(ctx, api.PromptCacheScope{SessionID: "web-search-session"})
	var usages []api.Usage
	ctx = websearch.WithUsageCallback(ctx, func(usage api.Usage) {
		usages = append(usages, usage)
	})

	got, err := WebSearchWithContext(ctx, "Moonshot AI Context Caching", "")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
	if got != "Summary: Kimi search result." {
		t.Fatalf("WebSearchWithContext() = %q, want final content", got)
	}

	if len(captured) != 2 {
		t.Fatalf("captured request count = %d, want 2", len(captured))
	}
	assertKimiWebSearchBasePayload(t, captured[0])
	assertKimiWebSearchBasePayload(t, captured[1])
	assertKimiWebSearchLoopMessages(t, captured[1]["messages"], toolArguments)
	if _, ok := captured[0]["tool_choice"]; ok {
		t.Fatalf("tool_choice = %#v, want omitted for %s", captured[0]["tool_choice"], kimiWebSearchToolName)
	}
	if _, ok := captured[0]["max_tokens"]; ok {
		t.Fatal("max_tokens should be omitted")
	}
	if len(usages) != 3 {
		t.Fatalf("usage callback count = %d, want 3", len(usages))
	}
	if usages[0].InputTokens != 11 || usages[0].OutputTokens != 2 || usages[0].CachedInputTokens != 3 {
		t.Fatalf("first usage = %+v, want input=11 output=2 cached=3", usages[0])
	}
	if usages[1].WebSearchCalls != 1 || usages[1].WebSearchResultTokens != 4 || usages[1].StorageCost != kimiWebSearchCallFeeUSD {
		t.Fatalf("web search usage = %+v, want one call, 4 observed result tokens, fee %.4f", usages[1], kimiWebSearchCallFeeUSD)
	}
	if usages[1].InputTokens != 0 || usages[1].OutputTokens != 0 {
		t.Fatalf("web search usage tokens = input %d output %d, want no token double count", usages[1].InputTokens, usages[1].OutputTokens)
	}
	if usages[2].InputTokens != 21 || usages[2].OutputTokens != 6 || usages[2].CachedInputTokens != 5 {
		t.Fatalf("second token usage = %+v, want input=21 output=6 cached=5", usages[2])
	}
}

func TestKimiWebSearchToolCallUsage_ParsesResultTokenObservations(t *testing.T) {
	usage := kimiWebSearchToolCallUsage([]api.OpenAIToolCall{
		{Function: api.OpenAIToolCallFunction{Name: kimiWebSearchToolName, Arguments: `{"usage":{"total_tokens":12}}`}},
		{Function: api.OpenAIToolCallFunction{Name: kimiWebSearchToolName, Arguments: `{"total_tokens":7}`}},
		{Function: api.OpenAIToolCallFunction{Name: kimiWebSearchToolName, Arguments: `not-json`}},
		{Function: api.OpenAIToolCallFunction{Name: "other", Arguments: `{"total_tokens":99}`}},
	})

	if usage == nil {
		t.Fatal("kimiWebSearchToolCallUsage() = nil, want usage")
	}
	if usage.WebSearchCalls != 3 {
		t.Fatalf("WebSearchCalls = %d, want 3", usage.WebSearchCalls)
	}
	if usage.WebSearchResultTokens != 19 {
		t.Fatalf("WebSearchResultTokens = %d, want 19", usage.WebSearchResultTokens)
	}
	if usage.StorageCost != 3*kimiWebSearchCallFeeUSD {
		t.Fatalf("StorageCost = %f, want %f", usage.StorageCost, 3*kimiWebSearchCallFeeUSD)
	}
	if usage.InputTokens != 0 || usage.OutputTokens != 0 {
		t.Fatalf("token usage = input %d output %d, want no token double count", usage.InputTokens, usage.OutputTokens)
	}
}

func TestWebSearchWithContext_InvalidToolArgumentsStillReplayAndReportCallFee(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
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
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_web","type":"builtin_function","function":{"name":"$web_search","arguments":"not-json"}}]}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
			})(w, r)
		case 2:
			kimiStreamingHandler([]string{
				`{"choices":[{"delta":{"content":"ok after invalid arguments replay"}}]}`,
				`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			})(w, r)
		default:
			t.Fatalf("unexpected request count %d", len(captured))
		}
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	ctx = websearch.WithUsageCallback(ctx, func(usage api.Usage) {
		if usage.WebSearchCalls != 1 || usage.StorageCost != kimiWebSearchCallFeeUSD {
			t.Fatalf("web search usage = %+v, want one fee observation", usage)
		}
	})

	got, err := WebSearchWithContext(ctx, "invalid arguments", "kimi-k2.6")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
	if got != "ok after invalid arguments replay" {
		t.Fatalf("WebSearchWithContext() = %q, want final content", got)
	}
	if len(captured) != 2 {
		t.Fatalf("captured request count = %d, want 2", len(captured))
	}
	assertKimiWebSearchLoopMessages(t, captured[1]["messages"], "not-json")
}

func TestWebSearchWithContext_MoonshotAliasUsesAliasDefaultModel(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{
			`{"choices":[{"delta":{"content":"alias result"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`,
		})(w, r)
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("moonshot", config.ProviderModelConfig{
		DefaultModel:    "moonshot-search-default",
		MaxOutputTokens: 77,
	})
	ctx := config.WithContext(context.Background(), cfg)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	got, err := websearch.SearchWithContext(ctx, "moonshot", "alias query", "")
	if err != nil {
		t.Fatalf("SearchWithContext(moonshot) error = %v", err)
	}
	if got != "alias result" {
		t.Fatalf("SearchWithContext(moonshot) = %q, want alias result", got)
	}
	if captured["model"] != "moonshot-search-default" {
		t.Fatalf("model = %#v, want moonshot alias default", captured["model"])
	}
	if captured["max_completion_tokens"] != float64(77) {
		t.Fatalf("max_completion_tokens = %#v, want 77", captured["max_completion_tokens"])
	}
}

func TestWebSearchWithContext_ForcedThinkingModelOmitsDisabledThinking(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{
			`{"choices":[{"delta":{"content":"forced thinking model result"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`,
		})(w, r)
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	got, err := WebSearchWithContext(ctx, "forced thinking", "kimi-k2-thinking")
	if err != nil {
		t.Fatalf("WebSearchWithContext() error = %v", err)
	}
	if got != "forced thinking model result" {
		t.Fatalf("WebSearchWithContext() = %q, want forced thinking model result", got)
	}
	if captured["model"] != "kimi-k2-thinking" {
		t.Fatalf("model = %#v, want kimi-k2-thinking", captured["model"])
	}
	if _, ok := captured["thinking"]; ok {
		t.Fatalf("thinking = %#v, want omitted for forced thinking model", captured["thinking"])
	}
}

func TestBuildKimiWebSearchRequest_OmitsDisabledThinkingForForcedCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("moonshot", config.ProviderModelConfig{
		DefaultModel: "corp-kimi",
		CatalogModel: "kimi-k2-thinking",
	})
	ctx := config.WithContext(context.Background(), cfg)
	req := buildKimiWebSearchRequest(ctx, initialKimiWebSearchMessages("catalog forced"), "corp-kimi", "moonshot")
	if req.Model != "corp-kimi" {
		t.Fatalf("model = %q, want corp-kimi", req.Model)
	}
	if req.Thinking != nil {
		t.Fatalf("thinking = %#v, want omitted when catalog model is forced-thinking", req.Thinking)
	}
}

func TestWebSearchWithContext_ErrorsAfterMaxToolLoops(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	requests := 0
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		kimiStreamingHandler([]string{
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_web","type":"builtin_function","function":{"name":"$web_search","arguments":"{\"query\":\"loop\"}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		})(w, r)
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	var webSearchCalls int
	var fee float64
	ctx = websearch.WithUsageCallback(ctx, func(usage api.Usage) {
		webSearchCalls += usage.WebSearchCalls
		fee += usage.StorageCost
	})
	_, err := WebSearchWithContext(ctx, "loop", "kimi-k2.6")
	if err == nil {
		t.Fatal("WebSearchWithContext() error = nil, want max loop error")
	}
	if !strings.Contains(err.Error(), "did not complete within 3 requests") {
		t.Fatalf("error = %v, want max request message", err)
	}
	if requests != kimiWebSearchMaxRequests {
		t.Fatalf("requests = %d, want %d", requests, kimiWebSearchMaxRequests)
	}
	if webSearchCalls != kimiWebSearchMaxRequests {
		t.Fatalf("webSearchCalls = %d, want %d charged tool call observations", webSearchCalls, kimiWebSearchMaxRequests)
	}
	if fee != float64(kimiWebSearchMaxRequests)*kimiWebSearchCallFeeUSD {
		t.Fatalf("fee = %f, want %f", fee, float64(kimiWebSearchMaxRequests)*kimiWebSearchCallFeeUSD)
	}
}

func TestWebSearchWithContext_ErrorsOnIncompleteFinishReasonWithContent(t *testing.T) {
	t.Setenv(kimiAPIKeyEnv, "test-key")
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		kimiStreamingHandler([]string{
			`{"choices":[{"delta":{"content":"partial Kimi search result"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"length"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`,
		})(w, r)
	})
	t.Setenv(kimiAPIURLEnv, server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	got, err := WebSearchWithContext(ctx, "partial", "kimi-k2.6")
	if err == nil {
		t.Fatal("WebSearchWithContext() error = nil, want incomplete finish_reason error")
	}
	if got != "" {
		t.Fatalf("WebSearchWithContext() = %q, want empty result on incomplete finish_reason", got)
	}
	if !strings.Contains(err.Error(), `finish_reason "length"`) {
		t.Fatalf("error = %v, want finish_reason length", err)
	}
}

func assertKimiWebSearchBasePayload(t *testing.T, body map[string]any) {
	t.Helper()
	if body["model"] != "kimi-search-test" {
		t.Fatalf("model = %#v, want kimi-search-test", body["model"])
	}
	if body["max_completion_tokens"] != float64(123) {
		t.Fatalf("max_completion_tokens = %#v, want 123", body["max_completion_tokens"])
	}
	if body["stream"] != true {
		t.Fatalf("stream = %#v, want true", body["stream"])
	}
	streamOptions, ok := body["stream_options"].(map[string]any)
	if !ok || streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %#v, want include_usage=true", body["stream_options"])
	}
	if key, ok := body["prompt_cache_key"].(string); !ok || key == "" || !strings.HasPrefix(key, "xelyon:kimi:v1:") {
		t.Fatalf("prompt_cache_key = %#v, want session-aware Kimi key", body["prompt_cache_key"])
	}
	thinking, ok := body["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want disabled", body["thinking"])
	}
	toolsPayload, ok := body["tools"].([]any)
	if !ok || len(toolsPayload) != 1 {
		t.Fatalf("tools = %#v, want one builtin tool", body["tools"])
	}
	tool, ok := toolsPayload[0].(map[string]any)
	if !ok || tool["type"] != "builtin_function" {
		t.Fatalf("tool = %#v, want builtin_function", toolsPayload[0])
	}
	function, ok := tool["function"].(map[string]any)
	if !ok || function["name"] != kimiWebSearchToolName {
		t.Fatalf("tool.function = %#v, want %s", tool["function"], kimiWebSearchToolName)
	}
}

func assertKimiWebSearchLoopMessages(t *testing.T, rawMessages any, wantToolContent string) {
	t.Helper()
	messages, ok := rawMessages.([]any)
	if !ok || len(messages) != 4 {
		t.Fatalf("messages = %#v, want system + user + assistant + tool", rawMessages)
	}
	assistant, ok := messages[2].(map[string]any)
	if !ok || assistant["role"] != "assistant" {
		t.Fatalf("assistant message = %#v, want role assistant", messages[2])
	}
	toolCallsPayload, ok := assistant["tool_calls"].([]any)
	if !ok || len(toolCallsPayload) != 1 {
		t.Fatalf("assistant tool_calls = %#v, want one tool call", assistant["tool_calls"])
	}
	toolCall, ok := toolCallsPayload[0].(map[string]any)
	if !ok || toolCall["id"] != "call_web" || toolCall["type"] != "builtin_function" {
		t.Fatalf("assistant tool_call = %#v, want returned builtin_function call", toolCallsPayload[0])
	}
	function, ok := toolCall["function"].(map[string]any)
	if !ok || function["name"] != kimiWebSearchToolName || function["arguments"] != wantToolContent {
		t.Fatalf("assistant tool_call.function = %#v, want %s with exact arguments", toolCall["function"], kimiWebSearchToolName)
	}
	toolMessage, ok := messages[3].(map[string]any)
	if !ok || toolMessage["role"] != "tool" || toolMessage["tool_call_id"] != "call_web" || toolMessage["name"] != kimiWebSearchToolName {
		t.Fatalf("tool message = %#v, want role/tool_call_id/name", messages[3])
	}
	if toolMessage["content"] != wantToolContent {
		t.Fatalf("tool message content = %#v, want exact arguments %s", toolMessage["content"], wantToolContent)
	}
	if _, ok := toolMessage["tool_name"]; ok {
		t.Fatalf("tool message = %#v, want Kimi name field not tool_name", toolMessage)
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

func TestChatWithImage_BuildsMultimodalRequest(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{
			`{"choices":[{"delta":{"content":"ok"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop","usage":{"prompt_tokens":9,"completion_tokens":4,"cached_tokens":3}}]}`,
		})(w, r)
	})
	t.Setenv("KIMI_API_URL", server.URL)

	ctx, out, _ := newKimiTestContext(t, true)
	p := New("test-key")
	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	got, err := p.ChatWithImage(ctx, "System", []api.Message{{Role: "assistant", Content: "previous"}}, "describe image", &api.ImageData{
		Base64:    kimiTestPNGBase64,
		MediaType: "image/png",
	}, "kimi-k2.6")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("ChatWithImage() = %q, want ok", got)
	}
	if strings.Contains(out.String(), "does not support image input") {
		t.Fatalf("warning output = %q, want no unsupported image warning", out.String())
	}
	if captured["model"] != "kimi-k2.6" {
		t.Fatalf("model = %#v, want kimi-k2.6", captured["model"])
	}
	if captured["max_completion_tokens"] != float64(32768) {
		t.Fatalf("max_completion_tokens = %#v, want 32768", captured["max_completion_tokens"])
	}
	streamOptions, ok := captured["stream_options"].(map[string]any)
	if !ok || streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %#v, want include_usage=true", captured["stream_options"])
	}
	if key, ok := captured["prompt_cache_key"].(string); !ok || key == "" {
		t.Fatalf("prompt_cache_key = %#v, want non-empty string", captured["prompt_cache_key"])
	}
	thinking, ok := captured["thinking"].(map[string]any)
	if !ok || thinking["type"] != "enabled" || thinking["keep"] != "all" {
		t.Fatalf("thinking = %#v, want enabled keep=all", captured["thinking"])
	}
	messages, ok := captured["messages"].([]any)
	if !ok || len(messages) != 3 {
		t.Fatalf("messages = %#v, want system + history + multimodal user", captured["messages"])
	}
	user, ok := messages[2].(map[string]any)
	if !ok || user["role"] != "user" {
		t.Fatalf("user message = %#v, want role user", messages[2])
	}
	content, ok := user["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content = %#v, want text + image parts", user["content"])
	}
	textPart, ok := content[0].(map[string]any)
	if !ok || textPart["type"] != "text" || textPart["text"] != "describe image" {
		t.Fatalf("text part = %#v, want describe image", content[0])
	}
	imagePart, ok := content[1].(map[string]any)
	if !ok || imagePart["type"] != "image_url" {
		t.Fatalf("image part = %#v, want image_url", content[1])
	}
	imageURL, ok := imagePart["image_url"].(map[string]any)
	if !ok || imageURL["url"] != "data:image/png;base64,"+kimiTestPNGBase64 {
		t.Fatalf("image_url = %#v, want Kimi data URL", imagePart["image_url"])
	}
	if gotUsage.InputTokens != 9 || gotUsage.OutputTokens != 4 || gotUsage.CachedInputTokens != 3 {
		t.Fatalf("usage = %+v, want input=9 output=4 cached=3", gotUsage)
	}
}

func TestChatWithImage_IncludesToolsWhenFunctionCallingEnabled(t *testing.T) {
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
	p.SetToolChoice("read_file")
	ctx, _, _ := newKimiTestContext(t, false)
	if _, err := p.ChatWithImage(ctx, "System", nil, "describe image", &api.ImageData{
		Base64:    kimiTestPNGBase64,
		MediaType: "image/png",
	}, "kimi-k2.6"); err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want one tool", captured["tools"])
	}
	toolChoice, ok := captured["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice = %#v, want forced function", captured["tool_choice"])
	}
	function, ok := toolChoice["function"].(map[string]any)
	if !ok || function["name"] != "read_file" {
		t.Fatalf("tool_choice.function = %#v, want read_file", toolChoice["function"])
	}
}

func TestChatWithImage_NoImageFallsBackToText(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	var captured openaicompat.ChatCompletionsRequest
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("KIMI_API_URL", server.URL)

	ctx, _, _ := newKimiTestContext(t, false)
	got, err := New("test-key").ChatWithImage(ctx, "System", nil, "describe image", nil, "kimi-k2.6")
	if err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("ChatWithImage() = %q, want ok", got)
	}
	if len(captured.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(captured.Messages))
	}
	if captured.Messages[1].Role != "user" || captured.Messages[1].Content != "describe image" {
		t.Fatalf("last message = %#v, want text-only user message", captured.Messages[1])
	}
}

func TestChatWithImage_InvalidImageInput(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Kimi API should not be called for invalid image input")
	})
	t.Setenv("KIMI_API_URL", server.URL)

	tests := []struct {
		name      string
		image     *api.ImageData
		wantError string
	}{
		{
			name:      "unsupported media type",
			image:     &api.ImageData{Base64: kimiTestPNGBase64, MediaType: "image/bmp"},
			wantError: "unsupported Kimi image media type",
		},
		{
			name:      "invalid base64",
			image:     &api.ImageData{Base64: "not-base64", MediaType: "image/png"},
			wantError: "invalid Kimi image base64 data",
		},
		{
			name:      "too large",
			image:     &api.ImageData{Base64: kimiTestPNGBase64, MediaType: "image/png", Size: api.MaxImageSize + 1},
			wantError: "kimi image input is too large",
		},
		{
			name:      "mismatched bytes",
			image:     &api.ImageData{Base64: "YWJj", MediaType: "image/png"},
			wantError: "invalid Kimi image bytes for media type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _, _ := newKimiTestContext(t, false)
			_, err := New("test-key").ChatWithImage(ctx, "System", nil, "describe image", tt.image, "kimi-k2.6")
			if err == nil {
				t.Fatal("ChatWithImage() error = nil, want invalid image error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("ChatWithImage() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}
