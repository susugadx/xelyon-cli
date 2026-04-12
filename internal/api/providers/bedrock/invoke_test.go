package bedrock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type mockInvokeModelWithResponseStreamClient struct {
	lastInput *bedrockruntime.InvokeModelWithResponseStreamInput
	output    *bedrockruntime.InvokeModelWithResponseStreamOutput
	err       error
}

func (m *mockInvokeModelWithResponseStreamClient) InvokeModelWithResponseStream(_ context.Context, input *bedrockruntime.InvokeModelWithResponseStreamInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error) {
	if input != nil {
		cloned := *input
		cloned.Body = append([]byte(nil), input.Body...)
		m.lastInput = &cloned
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.output, nil
}

func TestNew_UsesRegionEnvironment(t *testing.T) {
	t.Setenv("AWS_REGION", "ap-northeast-1")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	p, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if p.region != "ap-northeast-1" {
		t.Fatalf("region = %q, want ap-northeast-1", p.region)
	}
	if p.client == nil {
		t.Fatal("client should be initialized")
	}
}

func TestProvider_ChatWithTools_BuildsRequestFromContext(t *testing.T) {
	mockClient := &mockInvokeModelWithResponseStreamClient{err: errors.New("boom")}
	p := &Provider{client: mockClient}
	p.SetMCPTools([]api.ToolDefinition{
		{
			Name:        "custom_lookup",
			Description: "Lookup custom data",
			Parameters: map[string]any{
				"type": "object",
			},
		},
	})

	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:     "global.anthropic.claude-sonnet-4-6-v1",
		MaxOutputTokens:  321,
		AnthropicVersion: "bedrock-test-version",
		AnthropicBeta:    []string{"beta-from-config"},
	}
	cfg.PromptCache.Enabled = true
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "high"
	cfg.Compression.ClaudeCompaction = true

	ctx := newBedrockTestContext(cfg)
	_, err := p.ChatWithTools(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err == nil || !strings.Contains(err.Error(), "bedrock API error") {
		t.Fatalf("ChatWithTools() error = %v, want wrapped bedrock API error", err)
	}
	if mockClient.lastInput == nil {
		t.Fatal("InvokeModelWithResponseStream() should be called")
	}
	if got := aws.ToString(mockClient.lastInput.ModelId); got != "global.anthropic.claude-sonnet-4-6-v1" {
		t.Fatalf("ModelId = %q, want %q", got, "global.anthropic.claude-sonnet-4-6-v1")
	}
	if got := aws.ToString(mockClient.lastInput.ContentType); got != "application/json" {
		t.Fatalf("ContentType = %q, want application/json", got)
	}

	var req BedrockRequest
	if err := json.Unmarshal(mockClient.lastInput.Body, &req); err != nil {
		t.Fatalf("json.Unmarshal(request) error = %v", err)
	}
	if req.AnthropicVersion != "bedrock-test-version" {
		t.Fatalf("AnthropicVersion = %q, want %q", req.AnthropicVersion, "bedrock-test-version")
	}
	if req.MaxTokens != 321 {
		t.Fatalf("MaxTokens = %d, want 321", req.MaxTokens)
	}
	if req.CacheControl == nil || req.CacheControl.Type != "ephemeral" {
		t.Fatalf("CacheControl = %#v, want ephemeral cache control", req.CacheControl)
	}
	if req.Thinking == nil || req.Thinking.BudgetTokens != api.LevelToBudgetTokens("high") {
		t.Fatalf("Thinking = %#v, want high-level budget", req.Thinking)
	}
	if req.ContextManagement == nil {
		t.Fatal("ContextManagement should be set for supported Claude model")
	}
	if !containsString(req.AnthropicBeta, "beta-from-config") {
		t.Fatalf("AnthropicBeta = %v, want beta-from-config", req.AnthropicBeta)
	}
	if !containsString(req.AnthropicBeta, "context-management-2025-06-27") {
		t.Fatalf("AnthropicBeta = %v, want context-management beta", req.AnthropicBeta)
	}
	if !containsString(req.AnthropicBeta, "compact-2026-01-12") {
		t.Fatalf("AnthropicBeta = %v, want compaction beta", req.AnthropicBeta)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(req.Messages))
	}
	if !hasClaudeTool(req.Tools, "custom_lookup") {
		t.Fatalf("Tools = %#v, want to contain custom_lookup", req.Tools)
	}
	if req.System == nil {
		t.Fatal("System should be populated")
	}
}

