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
	"github.com/susugadx/xelyon-cli/internal/ledger"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type mockConverseStreamClient struct {
	lastInput *bedrockruntime.ConverseStreamInput
	output    *bedrockruntime.ConverseStreamOutput
	outputs   []*bedrockruntime.ConverseStreamOutput
	err       error
}

func (m *mockConverseStreamClient) ConverseStream(_ context.Context, input *bedrockruntime.ConverseStreamInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error) {
	m.lastInput = input
	if m.err != nil {
		return nil, m.err
	}
	if len(m.outputs) > 0 {
		output := m.outputs[0]
		m.outputs = m.outputs[1:]
		return output, nil
	}
	return m.output, nil
}

type fakeConverseStreamReader struct {
	events chan bedrocktypes.ConverseStreamOutput
	err    error
	closed bool
}

func (f *fakeConverseStreamReader) Events() <-chan bedrocktypes.ConverseStreamOutput {
	return f.events
}

func (f *fakeConverseStreamReader) Close() error {
	f.closed = true
	return nil
}

func (f *fakeConverseStreamReader) Err() error {
	return f.err
}

func newConverseStreamOutput(reader *fakeConverseStreamReader) *bedrockruntime.ConverseStreamOutput {
	output := &bedrockruntime.ConverseStreamOutput{}
	stream := bedrockruntime.NewConverseStreamEventStream(func(es *bedrockruntime.ConverseStreamEventStream) {
		es.Reader = reader
	})
	setUnexported(output, "eventStream", stream)
	return output
}

func newClosedConverseStreamOutput(events ...bedrocktypes.ConverseStreamOutput) (*bedrockruntime.ConverseStreamOutput, *fakeConverseStreamReader) {
	reader := &fakeConverseStreamReader{
		events: make(chan bedrocktypes.ConverseStreamOutput, len(events)),
	}
	for _, event := range events {
		reader.events <- event
	}
	close(reader.events)
	return newConverseStreamOutput(reader), reader
}

func TestProvider_ChatWithTools_UsesConverseStreamForNonClaudeBedrockModel(t *testing.T) {
	t.Setenv("BEDROCK_FUNCTION_CALLING", "0")

	output, _ := newClosedConverseStreamOutput(
		&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
			Value: bedrocktypes.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta:             &bedrocktypes.ContentBlockDeltaMemberText{Value: "Hello"},
			},
		},
		&bedrocktypes.ConverseStreamOutputMemberMetadata{
			Value: bedrocktypes.ConverseStreamMetadataEvent{
				Usage: &bedrocktypes.TokenUsage{
					InputTokens:  aws.Int32(11),
					OutputTokens: aws.Int32(7),
				},
			},
		},
	)
	mockInvoke := &mockInvokeModelWithResponseStreamClient{err: errors.New("should not call invoke")}
	mockConverse := &mockConverseStreamClient{output: output}
	p := &Provider{client: mockInvoke, converseClient: mockConverse}

	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    "amazon.nova-pro-v1:0",
		MaxOutputTokens: 64000,
	}
	cfg.PromptCache.Enabled = true
	cfg.Thinking.Enabled = true

	var usage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		usage = u
	})

	ctx := newBedrockTestContext(cfg)
	got, err := p.ChatWithTools(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if got != "Hello" {
		t.Fatalf("ChatWithTools() = %q, want Hello", got)
	}
	if mockInvoke.lastInput != nil {
		t.Fatal("InvokeModelWithResponseStream() should not be called for Converse route")
	}
	if mockConverse.lastInput == nil {
		t.Fatal("ConverseStream() should be called")
	}
	if got := aws.ToString(mockConverse.lastInput.ModelId); got != "amazon.nova-pro-v1:0" {
		t.Fatalf("ModelId = %q, want nova model", got)
	}
	if got := aws.ToInt32(mockConverse.lastInput.InferenceConfig.MaxTokens); got != 5000 {
		t.Fatalf("MaxTokens = %d, want 5000 from catalog", got)
	}
	if mockConverse.lastInput.ToolConfig != nil {
		t.Fatalf("ToolConfig = %#v, want nil when function calling disabled", mockConverse.lastInput.ToolConfig)
	}
	if len(mockConverse.lastInput.System) != 1 {
		t.Fatalf("len(System) = %d, want 1", len(mockConverse.lastInput.System))
	}
	system, ok := mockConverse.lastInput.System[0].(*bedrocktypes.SystemContentBlockMemberText)
	if !ok || system.Value != "system prompt" {
		t.Fatalf("System[0] = %#v, want text system prompt", mockConverse.lastInput.System[0])
	}
	if len(mockConverse.lastInput.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(mockConverse.lastInput.Messages))
	}
	message := mockConverse.lastInput.Messages[0]
	if message.Role != bedrocktypes.ConversationRoleUser {
		t.Fatalf("message.Role = %q, want user", message.Role)
	}
	text, ok := message.Content[0].(*bedrocktypes.ContentBlockMemberText)
	if !ok || text.Value != "hello" {
		t.Fatalf("message.Content[0] = %#v, want user text", message.Content[0])
	}
	if usage.InputTokens != 11 || usage.OutputTokens != 7 {
		t.Fatalf("usage = %#v, want input=11 output=7", usage)
	}
	if p.LastAnthropicContentBlocks() != nil {
		t.Fatal("Converse route should not populate Claude content block replay state")
	}
}

