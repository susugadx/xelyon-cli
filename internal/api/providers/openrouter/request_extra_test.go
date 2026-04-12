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