func TestProvider_ChatWithImage_BuildsMultimodalRequestAndVersionFallback(t *testing.T) {
	t.Setenv("BEDROCK_FUNCTION_CALLING", "0")

	mockClient := &mockInvokeModelWithResponseStreamClient{err: errors.New("boom")}
	p := &Provider{client: mockClient}

	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:     "global.anthropic.claude-sonnet-4-6-v1",
		MaxOutputTokens:  456,
		AnthropicVersion: "",
	}
	cfg.PromptCache.Enabled = false
	cfg.Thinking.Enabled = false

	ctx := newBedrockTestContext(cfg)
	image := &api.ImageData{
		MediaType: "image/png",
		Base64:    "aGVsbG8=",
	}

	_, err := p.ChatWithImage(ctx, "system prompt", []api.Message{{Role: "assistant", Content: "previous"}}, "describe this image", image, "global.anthropic.claude-sonnet-4-6-v1")
	if err == nil || !strings.Contains(err.Error(), "bedrock API error") {
		t.Fatalf("ChatWithImage() error = %v, want wrapped bedrock API error", err)
	}
	if mockClient.lastInput == nil {
		t.Fatal("InvokeModelWithResponseStream() should be called")
	}

	var raw map[string]any
	if err := json.Unmarshal(mockClient.lastInput.Body, &raw); err != nil {
		t.Fatalf("json.Unmarshal(request) error = %v", err)
	}
	if raw["anthropic_version"] != bedrockAnthropicVersion {
		t.Fatalf("anthropic_version = %v, want %q fallback", raw["anthropic_version"], bedrockAnthropicVersion)
	}
	if raw["cache_control"] != nil {
		t.Fatalf("cache_control = %v, want omitted", raw["cache_control"])
	}
	if raw["thinking"] != nil {
		t.Fatalf("thinking = %v, want omitted", raw["thinking"])
	}
	if raw["tools"] != nil {
		t.Fatalf("tools = %v, want omitted when function calling disabled", raw["tools"])
	}

	messages, ok := raw["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %v, want two messages", raw["messages"])
	}
	lastMessage, ok := messages[len(messages)-1].(map[string]any)
	if !ok {
		t.Fatalf("last message = %T, want map", messages[len(messages)-1])
	}
	if lastMessage["role"] != "user" {
		t.Fatalf("last message role = %v, want user", lastMessage["role"])
	}

	content, ok := lastMessage["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content = %v, want image + text", lastMessage["content"])
	}
	imagePart, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("image part = %T, want map", content[0])
	}
	if imagePart["type"] != "image" {
		t.Fatalf("image part type = %v, want image", imagePart["type"])
	}
	source, ok := imagePart["source"].(map[string]any)
	if !ok {
		t.Fatalf("image source = %T, want map", imagePart["source"])
	}
	if source["media_type"] != "image/png" {
		t.Fatalf("media_type = %v, want image/png", source["media_type"])
	}
	if source["data"] != "aGVsbG8=" {
		t.Fatalf("data = %v, want base64 payload", source["data"])
	}
	textPart, ok := content[1].(map[string]any)
	if !ok || textPart["text"] != "describe this image" {
		t.Fatalf("text part = %v, want describe text", content[1])
	}
}

func newBedrockTestContext(cfg *config.Config) context.Context {
	var out bytes.Buffer
	runtime := ui.NewRuntime(strings.NewReader(""), &out, &out)
	ctx := ui.WithRuntime(context.Background(), runtime)
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	return config.WithContext(ctx, cfg)
}

func hasClaudeTool(tools []claude.ClaudeTool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
