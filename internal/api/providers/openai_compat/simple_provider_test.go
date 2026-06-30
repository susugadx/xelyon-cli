package openaicompat

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestSimpleProviderBuildChatCompletionsRequestIncludesToolsChoiceAndActiveContext(t *testing.T) {
	var seenMCPTools []api.ToolDefinition
	provider := NewSimpleProvider("test-key", SimpleProviderSpec{
		ProviderKey:        "compat-test",
		DisplayName:        "Compat Test",
		DefaultURL:         "https://example.test/v1/chat/completions",
		FunctionCallingEnv: "XELYON_COMPAT_TEST_FUNCTION_CALLING",
		BuildTools: func(_ context.Context, mcpTools []api.ToolDefinition) []api.OpenAITool {
			seenMCPTools = append([]api.ToolDefinition(nil), mcpTools...)
			return []api.OpenAITool{{
				Type:     "function",
				Function: &api.ToolDefinition{Name: "mcp_probe"},
			}}
		},
	})
	t.Setenv("XELYON_COMPAT_TEST_FUNCTION_CALLING", "1")
	provider.SetMCPTools([]api.ToolDefinition{{Name: "mcp_probe"}})
	provider.SetToolChoice("mcp_probe")

	ctx := api.WithActiveContextBlocks(context.Background(), []api.ActiveContextBlock{{
		Name:    "state",
		Content: "<current_task_state>\nstate\n</current_task_state>",
	}})
	req := provider.BuildChatCompletionsRequest(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, "compat-model")
	body := decodeCompatRequest(t, req)

	if len(seenMCPTools) != 1 || seenMCPTools[0].Name != "mcp_probe" {
		t.Fatalf("BuildTools mcp tools = %#v, want configured MCP tool", seenMCPTools)
	}
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 3 {
		t.Fatalf("messages = %#v, want system + active context + history", body["messages"])
	}
	active, ok := messages[1].(map[string]any)
	if !ok || active["role"] != "system" || active["content"] != "<current_task_state>\nstate\n</current_task_state>" {
		t.Fatalf("active context message = %#v, want ephemeral system message", messages[1])
	}
	tools, ok := body["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want one function tool", body["tools"])
	}
	toolChoice, ok := body["tool_choice"].(map[string]any)
	if !ok || toolChoice["type"] != "function" {
		t.Fatalf("tool_choice = %#v, want forced function", body["tool_choice"])
	}
	function, ok := toolChoice["function"].(map[string]any)
	if !ok || function["name"] != "mcp_probe" {
		t.Fatalf("tool_choice.function = %#v, want mcp_probe", toolChoice["function"])
	}
}

func TestSimpleProviderBuildChatCompletionsRequestOmitsToolsWhenFunctionCallingDisabled(t *testing.T) {
	buildToolsCalled := false
	provider := NewSimpleProvider("test-key", SimpleProviderSpec{
		ProviderKey:        "compat-test",
		DisplayName:        "Compat Test",
		DefaultURL:         "https://example.test/v1/chat/completions",
		FunctionCallingEnv: "XELYON_COMPAT_TEST_FUNCTION_CALLING",
		BuildTools: func(context.Context, []api.ToolDefinition) []api.OpenAITool {
			buildToolsCalled = true
			return []api.OpenAITool{{
				Type:     "function",
				Function: &api.ToolDefinition{Name: "mcp_probe"},
			}}
		},
	})
	t.Setenv("XELYON_COMPAT_TEST_FUNCTION_CALLING", "0")
	provider.SetToolChoice("mcp_probe")

	req := provider.BuildChatCompletionsRequest(context.Background(), "system prompt", nil, "compat-model")
	body := decodeCompatRequest(t, req)

	if buildToolsCalled {
		t.Fatal("BuildTools called when function calling env disabled")
	}
	if _, ok := body["tools"]; ok {
		t.Fatalf("tools = %#v, want omitted", body["tools"])
	}
	if _, ok := body["tool_choice"]; ok {
		t.Fatalf("tool_choice = %#v, want omitted", body["tool_choice"])
	}
}

