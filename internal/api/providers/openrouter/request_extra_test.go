package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newOpenRouterTestContext(t *testing.T, cfg *config.Config) (context.Context, *bytes.Buffer) {
	t.Helper()

	var out bytes.Buffer
	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(strings.NewReader(""), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	if cfg != nil {
		ctx = config.WithContext(ctx, cfg)
	}
	return ctx, &out
}

func TestChatWithTools_WarnsOnThinkingAndUsesForcedToolChoice(t *testing.T) {
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "")

	prevNoColor := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = prevNoColor }()

	var requestBody map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("json decode failed: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.ChatResponse{
			Choices: []api.Choice{{Message: api.Message{Content: "ok"}}},
		})
	})

	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	cfg.ProviderModels["openrouter"] = config.ProviderModelConfig{
		DefaultModel:    "openai/gpt-4-turbo",
		MaxOutputTokens: 123,
	}

	p := New("test-key")
	p.APIURL = server.URL
	p.SetToolChoice("read_file")

	ctx, out := newOpenRouterTestContext(t, cfg)
	got, err := p.ChatWithTools(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("ChatWithTools() = %q, want %q", got, "ok")
	}

	if requestBody["model"] != "openai/gpt-4-turbo" {
		t.Fatalf("model = %v, want openai/gpt-4-turbo", requestBody["model"])
	}
	if requestBody["max_tokens"] != float64(123) {
		t.Fatalf("max_tokens = %v, want 123", requestBody["max_tokens"])
	}
	streamOptions, ok := requestBody["stream_options"].(map[string]any)
	if !ok || streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %v, want include_usage=true", requestBody["stream_options"])
	}
	toolChoice, ok := requestBody["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice = %T, want map", requestBody["tool_choice"])
	}
	if toolChoice["type"] != "function" {
		t.Fatalf("tool_choice.type = %v, want function", toolChoice["type"])
	}
	functionBody, ok := toolChoice["function"].(map[string]any)
	if !ok || functionBody["name"] != "read_file" {
		t.Fatalf("tool_choice.function = %v, want read_file", toolChoice["function"])
	}
	if !strings.Contains(out.String(), "OpenRouter does not support Extended Thinking") {
		t.Fatalf("output = %q, want thinking warning", out.String())
	}
}

func TestChatWithImageRequest_BuildsStreamingMultimodalPayload(t *testing.T) {
	var requestBody map[string]any
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("HTTP-Referer") == "" {
			t.Fatal("HTTP-Referer header should be set")
		}
		if r.Header.Get("X-Title") != "XELYON CLI" {
			t.Fatalf("X-Title = %q, want XELYON CLI", r.Header.Get("X-Title"))
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("json decode failed: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Vision\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	})

	cfg := config.DefaultConfig()
	cfg.ProviderModels["openrouter"] = config.ProviderModelConfig{
		DefaultModel:    "openai/gpt-4o",
		MaxOutputTokens: 222,
	}

	p := New("test-key")
	p.APIURL = server.URL

	ctx, _ := newOpenRouterTestContext(t, cfg)
	got, err := p.chatWithImageRequest(ctx, "system prompt", []api.Message{{Role: "assistant", Content: "previous"}}, "describe image", &api.ImageData{
		MediaType: "image/png",
		Base64:    "dGVzdA==",
	}, "openai/gpt-4o")
	if err != nil {
		t.Fatalf("chatWithImageRequest() error = %v", err)
	}
	if got != "Vision" {
		t.Fatalf("chatWithImageRequest() = %q, want %q", got, "Vision")
	}

	if requestBody["model"] != "openai/gpt-4o" {
		t.Fatalf("model = %v, want openai/gpt-4o", requestBody["model"])
	}
	if requestBody["max_tokens"] != float64(222) {
		t.Fatalf("max_tokens = %v, want 222", requestBody["max_tokens"])
	}
	streamOptions, ok := requestBody["stream_options"].(map[string]any)
	if !ok || streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %v, want include_usage=true", requestBody["stream_options"])
	}

	messages, ok := requestBody["messages"].([]any)
	if !ok || len(messages) != 3 {
		t.Fatalf("messages = %v, want system + history + multimodal user", requestBody["messages"])
	}
	lastMessage, ok := messages[2].(map[string]any)
	if !ok {
		t.Fatalf("last message = %T, want map", messages[2])
	}
	content, ok := lastMessage["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content = %v, want text + image", lastMessage["content"])
	}
	if textPart, ok := content[0].(map[string]any); !ok || textPart["type"] != "text" || textPart["text"] != "describe image" {
		t.Fatalf("text part = %v, want describe image", content[0])
	}
	imagePart, ok := content[1].(map[string]any)
	if !ok {
		t.Fatalf("image part = %T, want map", content[1])
	}
	imageURL, ok := imagePart["image_url"].(map[string]any)
	if !ok || imageURL["url"] != "data:image/png;base64,dGVzdA==" {
		t.Fatalf("image_url = %v, want data URL", imagePart["image_url"])
	}
}

