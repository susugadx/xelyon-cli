package bedrock

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newBedrockStreamOutput(reader *fakeResponseStreamReader) *bedrockruntime.InvokeModelWithResponseStreamOutput {
	output := &bedrockruntime.InvokeModelWithResponseStreamOutput{}
	stream := bedrockruntime.NewInvokeModelWithResponseStreamEventStream(func(es *bedrockruntime.InvokeModelWithResponseStreamEventStream) {
		es.Reader = reader
	})
	setUnexported(output, "eventStream", stream)
	return output
}

func TestInvokeStream_SuccessAndMarshalFailure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		reader := &fakeResponseStreamReader{
			events: make(chan bedrocktypes.ResponseStream, 2),
		}
		reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
			Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`)},
		}
		reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
			Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"message_stop"}`)},
		}

		mockClient := &mockInvokeModelWithResponseStreamClient{
			output: newBedrockStreamOutput(reader),
		}
		p := &Provider{client: mockClient}

		ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard))
		ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

		got, err := p.invokeStream(ctx, "model-id", BedrockRequest{
			AnthropicVersion: "test-version",
			MaxTokens:        5,
		})
		if err != nil {
			t.Fatalf("invokeStream() error = %v", err)
		}
		if got != "Hello" {
			t.Fatalf("invokeStream() = %q, want %q", got, "Hello")
		}
		if mockClient.lastInput == nil {
			t.Fatal("InvokeModelWithResponseStream() should be called")
		}
		if aws.ToString(mockClient.lastInput.ModelId) != "model-id" {
			t.Fatalf("ModelId = %q, want %q", aws.ToString(mockClient.lastInput.ModelId), "model-id")
		}
		if got := aws.ToString(mockClient.lastInput.Accept); got != "application/json" {
			t.Fatalf("Accept = %q, want application/json", got)
		}
	})

	t.Run("marshalError", func(t *testing.T) {
		p := &Provider{client: &mockInvokeModelWithResponseStreamClient{}}
		_, err := p.invokeStream(context.Background(), "model-id", map[string]any{
			"bad": func() {},
		})
		if err == nil || !strings.Contains(err.Error(), "request marshal failed") {
			t.Fatalf("invokeStream() error = %v, want marshal failure", err)
		}
	})
}

func TestChatWithImage_WithoutImageFallsBackToTextRequest(t *testing.T) {
	mockClient := &mockInvokeModelWithResponseStreamClient{err: errors.New("boom")}
	p := &Provider{client: mockClient}

	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    defaultModel,
		MaxOutputTokens: 77,
	}
	ctx := newBedrockTestContext(cfg)

	_, err := p.ChatWithImage(ctx, "system prompt", nil, "plain message", nil, "")
	if err == nil || !strings.Contains(err.Error(), "bedrock API error") {
		t.Fatalf("ChatWithImage() error = %v, want wrapped bedrock API error", err)
	}
	if mockClient.lastInput == nil {
		t.Fatal("InvokeModelWithResponseStream() should be called")
	}
	if got := aws.ToString(mockClient.lastInput.ModelId); got != defaultModel {
		t.Fatalf("ModelId = %q, want %q", got, defaultModel)
	}

	var req BedrockRequest
	if err := json.Unmarshal(mockClient.lastInput.Body, &req); err != nil {
		t.Fatalf("json.Unmarshal(request) error = %v", err)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(req.Messages))
	}
	if req.Messages[0].Role != "user" {
		t.Fatalf("Messages[0].Role = %q, want user", req.Messages[0].Role)
	}
	if len(req.Messages[0].Content) != 1 || req.Messages[0].Content[0].Text != "plain message" {
		t.Fatalf("Messages[0].Content = %#v, want plain message block", req.Messages[0].Content)
	}
}

func TestSetMCPEnabled_NoOp(t *testing.T) {
	p := &Provider{}
	p.SetMCPEnabled(true)
	p.SetMCPEnabled(false)
}

func TestBuildBedrockContextManagement_LeavesHeadersWhenDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false

	contextManagement, headers := buildBedrockContextManagement("global.anthropic.claude-sonnet-4-6-v1", cfg.Compression, []string{"existing"})
	if contextManagement == nil {
		t.Fatal("ContextManagement should still be enabled for clear_tool_uses")
	}
	if !containsString(headers, "existing") {
		t.Fatalf("headers = %v, want existing header preserved", headers)
	}
}