func TestBuildConverseStreamInput_ToolUseDisabledOmitsToolConfig(t *testing.T) {
	t.Setenv("BEDROCK_FUNCTION_CALLING", "1")

	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel: "amazon.nova-pro-v1:0",
	}
	p := &Provider{}
	p.SetMCPTools([]api.ToolDefinition{{Name: "custom_lookup", Description: "custom lookup"}})
	ctx := api.WithToolUseDisabled(newBedrockTestContext(cfg))

	input, err := p.buildConverseStreamInput(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, p.resolveBedrockRequestContext(ctx, ""))
	if err != nil {
		t.Fatalf("buildConverseStreamInput() error = %v", err)
	}
	if input.ToolConfig != nil {
		t.Fatalf("ToolConfig = %#v, want nil when tool use is disabled", input.ToolConfig)
	}
}

func TestBuildConverseStreamInput_AddsActiveContextAsSeparateSystemBlock(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel: "amazon.nova-pro-v1:0",
	}
	p := &Provider{}
	evidence := bedrockTestRehydratedEvidence()
	ctx := api.WithActiveContextBlocks(newBedrockTestContext(cfg), []api.ActiveContextBlock{{
		Name:    "provider_history_rehydrated_evidence",
		Content: evidence,
	}})

	input, err := p.buildConverseStreamInput(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, p.resolveBedrockRequestContext(ctx, ""))
	if err != nil {
		t.Fatalf("buildConverseStreamInput() error = %v", err)
	}
	if len(input.System) != 2 {
		t.Fatalf("len(System) = %d, want system prompt plus active context block", len(input.System))
	}
	base, ok := input.System[0].(*bedrocktypes.SystemContentBlockMemberText)
	if !ok || base.Value != "system prompt" {
		t.Fatalf("System[0] = %#v, want base system prompt", input.System[0])
	}
	active, ok := input.System[1].(*bedrocktypes.SystemContentBlockMemberText)
	if !ok || active.Value != evidence {
		t.Fatalf("System[1] = %#v, want active context system block", input.System[1])
	}
}

func bedrockTestRehydratedEvidence() string {
	return ledger.RenderRehydratedEvidenceBlock(ledger.RehydratedEvidenceBlock{Items: []ledger.RehydratedEvidenceItem{{
		Path:       "README.md",
		StartLine:  1,
		EndLine:    2,
		Source:     "read_file",
		Reason:     ledger.RehydratePlanReasonOmittedProviderHistory,
		ToolCallID: "call_read",
		Content:    "line one\nline two",
	}}})
}

func TestProvider_ChatWithTools_RejectsUnsupportedConverseModelBeforeAPI(t *testing.T) {
	mockConverse := &mockConverseStreamClient{err: errors.New("should not call converse")}
	p := &Provider{converseClient: mockConverse}

	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    "us.meta.llama4-scout-17b-instruct-v1:0",
		MaxOutputTokens: 64000,
	}
	ctx := newBedrockTestContext(cfg)

	_, err := p.ChatWithTools(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err == nil || !strings.Contains(err.Error(), "requires a model with streaming tool use support") {
		t.Fatalf("ChatWithTools() error = %v, want unsupported Converse tool-use error", err)
	}
	if mockConverse.lastInput != nil {
		t.Fatal("ConverseStream() should not be called for unsupported Converse model")
	}
}

