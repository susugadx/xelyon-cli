package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProvider_BuildChatRequest_ModelAndTokenPolicy(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("ollama", config.ProviderModelConfig{
		DefaultModel:    "configured-llama",
		MaxOutputTokens: 4321,
		ModelOverrides: map[string]config.ModelOverride{
			"explicit-llama": {MaxOutputTokens: 99},
		},
	})

	ctx := api.WithToolUseDisabled(config.WithContext(context.Background(), cfg))
	p := New("http://ollama.test")

	defaultBuild := p.buildChatRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, "")
	if defaultBuild.Model != "configured-llama" {
		t.Fatalf("Model = %q, want configured default", defaultBuild.Model)
	}
	if defaultBuild.Request.Model != "configured-llama" {
		t.Fatalf("Request.Model = %q, want configured default", defaultBuild.Request.Model)
	}
	if defaultBuild.Request.Options == nil || defaultBuild.Request.Options.NumPredict != 4321 {
		t.Fatalf("Options.NumPredict = %#v, want 4321", defaultBuild.Request.Options)
	}

	explicitBuild := p.buildChatRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, "explicit-llama")
	if explicitBuild.Model != "explicit-llama" {
		t.Fatalf("explicit Model = %q, want explicit model", explicitBuild.Model)
	}
	if explicitBuild.Request.Options == nil || explicitBuild.Request.Options.NumPredict != 99 {
		t.Fatalf("explicit Options.NumPredict = %#v, want override value 99", explicitBuild.Request.Options)
	}
}

func TestProvider_ChatWithTools_UsesBuiltChatRequest(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("ollama", config.ProviderModelConfig{
		DefaultModel:    "configured-llama",
		MaxOutputTokens: 4321,
	})

	var received OllamaRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("URL path = %q, want /api/chat", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		ollamaStreamingHandler([]string{"ok"})(w, r)
	})

	ctx := api.WithToolUseDisabled(config.WithContext(context.Background(), cfg))
	p := New(server.URL)
	got, err := p.ChatWithTools(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("ChatWithTools() = %q, want ok", got)
	}
	if received.Model != "configured-llama" {
		t.Fatalf("request model = %q, want configured default", received.Model)
	}
	if received.Options == nil || received.Options.NumPredict != 4321 {
		t.Fatalf("request Options.NumPredict = %#v, want 4321", received.Options)
	}
}

func TestProvider_BuildChatRequest_FallbackModelAndPayloadShape(t *testing.T) {
	ctx := api.WithToolUseDisabled(config.WithContext(context.Background(), &config.Config{}))
	p := New("http://ollama.test")

	build := p.buildChatRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, "")
	if build.Model != "llama3" {
		t.Fatalf("Model = %q, want built-in fallback llama3", build.Model)
	}
	if build.URL != "http://ollama.test/api/chat" {
		t.Fatalf("URL = %q, want chat endpoint", build.URL)
	}

	req := build.Request
	if !req.Stream {
		t.Fatal("Stream = false, want true")
	}
	if len(req.Messages) != 2 {
		t.Fatalf("Messages length = %d, want 2", len(req.Messages))
	}
	if req.Messages[0].Role != "system" || req.Messages[0].Content != "System" {
		t.Fatalf("Messages[0] = %#v, want system prompt", req.Messages[0])
	}
	if req.Messages[1].Role != "user" || req.Messages[1].Content != "Hi" {
		t.Fatalf("Messages[1] = %#v, want history message", req.Messages[1])
	}
	if req.Options == nil {
		t.Fatal("Options = nil, want options object")
	}
}

func TestProvider_BuildChatRequest_FunctionCallingFields(t *testing.T) {
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	ctx := api.WithToolDefinitions(context.Background(), []api.ToolDefinition{testOllamaToolDefinition("probe_tool")})
	p := New("http://ollama.test")

	build := p.buildChatRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, "llama3")
	req := build.Request
	if len(req.Tools) != 1 {
		t.Fatalf("Tools length = %d, want 1", len(req.Tools))
	}
	if req.Tools[0].Function == nil || req.Tools[0].Function.Name != "probe_tool" {
		t.Fatalf("Tools[0] = %#v, want probe_tool", req.Tools[0])
	}
	if req.ToolChoice != "auto" {
		t.Fatalf("ToolChoice = %q, want auto", req.ToolChoice)
	}
}

func TestProvider_BuildChatRequest_ForcedToolChoice(t *testing.T) {
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	ctx := api.WithToolDefinitions(context.Background(), []api.ToolDefinition{testOllamaToolDefinition("read_file")})
	p := New("http://ollama.test")
	p.SetToolChoice("read_file")

	build := p.buildChatRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, "llama3")
	if build.Request.ToolChoice != "read_file" {
		t.Fatalf("ToolChoice = %q, want forced tool choice", build.Request.ToolChoice)
	}
}

func TestProvider_BuildChatRequest_OmitsToolFieldsWhenFunctionCallingDisabled(t *testing.T) {
	t.Setenv("OLLAMA_FUNCTION_CALLING", "0")

	ctx := api.WithToolDefinitions(context.Background(), []api.ToolDefinition{testOllamaToolDefinition("probe_tool")})
	p := New("http://ollama.test")
	p.SetToolChoice("probe_tool")

	build := p.buildChatRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, "llama3")
	assertOllamaToolFieldsOmitted(t, build.Request)
}

func TestProvider_BuildChatRequest_OmitsToolFieldsWhenToolUseDisabled(t *testing.T) {
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")

	ctx := api.WithToolDefinitions(context.Background(), []api.ToolDefinition{testOllamaToolDefinition("probe_tool")})
	ctx = api.WithToolUseDisabled(ctx)
	p := New("http://ollama.test")
	p.SetToolChoice("probe_tool")

	build := p.buildChatRequest(ctx, "System", []api.Message{{Role: "user", Content: "Hi"}}, "llama3")
	assertOllamaToolFieldsOmitted(t, build.Request)
}

func TestProvider_ChatEndpointURL_PreservesExistingConcatenation(t *testing.T) {
	p := New("http://ollama.test/root/")
	if got := p.chatEndpointURL(); got != "http://ollama.test/root//api/chat" {
		t.Fatalf("chatEndpointURL() = %q, want existing simple concatenation", got)
	}
}

func testOllamaToolDefinition(name string) api.ToolDefinition {
	return api.ToolDefinition{
		Name:        name,
		Description: "test tool",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

func assertOllamaToolFieldsOmitted(t *testing.T, req OllamaRequest) {
	t.Helper()

	if len(req.Tools) != 0 {
		t.Fatalf("Tools length = %d, want 0", len(req.Tools))
	}
	if req.ToolChoice != "" {
		t.Fatalf("ToolChoice = %q, want empty", req.ToolChoice)
	}

	var encoded map[string]any
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if err := json.Unmarshal(data, &encoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if _, ok := encoded["tools"]; ok {
		t.Fatalf("encoded request contains tools: %#v", encoded["tools"])
	}
	if _, ok := encoded["tool_choice"]; ok {
		t.Fatalf("encoded request contains tool_choice: %#v", encoded["tool_choice"])
	}
}
