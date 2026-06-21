package bedrock

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestInvokeClaudeMessagesStream_CapturesRequestIDAndClearsBeforeNextRequest(t *testing.T) {
	first := newBedrockInvokeRequestIDOutput("req-invoke-1")
	second := newBedrockInvokeRequestIDOutput("")
	mockClient := &mockInvokeModelWithResponseStreamClient{outputs: []*bedrockruntime.InvokeModelWithResponseStreamOutput{first, second}}
	p := &Provider{client: mockClient}

	ctx := bedrockRequestIDTestContext(config.DefaultConfig())
	for i, want := range []string{"req-invoke-1", ""} {
		if _, err := p.invokeClaudeMessagesStream(ctx, "global.anthropic.claude-sonnet-4-6", BedrockClaudeMessagesRequest{
			AnthropicVersion: bedrockAnthropicVersion,
			MaxTokens:        8,
		}); err != nil {
			t.Fatalf("invoke request %d error = %v", i+1, err)
		}
		if got := p.lastBedrockRequestID(); got != want {
			t.Fatalf("request %d lastBedrockRequestID() = %q, want %q", i+1, got, want)
		}
	}
}

func TestConverseStream_CapturesRequestIDAndClearsBeforeNextRequest(t *testing.T) {
	t.Setenv("BEDROCK_FUNCTION_CALLING", "0")

	first, _ := newClosedConverseStreamOutput(
		&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
			Value: bedrocktypes.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta:             &bedrocktypes.ContentBlockDeltaMemberText{Value: "first"},
			},
		},
	)
	awsmiddleware.SetRequestIDMetadata(&first.ResultMetadata, "req-converse-1")
	second, _ := newClosedConverseStreamOutput(
		&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
			Value: bedrocktypes.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta:             &bedrocktypes.ContentBlockDeltaMemberText{Value: "second"},
			},
		},
	)

	mockConverse := &mockConverseStreamClient{outputs: []*bedrockruntime.ConverseStreamOutput{first, second}}
	p := &Provider{converseClient: mockConverse}
	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{DefaultModel: "amazon.nova-pro-v1:0"}
	ctx := bedrockRequestIDTestContext(cfg)

	for i, want := range []string{"req-converse-1", ""} {
		if _, err := p.ChatWithTools(ctx, "system", []api.Message{{Role: "user", Content: "hello"}}, ""); err != nil {
			t.Fatalf("converse request %d error = %v", i+1, err)
		}
		if got := p.lastBedrockRequestID(); got != want {
			t.Fatalf("request %d lastBedrockRequestID() = %q, want %q", i+1, got, want)
		}
	}
}

func newBedrockInvokeRequestIDOutput(requestID string) *bedrockruntime.InvokeModelWithResponseStreamOutput {
	reader := &fakeResponseStreamReader{
		events: make(chan bedrocktypes.ResponseStream, 2),
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}`)},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"message_stop"}`)},
	}
	close(reader.events)

	output := newBedrockStreamOutput(reader)
	if strings.TrimSpace(requestID) != "" {
		awsmiddleware.SetRequestIDMetadata(&output.ResultMetadata, requestID)
	}
	return output
}

func bedrockRequestIDTestContext(cfg *config.Config) context.Context {
	ctx := config.WithContext(context.Background(), cfg)
	ctx = uiruntime.WithRuntime(ctx, uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	return ctx
}