func TestProvider_ChatWithTools_AllowsUnsupportedConverseModelWhenToolPayloadDisabled(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func(context.Context) context.Context
		envValue string
	}{
		{
			name:     "request disables tool use",
			setupCtx: api.WithToolUseDisabled,
		},
		{
			name:     "env disables function calling",
			setupCtx: func(ctx context.Context) context.Context { return ctx },
			envValue: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("BEDROCK_FUNCTION_CALLING", tt.envValue)
			}
			output, _ := newClosedConverseStreamOutput(
				&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
					Value: bedrocktypes.ContentBlockDeltaEvent{
						ContentBlockIndex: aws.Int32(0),
						Delta:             &bedrocktypes.ContentBlockDeltaMemberText{Value: "Text only"},
					},
				},
			)
			mockConverse := &mockConverseStreamClient{output: output}
			p := &Provider{converseClient: mockConverse}

			cfg := config.DefaultConfig()
			cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
				DefaultModel:    "us.meta.llama4-scout-17b-instruct-v1:0",
				MaxOutputTokens: 64000,
			}
			ctx := tt.setupCtx(newBedrockTestContext(cfg))

			got, err := p.ChatWithTools(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, "")
			if err != nil {
				t.Fatalf("ChatWithTools() error = %v", err)
			}
			if got != "Text only" {
				t.Fatalf("ChatWithTools() = %q, want Text only", got)
			}
			if mockConverse.lastInput == nil {
				t.Fatal("ConverseStream() should be called when tool payload is disabled")
			}
			if mockConverse.lastInput.ToolConfig != nil {
				t.Fatalf("ToolConfig = %#v, want nil when tool payload is disabled", mockConverse.lastInput.ToolConfig)
			}
		})
	}
}

func TestProvider_ChatWithTools_AllowsSupportedConverseCatalogAlias(t *testing.T) {
	output, _ := newClosedConverseStreamOutput(
		&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
			Value: bedrocktypes.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta:             &bedrocktypes.ContentBlockDeltaMemberText{Value: "Alias OK"},
			},
		},
	)
	mockConverse := &mockConverseStreamClient{output: output}
	p := &Provider{converseClient: mockConverse}

	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    "corp-nova-pro",
		CatalogModel:    "amazon.nova-pro-v1:0",
		MaxOutputTokens: 64000,
	}
	ctx := newBedrockTestContext(cfg)

	got, err := p.ChatWithTools(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if got != "Alias OK" {
		t.Fatalf("ChatWithTools() = %q, want Alias OK", got)
	}
	if mockConverse.lastInput == nil {
		t.Fatal("ConverseStream() should be called for supported catalog alias")
	}
	if gotModel := aws.ToString(mockConverse.lastInput.ModelId); gotModel != "corp-nova-pro" {
		t.Fatalf("ModelId = %q, want configured alias", gotModel)
	}
}