func TestBuildOpenAITextChatPayload_FixesToolChoiceAndIncludeUsage(t *testing.T) {
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "")

	cfg := config.DefaultConfig()
	cfg.ProviderModels["openrouter"] = config.ProviderModelConfig{
		DefaultModel:    "openai/gpt-4o",
		MaxOutputTokens: 456,
	}

	forcedToolName := "read_file"
	tests := []struct {
		name       string
		toolChoice *string
	}{
		{
			name:       "auto tool choice by default",
			toolChoice: nil,
		},
		{
			name:       "forced function tool choice",
			toolChoice: &forcedToolName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New("test-key")
			if tt.toolChoice != nil {
				p.SetToolChoice(*tt.toolChoice)
			}

			ctx, _ := newOpenRouterTestContext(t, cfg)
			payload, err := p.buildOpenAITextChatPayload(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, "openai/gpt-4o")
			if err != nil {
				t.Fatalf("buildOpenAITextChatPayload() error = %v", err)
			}

			var body map[string]any
			if err := json.Unmarshal(payload, &body); err != nil {
				t.Fatalf("json unmarshal failed: %v", err)
			}

			streamOptions, ok := body["stream_options"].(map[string]any)
			if !ok || streamOptions["include_usage"] != true {
				t.Fatalf("stream_options = %v, want include_usage=true", body["stream_options"])
			}

			if tt.toolChoice == nil {
				if body["tool_choice"] != "auto" {
					t.Fatalf("tool_choice = %v, want auto", body["tool_choice"])
				}
				return
			}

			toolChoiceBody, ok := body["tool_choice"].(map[string]any)
			if !ok {
				t.Fatalf("tool_choice = %T, want map", body["tool_choice"])
			}
			if toolChoiceBody["type"] != "function" {
				t.Fatalf("tool_choice.type = %v, want function", toolChoiceBody["type"])
			}
			functionBody, ok := toolChoiceBody["function"].(map[string]any)
			if !ok || functionBody["name"] != forcedToolName {
				t.Fatalf("tool_choice.function = %v, want read_file", toolChoiceBody["function"])
			}
		})
	}
}