func TestSimpleProviderBuildChatCompletionsRequestOmitsToolsWhenRequestDisablesToolUse(t *testing.T) {
	buildToolsCalled := false
	provider := NewSimpleProvider("test-key", SimpleProviderSpec{
		ProviderKey: "compat-test",
		DisplayName: "Compat Test",
		DefaultURL:  "https://example.test/v1/chat/completions",
		BuildTools: func(context.Context, []api.ToolDefinition) []api.OpenAITool {
			buildToolsCalled = true
			return []api.OpenAITool{{
				Type:     "function",
				Function: &api.ToolDefinition{Name: "mcp_probe"},
			}}
		},
	})
	provider.SetToolChoice("mcp_probe")

	req := provider.BuildChatCompletionsRequest(api.WithToolUseDisabled(context.Background()), "system prompt", nil, "compat-model")
	body := decodeCompatRequest(t, req)

	if buildToolsCalled {
		t.Fatal("BuildTools called when request disabled tool use")
	}
	if _, ok := body["tools"]; ok {
		t.Fatalf("tools = %#v, want omitted", body["tools"])
	}
	if _, ok := body["tool_choice"]; ok {
		t.Fatalf("tool_choice = %#v, want omitted", body["tool_choice"])
	}
}

func TestSimpleProviderBuildChatCompletionsRequestKeepsImageHistoryTextOnlyWhenUnsupported(t *testing.T) {
	provider := NewSimpleProvider("test-key", SimpleProviderSpec{
		ProviderKey:    "compat-test",
		DisplayName:    "Compat Test",
		DefaultURL:     "https://example.test/v1/chat/completions",
		SupportsImages: false,
	})

	req := provider.BuildChatCompletionsRequest(context.Background(), "system prompt", []api.Message{
		api.NewUserImageMessage("inspect", &api.ImageData{MediaType: "image/png", Base64: "aW1hZ2U="}),
	}, "compat-model")
	body := decodeCompatRequest(t, req)
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v, want system + image history", body["messages"])
	}
	imageMessage, ok := messages[1].(map[string]any)
	if !ok {
		t.Fatalf("image message = %#v, want object", messages[1])
	}
	if imageMessage["content"] != "inspect" {
		t.Fatalf("image history content = %#v, want text-only content", imageMessage["content"])
	}
}

func TestSimpleProviderBuildChatCompletionsRequestSendsImageHistoryWhenSupported(t *testing.T) {
	provider := NewSimpleProvider("test-key", SimpleProviderSpec{
		ProviderKey:    "compat-test",
		DisplayName:    "Compat Test",
		DefaultURL:     "https://example.test/v1/chat/completions",
		SupportsImages: true,
	})

	req := provider.BuildChatCompletionsRequest(context.Background(), "system prompt", []api.Message{
		api.NewUserImageMessage("inspect", &api.ImageData{MediaType: "image/png", Base64: "aW1hZ2U="}),
	}, "compat-model")
	body := decodeCompatRequest(t, req)
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v, want system + image history", body["messages"])
	}
	imageMessage, ok := messages[1].(map[string]any)
	if !ok {
		t.Fatalf("image message = %#v, want object", messages[1])
	}
	parts, ok := imageMessage["content"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("image content = %#v, want text + image_url parts", imageMessage["content"])
	}
	imagePart, ok := parts[1].(map[string]any)
	if !ok {
		t.Fatalf("image part = %#v, want object", parts[1])
	}
	imageURL, ok := imagePart["image_url"].(map[string]any)
	if !ok || imageURL["url"] != "data:image/png;base64,aW1hZ2U=" {
		t.Fatalf("image_url = %#v, want data URL", imagePart["image_url"])
	}
}

func decodeCompatRequest(t *testing.T, req ChatCompletionsRequest) map[string]any {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return body
}