func TestChatWithImage_BuildsToolAndThinkingForMultimodalRequest(t *testing.T) {
	t.Setenv("BEDROCK_FUNCTION_CALLING", "")

	mockClient := &mockInvokeModelWithResponseStreamClient{err: errors.New("boom")}
	p := &Provider{client: mockClient}
	p.SetMCPTools([]api.ToolDefinition{
		{Name: "lookup", Description: "Lookup", Parameters: map[string]any{"type": "object"}},
	})

	cfg := config.DefaultConfig()
	legacyThinkingModel := "global.anthropic.claude-opus-4-5-20251101-v1:0"
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    legacyThinkingModel,
		MaxOutputTokens: 88,
	}
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "high"

	ctx := newBedrockTestContext(cfg)
	_, err := p.ChatWithImage(ctx, "system prompt", []api.Message{{Role: "assistant", Content: "previous"}}, "describe", &api.ImageData{
		MediaType: "image/png",
		Base64:    "dGVzdA==",
	}, "")
	if err == nil || !strings.Contains(err.Error(), "bedrock API error") {
		t.Fatalf("ChatWithImage() error = %v, want wrapped bedrock API error", err)
	}

	var req BedrockMultimodalRequest
	if err := json.Unmarshal(mockClient.lastInput.Body, &req); err != nil {
		t.Fatalf("json.Unmarshal(request) error = %v", err)
	}
	if req.Thinking == nil || req.Thinking.BudgetTokens != api.LevelToBudgetTokens("high") {
		t.Fatalf("Thinking = %#v, want high-level budget", req.Thinking)
	}
	if req.OutputConfig != nil {
		t.Fatalf("OutputConfig = %#v, want nil for legacy thinking", req.OutputConfig)
	}
	if len(req.Tools) == 0 || !hasClaudeTool(req.Tools, "lookup") {
		t.Fatalf("Tools = %#v, want lookup tool", req.Tools)
	}
}

func TestChatWithImage_BuildsAdaptiveThinkingForOpus47(t *testing.T) {
	t.Setenv("BEDROCK_FUNCTION_CALLING", "0")

	mockClient := &mockInvokeModelWithResponseStreamClient{err: errors.New("boom")}
	p := &Provider{client: mockClient}

	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    defaultModel,
		MaxOutputTokens: 88,
	}
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"

	ctx := newBedrockTestContext(cfg)
	model := "global.anthropic.claude-opus-4-7-v1:0"
	_, err := p.ChatWithImage(ctx, "system prompt", []api.Message{{Role: "assistant", Content: "previous"}}, "describe", &api.ImageData{
		MediaType: "image/png",
		Base64:    "dGVzdA==",
	}, model)
	if err == nil || !strings.Contains(err.Error(), "bedrock API error") {
		t.Fatalf("ChatWithImage() error = %v, want wrapped bedrock API error", err)
	}

	var req BedrockMultimodalRequest
	if err := json.Unmarshal(mockClient.lastInput.Body, &req); err != nil {
		t.Fatalf("json.Unmarshal(request) error = %v", err)
	}
	if req.MaxTokens != 128000 {
		t.Fatalf("MaxTokens = %d, want 128000 catalog limit without thinking budget addition", req.MaxTokens)
	}
	if req.Thinking == nil || req.Thinking.Type != "adaptive" {
		t.Fatalf("Thinking = %#v, want adaptive", req.Thinking)
	}
	if req.OutputConfig == nil || req.OutputConfig.Effort != "xhigh" {
		t.Fatalf("OutputConfig = %#v, want effort=xhigh", req.OutputConfig)
	}
	if !containsString(req.AnthropicBeta, bedrockEffortBetaHeader) {
		t.Fatalf("AnthropicBeta = %v, want effort beta header", req.AnthropicBeta)
	}
	assertBedrockThinkingBudgetOmitted(t, mockClient.lastInput.Body)
}

func TestChatWithImage_UsesCatalogModelForOpus47Alias(t *testing.T) {
	t.Setenv("BEDROCK_FUNCTION_CALLING", "0")

	mockClient := &mockInvokeModelWithResponseStreamClient{err: errors.New("boom")}
	p := &Provider{client: mockClient}

	model := "corp-bedrock-opus47"
	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    model,
		CatalogModel:    "global.anthropic.claude-opus-4-7-v1:0",
		MaxOutputTokens: 64000,
	}
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"

	ctx := newBedrockTestContext(cfg)
	_, err := p.ChatWithImage(ctx, "system prompt", nil, "describe", &api.ImageData{
		MediaType: "image/png",
		Base64:    "dGVzdA==",
	}, model)
	if err == nil || !strings.Contains(err.Error(), "bedrock API error") {
		t.Fatalf("ChatWithImage() error = %v, want wrapped bedrock API error", err)
	}
	if got := aws.ToString(mockClient.lastInput.ModelId); got != model {
		t.Fatalf("ModelId = %q, want raw alias model %q", got, model)
	}

	var req BedrockMultimodalRequest
	if err := json.Unmarshal(mockClient.lastInput.Body, &req); err != nil {
		t.Fatalf("json.Unmarshal(request) error = %v", err)
	}
	if req.MaxTokens != 128000 {
		t.Fatalf("MaxTokens = %d, want 128000 catalog limit", req.MaxTokens)
	}
	if req.Thinking == nil || req.Thinking.Type != "adaptive" {
		t.Fatalf("Thinking = %#v, want adaptive via catalog_model", req.Thinking)
	}
	if req.OutputConfig == nil || req.OutputConfig.Effort != "xhigh" {
		t.Fatalf("OutputConfig = %#v, want effort=xhigh via catalog_model", req.OutputConfig)
	}
	if !containsString(req.AnthropicBeta, bedrockEffortBetaHeader) {
		t.Fatalf("AnthropicBeta = %v, want effort beta header", req.AnthropicBeta)
	}
	if containsString(req.AnthropicBeta, "compact-2026-01-12") {
		t.Fatalf("AnthropicBeta = %v, should not include Bedrock Opus 4.7 compaction beta", req.AnthropicBeta)
	}
	assertBedrockThinkingBudgetOmitted(t, mockClient.lastInput.Body)
}