func TestNewOpenRouterJSONRequest_SetsRequiredHeaders(t *testing.T) {
	p := New("test-key")
	payload := []byte(`{"model":"openai/gpt-4o"}`)

	req, err := p.newOpenRouterJSONRequest(context.Background(), "https://example.com/v1/chat/completions", payload)
	if err != nil {
		t.Fatalf("newOpenRouterJSONRequest() error = %v", err)
	}

	if req.Method != http.MethodPost {
		t.Fatalf("method = %q, want %q", req.Method, http.MethodPost)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("Authorization") != "Bearer test-key" {
		t.Fatalf("Authorization = %q, want Bearer test-key", req.Header.Get("Authorization"))
	}
	if req.Header.Get("HTTP-Referer") != "https://github.com/susugadx/xelyon-cli" {
		t.Fatalf("HTTP-Referer = %q, want https://github.com/susugadx/xelyon-cli", req.Header.Get("HTTP-Referer"))
	}
	if req.Header.Get("X-Title") != "XELYON CLI" {
		t.Fatalf("X-Title = %q, want XELYON CLI", req.Header.Get("X-Title"))
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if string(body) != string(payload) {
		t.Fatalf("body = %q, want %q", string(body), string(payload))
	}
}

func TestBuildClaudeChatPayload_FixesCompactionCacheAndImageContent(t *testing.T) {
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "0")

	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false
	cfg.PromptCache.Enabled = true
	cfg.ProviderModels["openrouter"] = config.ProviderModelConfig{
		DefaultModel:    "anthropic/claude-3.5-sonnet",
		MaxOutputTokens: 321,
	}

	p := New("test-key")
	ctx, _ := newOpenRouterTestContext(t, cfg)

	payload, err := p.buildClaudeChatPayload(
		ctx,
		"System prompt",
		[]api.Message{{Role: "assistant", Content: "previous"}},
		"Describe this",
		"anthropic/claude-3.5-sonnet",
		&api.ImageData{
			MediaType: "image/png",
			Base64:    "dGVzdA==",
		},
	)
	if err != nil {
		t.Fatalf("buildClaudeChatPayload() error = %v", err)
	}

	var body struct {
		Model             string            `json:"model"`
		AnthropicVersion  string            `json:"anthropic_version"`
		AnthropicBeta     []string          `json:"anthropic_beta,omitempty"`
		CacheControl      *api.CacheControl `json:"cache_control,omitempty"`
		Messages          []interface{}     `json:"messages"`
		Stream            bool              `json:"stream"`
		ContextManagement *struct {
			Edits []struct {
				Type string `json:"type"`
			} `json:"edits"`
		} `json:"context_management,omitempty"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if body.Model != "anthropic/claude-3.5-sonnet" {
		t.Fatalf("model = %q, want anthropic/claude-3.5-sonnet", body.Model)
	}
	if body.AnthropicVersion != "2023-06-01" {
		t.Fatalf("anthropic_version = %q, want 2023-06-01", body.AnthropicVersion)
	}
	if !body.Stream {
		t.Fatal("stream = false, want true")
	}
	if body.CacheControl == nil || body.CacheControl.Type != "ephemeral" {
		t.Fatalf("cache_control = %+v, want top-level ephemeral cache_control", body.CacheControl)
	}
	if body.ContextManagement == nil || len(body.ContextManagement.Edits) != 1 {
		t.Fatalf("context_management = %+v, want one clear_tool_uses edit", body.ContextManagement)
	}
	if body.ContextManagement.Edits[0].Type != "clear_tool_uses_20250919" {
		t.Fatalf("context_management.edits[0].type = %q, want clear_tool_uses_20250919", body.ContextManagement.Edits[0].Type)
	}
	if !containsString(body.AnthropicBeta, "context-management-2025-06-27") {
		t.Fatalf("anthropic_beta should include context-management-2025-06-27, got %v", body.AnthropicBeta)
	}
	if containsString(body.AnthropicBeta, "compact-2026-01-12") {
		t.Fatalf("anthropic_beta should not include compact-2026-01-12, got %v", body.AnthropicBeta)
	}

	if len(body.Messages) == 0 {
		t.Fatal("messages should not be empty")
	}
	lastMessage, ok := body.Messages[len(body.Messages)-1].(map[string]interface{})
	if !ok {
		t.Fatalf("last message should be map, got %T", body.Messages[len(body.Messages)-1])
	}
	content, ok := lastMessage["content"].([]interface{})
	if !ok || len(content) != 2 {
		t.Fatalf("content = %v, want image + text", lastMessage["content"])
	}

	firstPart, ok := content[0].(map[string]interface{})
	if !ok || firstPart["type"] != "image" {
		t.Fatalf("first content part = %v, want image part", content[0])
	}
	secondPart, ok := content[1].(map[string]interface{})
	if !ok || secondPart["type"] != "text" || secondPart["text"] != "Describe this" {
		t.Fatalf("second content part = %v, want text part", content[1])
	}
}

func TestBuildClaudeChatPayload_FunctionCallingDisabledOmitsTools(t *testing.T) {
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "0")

	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false

	p := New("test-key")
	ctx, _ := newOpenRouterTestContext(t, cfg)

	payload, err := p.buildClaudeChatPayload(
		ctx,
		"System prompt",
		[]api.Message{{Role: "user", Content: "hello"}},
		"",
		"anthropic/claude-3.5-sonnet",
		nil,
	)
	if err != nil {
		t.Fatalf("buildClaudeChatPayload() error = %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if _, ok := body["tools"]; ok {
		t.Fatalf("tools should be omitted when function calling is disabled, got %v", body["tools"])
	}
}