func TestBuildConverseStreamInput_MaxTokensSelection(t *testing.T) {
	t.Setenv("BEDROCK_FUNCTION_CALLING", "0")

	tests := []struct {
		name      string
		model     string
		cfg       *config.Config
		want      int32
		wantUnset bool
	}{
		{
			name:  "catalog known model uses catalog limit",
			model: "meta.llama3-3-70b-instruct-v1:0",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
					DefaultModel:    "meta.llama3-3-70b-instruct-v1:0",
					MaxOutputTokens: 64000,
				}
				return cfg
			}(),
			want: 4000,
		},
		{
			name:  "model override wins over catalog limit",
			model: "meta.llama3-3-70b-instruct-v1:0",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
					DefaultModel:    "meta.llama3-3-70b-instruct-v1:0",
					MaxOutputTokens: 64000,
					ModelOverrides: map[string]config.ModelOverride{
						"meta.llama3-3-70b-instruct-v1:0": {MaxOutputTokens: 2048},
					},
				}
				return cfg
			}(),
			want: 2048,
		},
		{
			name:  "catalog alias uses catalog model limit",
			model: "corp-nova-pro",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
					DefaultModel:    "corp-nova-pro",
					CatalogModel:    "amazon.nova-pro-v1:0",
					MaxOutputTokens: 64000,
				}
				return cfg
			}(),
			want: 5000,
		},
		{
			name:  "model override wins for unknown model",
			model: "writer.future-model-v1:0",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
					DefaultModel:    "writer.future-model-v1:0",
					MaxOutputTokens: 64000,
					ModelOverrides: map[string]config.ModelOverride{
						"writer.future-model-v1:0": {MaxOutputTokens: 1234},
					},
				}
				return cfg
			}(),
			want: 1234,
		},
		{
			name:  "unknown model omits provider default",
			model: "writer.future-model-v1:0",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
					DefaultModel:    "writer.future-model-v1:0",
					MaxOutputTokens: 64000,
				}
				return cfg
			}(),
			wantUnset: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{}
			ctx := newBedrockTestContext(tt.cfg)
			req := p.resolveBedrockRequestContext(ctx, tt.model)

			input, err := p.buildConverseStreamInput(ctx, "system prompt", []api.Message{{Role: "user", Content: "hello"}}, req)
			if err != nil {
				t.Fatalf("buildConverseStreamInput() error = %v", err)
			}
			if tt.wantUnset {
				if input.InferenceConfig != nil && input.InferenceConfig.MaxTokens != nil {
					t.Fatalf("MaxTokens = %d, want unset for unknown Converse cap", aws.ToInt32(input.InferenceConfig.MaxTokens))
				}
				return
			}
			if input.InferenceConfig == nil || input.InferenceConfig.MaxTokens == nil {
				t.Fatal("MaxTokens is unset, want configured value")
			}
			if got := aws.ToInt32(input.InferenceConfig.MaxTokens); got != tt.want {
				t.Fatalf("MaxTokens = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConvertToConverseMessages_ToolUseAndToolResultHistory(t *testing.T) {
	messages, err := convertToConverseMessages([]api.Message{
		{
			Role: "assistant",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID:   "toolu_1",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"README.md"}`,
					},
				},
			},
		},
		{Role: "tool", ToolCallID: "toolu_1", ToolName: "read_file", Content: "contents"},
		{Role: "tool", ToolCallID: "toolu_2", ToolName: "list_files", Content: "files"},
	})
	if err != nil {
		t.Fatalf("convertToConverseMessages() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
	if messages[0].Role != bedrocktypes.ConversationRoleAssistant {
		t.Fatalf("messages[0].Role = %q, want assistant", messages[0].Role)
	}
	toolUse, ok := messages[0].Content[0].(*bedrocktypes.ContentBlockMemberToolUse)
	if !ok {
		t.Fatalf("messages[0].Content[0] = %T, want tool use", messages[0].Content[0])
	}
	if aws.ToString(toolUse.Value.ToolUseId) != "toolu_1" || aws.ToString(toolUse.Value.Name) != "read_file" {
		t.Fatalf("toolUse = %#v, want id/name", toolUse.Value)
	}
	var args map[string]any
	if err := unmarshalSmithyDocument(toolUse.Value.Input, &args); err != nil {
		t.Fatalf("toolUse input decode error = %v", err)
	}
	if args["path"] != "README.md" {
		t.Fatalf("toolUse args = %#v, want path", args)
	}

	if messages[1].Role != bedrocktypes.ConversationRoleUser {
		t.Fatalf("messages[1].Role = %q, want user tool result", messages[1].Role)
	}
	if len(messages[1].Content) != 2 {
		t.Fatalf("len(tool results) = %d, want 2", len(messages[1].Content))
	}
	result, ok := messages[1].Content[0].(*bedrocktypes.ContentBlockMemberToolResult)
	if !ok {
		t.Fatalf("messages[1].Content[0] = %T, want tool result", messages[1].Content[0])
	}
	if aws.ToString(result.Value.ToolUseId) != "toolu_1" || result.Value.Status != "" {
		t.Fatalf("tool result = %#v, want id without provider-specific status", result.Value)
	}
}

func TestConvertToConverseMessages_InvalidToolArguments(t *testing.T) {
	_, err := convertToConverseMessages([]api.Message{
		{
			Role: "assistant",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID: "toolu_bad",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{bad-json`,
					},
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "toolu_bad") {
		t.Fatalf("convertToConverseMessages() error = %v, want tool id context", err)
	}
}

func TestBuildConverseToolConfig_CombinesContextAndMCPTools(t *testing.T) {
	ctx := api.WithToolDefinitions(context.Background(), []api.ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read file",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
			},
		},
	})

	toolConfig := buildConverseToolConfig(ctx, []api.ToolDefinition{
		{Name: "read_file", Description: "duplicate should be ignored"},
		{Name: "custom_lookup", Strict: true},
	})
	if toolConfig == nil || len(toolConfig.Tools) != 2 {
		t.Fatalf("ToolConfig = %#v, want 2 merged tools", toolConfig)
	}
	first := toolConfig.Tools[0].(*bedrocktypes.ToolMemberToolSpec).Value
	second := toolConfig.Tools[1].(*bedrocktypes.ToolMemberToolSpec).Value
	if aws.ToString(first.Name) != "read_file" {
		t.Fatalf("first.Name = %q, want read_file", aws.ToString(first.Name))
	}
	if aws.ToString(second.Name) != "custom_lookup" {
		t.Fatalf("second.Name = %q, want custom_lookup", aws.ToString(second.Name))
	}
	if second.Strict == nil || !aws.ToBool(second.Strict) {
		t.Fatalf("second.Strict = %#v, want true", second.Strict)
	}
	var schema map[string]any
	if err := unmarshalSmithyDocument(second.InputSchema.(*bedrocktypes.ToolInputSchemaMemberJson).Value, &schema); err != nil {
		t.Fatalf("schema decode error = %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("empty schema fallback = %#v, want object schema", schema)
	}
}

