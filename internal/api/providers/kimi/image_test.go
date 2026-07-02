package kimi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
)

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

func TestBuildChatCompletionsRequest_SerializesImageHistoryAtOriginalPosition(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	p := New("test-key")
	ctx, _, _ := newKimiTestContext(t, false)
	built := p.buildChatCompletionsRequest(ctx, "System", []api.Message{
		api.NewUserImageMessage("inspect", &api.ImageData{MediaType: "image/png", Base64: kimiTestPNGBase64}),
		{Role: "assistant", ToolCalls: []api.OpenAIToolCall{{
			ID:       "call_1",
			Type:     "function",
			Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"README.md"}`},
		}}},
		{Role: "tool", ToolCallID: "call_1", Content: "README contents"},
	}, "kimi-k2.6")

	payload, err := json.Marshal(built.Request)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 4 {
		t.Fatalf("messages = %#v, want system + image + assistant tool call + tool result", body["messages"])
	}
	imageMessage, ok := messages[1].(map[string]any)
	if !ok {
		t.Fatalf("image message = %#v, want object", messages[1])
	}
	content, ok := imageMessage["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("image content = %#v, want text + image_url", imageMessage["content"])
	}
	imagePart, ok := content[1].(map[string]any)
	if !ok {
		t.Fatalf("image part = %#v, want object", content[1])
	}
	imageURL, ok := imagePart["image_url"].(map[string]any)
	if !ok || imageURL["url"] != "data:image/png;base64,"+kimiTestPNGBase64 {
		t.Fatalf("image_url = %#v, want data URL", imagePart["image_url"])
	}
	if messages[2].(map[string]any)["tool_calls"] == nil {
		t.Fatalf("assistant message = %#v, want tool_calls preserved", messages[2])
	}
	if messages[3].(map[string]any)["tool_call_id"] != "call_1" {
		t.Fatalf("tool result message = %#v, want call_1", messages[3])
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

	p := newKimiReadFileToolProviderForTest()
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

func TestChatWithImage_RequestToolUseDisabledOmitsTools(t *testing.T) {
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
	if _, err := p.ChatWithImage(ctx, "System", nil, "describe image", &api.ImageData{
		Base64:    kimiTestPNGBase64,
		MediaType: "image/png",
	}, "kimi-k2.6"); err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}
	assertKimiToolPayloadOmitted(t, captured)
}

func TestChatWithImage_StripsToolNameFromHistoryToolResults(t *testing.T) {
	t.Setenv("KIMI_FUNCTION_CALLING", "0")
	var captured map[string]any
	server := mockKimiAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		kimiStreamingHandler([]string{`{"choices":[{"delta":{"content":"ok"}}]}`})(w, r)
	})
	t.Setenv("KIMI_API_URL", server.URL)

	history := []api.Message{
		{
			Role: "assistant",
			ToolCalls: []api.OpenAIToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"README.md"}`,
				},
			}},
		},
		{
			Role:       "tool",
			Content:    "README contents",
			ToolCallID: "call_1",
			ToolName:   "read_file",
		},
	}

	ctx, _, _ := newKimiTestContext(t, false)
	if _, err := New("test-key").ChatWithImage(ctx, "System", history, "describe image", &api.ImageData{
		Base64:    kimiTestPNGBase64,
		MediaType: "image/png",
	}, "kimi-k2.6"); err != nil {
		t.Fatalf("ChatWithImage() error = %v", err)
	}

	messages, ok := captured["messages"].([]any)
	if !ok || len(messages) != 4 {
		t.Fatalf("messages = %#v, want system + assistant + tool + multimodal user", captured["messages"])
	}
	toolResult, ok := messages[2].(map[string]any)
	if !ok || toolResult["role"] != "tool" || toolResult["tool_call_id"] != "call_1" || toolResult["content"] != "README contents" {
		t.Fatalf("tool result message = %#v, want role/tool_call_id/content preserved", messages[2])
	}
	if _, ok := toolResult["tool_name"]; ok {
		t.Fatalf("tool result message = %#v, want no tool_name in Kimi image payload", toolResult)
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