type smithyDocumentMarshaler interface {
	MarshalSmithyDocument() ([]byte, error)
}

func unmarshalSmithyDocument(doc smithyDocumentMarshaler, out any) error {
	raw, err := doc.MarshalSmithyDocument()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func TestProvider_HandleConverseStream_CombinesTextToolCallsAndUsage(t *testing.T) {
	output, reader := newClosedConverseStreamOutput(
		&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
			Value: bedrocktypes.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta:             &bedrocktypes.ContentBlockDeltaMemberText{Value: "Look"},
			},
		},
		&bedrocktypes.ConverseStreamOutputMemberContentBlockStart{
			Value: bedrocktypes.ContentBlockStartEvent{
				ContentBlockIndex: aws.Int32(1),
				Start: &bedrocktypes.ContentBlockStartMemberToolUse{
					Value: bedrocktypes.ToolUseBlockStart{
						ToolUseId: aws.String("toolu_1"),
						Name:      aws.String("read_file"),
					},
				},
			},
		},
		&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
			Value: bedrocktypes.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(1),
				Delta: &bedrocktypes.ContentBlockDeltaMemberToolUse{
					Value: bedrocktypes.ToolUseBlockDelta{Input: aws.String(`{"path":"`)},
				},
			},
		},
		&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
			Value: bedrocktypes.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(1),
				Delta: &bedrocktypes.ContentBlockDeltaMemberToolUse{
					Value: bedrocktypes.ToolUseBlockDelta{Input: aws.String(`README.md"}`)},
				},
			},
		},
		&bedrocktypes.ConverseStreamOutputMemberContentBlockStop{
			Value: bedrocktypes.ContentBlockStopEvent{ContentBlockIndex: aws.Int32(1)},
		},
		&bedrocktypes.ConverseStreamOutputMemberMessageStop{
			Value: bedrocktypes.MessageStopEvent{StopReason: bedrocktypes.StopReasonToolUse},
		},
		&bedrocktypes.ConverseStreamOutputMemberMetadata{
			Value: bedrocktypes.ConverseStreamMetadataEvent{
				Usage: &bedrocktypes.TokenUsage{
					InputTokens:           aws.Int32(10),
					OutputTokens:          aws.Int32(4),
					CacheReadInputTokens:  aws.Int32(2),
					CacheWriteInputTokens: aws.Int32(3),
				},
			},
		},
	)

	var usage api.Usage
	p := &Provider{
		usageCallback: func(u api.Usage) {
			usage = u
		},
	}

	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	content, err := p.handleConverseStream(ctx, output, ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleConverseStream() error = %v", err)
	}
	if !strings.Contains(content, "Look") {
		t.Fatalf("content = %q, want streamed text", content)
	}
	if !strings.Contains(content, `"tool":"read_file"`) || !strings.Contains(content, `"path":"README.md"`) {
		t.Fatalf("content = %q, want tool call JSON", content)
	}
	if usage.InputTokens != 10 || usage.OutputTokens != 4 || usage.CachedInputTokens != 2 || usage.CacheCreationTokens != 3 {
		t.Fatalf("usage = %#v, want token usage from metadata", usage)
	}
	if !reader.closed {
		t.Fatal("converse stream reader should be closed")
	}
}

func TestProvider_ChatWithImage_ConverseRouteRejectsImages(t *testing.T) {
	mockConverse := &mockConverseStreamClient{}
	p := &Provider{converseClient: mockConverse}

	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    "amazon.nova-pro-v1:0",
		MaxOutputTokens: 4096,
	}
	ctx := newBedrockTestContext(cfg)

	_, err := p.ChatWithImage(ctx, "system prompt", nil, "describe", &api.ImageData{
		Base64:    "aGVsbG8=",
		MediaType: "image/png",
	}, "")
	if err == nil || !strings.Contains(err.Error(), "does not support image input yet") {
		t.Fatalf("ChatWithImage() error = %v, want Converse image unsupported error", err)
	}
	if mockConverse.lastInput != nil {
		t.Fatal("ConverseStream() should not be called for unsupported image input")
	}
}
