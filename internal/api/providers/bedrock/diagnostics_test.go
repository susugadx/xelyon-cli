package bedrock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnose_ClaudeMessagesSmokeRequestsReportRequestIDUsageAndCost(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	mockInvoke := &mockInvokeModelWithResponseStreamClient{outputs: []*bedrockruntime.InvokeModelWithResponseStreamOutput{
		newClaudeDiagnosticSmokeOutput("req-text", []string{`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"xelyon bedrock doctor ok"}}`}, 10, 4),
		newClaudeDiagnosticSmokeOutput("req-tool", []string{
			`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"xelyon_bedrock_doctor_probe"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"value\":\"bedrock-tool-ok\"}"}}`,
			`{"type":"content_block_stop","index":0}`,
		}, 11, 5),
		newClaudeDiagnosticSmokeOutput("req-image", []string{`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"red"}}`}, 12, 6),
		newClaudeDiagnosticSmokeOutput("req-thinking", []string{`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"xelyon bedrock thinking ok"}}`}, 13, 7),
	}}

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:        bedrockDiagnosticTestConfig(defaultModel, ""),
		Model:         defaultModel,
		RunSmoke:      true,
		TextSmoke:     true,
		ToolSmoke:     true,
		ImageSmoke:    true,
		ThinkingSmoke: true,
		invokeClient:  mockInvoke,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
	}
	if report.Route != string(bedrockRouteClaudeMessages) {
		t.Fatalf("Route = %q, want Claude Messages", report.Route)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 4 {
		t.Fatalf("Smoke requests = %#v, want 4 requests", report.Smoke)
	}
	wantIDs := map[string]string{
		"text":     "req-text",
		"tool":     "req-tool",
		"image":    "req-image",
		"thinking": "req-thinking",
	}
	for _, request := range report.Smoke.Requests {
		if request.RequestID != wantIDs[request.Name] {
			t.Fatalf("%s request_id = %q, want %q", request.Name, request.RequestID, wantIDs[request.Name])
		}
		if !request.UsageObserved {
			t.Fatalf("%s UsageObserved = false, want true", request.Name)
		}
		if request.Cost.PricingUnavailable || request.Cost.USD <= 0 {
			t.Fatalf("%s Cost = %#v, want available positive cost", request.Name, request.Cost)
		}
	}
	if report.Smoke.Usage.InputTokens != 58 {
		t.Fatalf("summary input tokens = %d, want 58", report.Smoke.Usage.InputTokens)
	}
}

func TestDiagnose_ClaudeSmokePreservesConfiguredAnthropicVersion(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	cfg := bedrockDiagnosticTestConfig(defaultModel, defaultModel)
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:     defaultModel,
		CatalogModel:     defaultModel,
		MaxOutputTokens:  9999,
		AnthropicVersion: "bedrock-test-version",
		AnthropicBeta:    []string{"beta-from-config"},
	}
	mockInvoke := &mockInvokeModelWithResponseStreamClient{output: newClaudeDiagnosticSmokeOutput("req-custom-version", []string{
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}`,
	}, 10, 4)}

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          cfg,
		Model:           defaultModel,
		RunSmoke:        true,
		TextSmoke:       true,
		MaxOutputTokens: 32,
		invokeClient:    mockInvoke,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
	}
	if mockInvoke.lastInput == nil {
		t.Fatal("InvokeModelWithResponseStream() should be called")
	}
	var body struct {
		AnthropicVersion string   `json:"anthropic_version"`
		AnthropicBeta    []string `json:"anthropic_beta"`
		MaxTokens        int      `json:"max_tokens"`
	}
	if err := json.Unmarshal(mockInvoke.lastInput.Body, &body); err != nil {
		t.Fatalf("json.Unmarshal(request body) error = %v", err)
	}
	if body.AnthropicVersion != "bedrock-test-version" {
		t.Fatalf("anthropic_version = %q, want configured version", body.AnthropicVersion)
	}
	if !containsString(body.AnthropicBeta, "beta-from-config") {
		t.Fatalf("anthropic_beta = %v, want configured beta header", body.AnthropicBeta)
	}
	if body.MaxTokens != 32 {
		t.Fatalf("max_tokens = %d, want diagnostic smoke override", body.MaxTokens)
	}
}

func TestDiagnose_ConverseSmokeReportsUsageAndSkipsUnsupportedSmoke(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)
	t.Setenv("BEDROCK_FUNCTION_CALLING", "1")

	mockConverse := &mockConverseStreamClient{outputs: []*bedrockruntime.ConverseStreamOutput{
		newConverseDiagnosticTextOutput("req-converse-text", "converse ok", 20, 8, 3, 2),
		newConverseDiagnosticToolOutput("req-converse-tool", 21, 9, 4, 1),
	}}

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         bedrockDiagnosticTestConfig("amazon.nova-pro-v1:0", ""),
		Model:          "amazon.nova-pro-v1:0",
		RunSmoke:       true,
		TextSmoke:      true,
		ToolSmoke:      true,
		ImageSmoke:     true,
		ThinkingSmoke:  true,
		converseClient: mockConverse,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
	}
	if report.Route != string(bedrockRouteConverseStream) {
		t.Fatalf("Route = %q, want ConverseStream", report.Route)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 4 {
		t.Fatalf("Smoke requests = %#v, want 4 requests", report.Smoke)
	}
	text := bedrockDiagnosticRequest(report.Smoke.Requests, "text")
	if text.RequestID != "req-converse-text" || !text.UsageObserved || text.Usage.CachedInputTokens != 3 || text.Usage.CacheCreationTokens != 2 {
		t.Fatalf("text request = %#v, want request ID and cache usage", text)
	}
	tool := bedrockDiagnosticRequest(report.Smoke.Requests, "tool")
	if tool.RequestID != "req-converse-tool" || !strings.Contains(tool.Content, `"tool":"xelyon_bedrock_doctor_probe"`) {
		t.Fatalf("tool request = %#v, want request ID and tool JSON", tool)
	}
	for _, name := range []string{"image", "thinking"} {
		request := bedrockDiagnosticRequest(report.Smoke.Requests, name)
		if !request.Skipped {
			t.Fatalf("%s request = %#v, want skipped", name, request)
		}
		if !hasBedrockDiagnosticCheck(report, name+"_smoke", DiagnosticStatusWarn) {
			t.Fatalf("missing %s_smoke warn check: %#v", name, report.Checks)
		}
	}
}

func TestDiagnose_CatalogPolicyUsesRuntimeMaxTokenPrecedence(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	tests := []struct {
		name      string
		cfg       *config.Config
		model     string
		wantMax   string
		unwantMax string
		wantRoute string
	}{
		{
			name:      "converse catalog beats provider default",
			cfg:       bedrockDiagnosticPolicyMaxConfig("amazon.nova-pro-v1:0", "amazon.nova-pro-v1:0", 9999, config.ModelOverride{}),
			model:     "amazon.nova-pro-v1:0",
			wantMax:   "max_output_tokens=5000",
			unwantMax: "max_output_tokens=9999",
			wantRoute: string(bedrockRouteConverseStream),
		},
		{
			name:      "converse model override beats provider default",
			cfg:       bedrockDiagnosticPolicyMaxConfig("amazon.nova-pro-v1:0", "amazon.nova-pro-v1:0", 9999, config.ModelOverride{CatalogModel: "amazon.nova-pro-v1:0", MaxOutputTokens: 2048}),
			model:     "amazon.nova-pro-v1:0",
			wantMax:   "max_output_tokens=2048",
			unwantMax: "max_output_tokens=9999",
			wantRoute: string(bedrockRouteConverseStream),
		},
		{
			name:      "claude catalog beats provider default",
			cfg:       bedrockDiagnosticPolicyMaxConfig(defaultModel, defaultModel, 9999, config.ModelOverride{}),
			model:     defaultModel,
			wantMax:   "max_output_tokens=64000",
			unwantMax: "max_output_tokens=9999",
			wantRoute: string(bedrockRouteClaudeMessages),
		},
		{
			name:      "claude model override beats provider default",
			cfg:       bedrockDiagnosticPolicyMaxConfig(defaultModel, defaultModel, 9999, config.ModelOverride{CatalogModel: defaultModel, MaxOutputTokens: 2048}),
			model:     defaultModel,
			wantMax:   "max_output_tokens=2048",
			unwantMax: "max_output_tokens=9999",
			wantRoute: string(bedrockRouteClaudeMessages),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Diagnose(context.Background(), DiagnosticOptions{
				Config:           tt.cfg,
				Model:            tt.model,
				skipAWSAuthCheck: true,
			})

			if report.HasFailures() {
				t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
			}
			if report.Route != tt.wantRoute {
				t.Fatalf("Route = %q, want %q", report.Route, tt.wantRoute)
			}
			check, ok := bedrockDiagnosticCheck(report, "catalog_policy")
			if !ok {
				t.Fatalf("missing catalog_policy check: %#v", report.Checks)
			}
			if !strings.Contains(check.Detail, tt.wantMax) {
				t.Fatalf("catalog_policy detail = %q, want %s", check.Detail, tt.wantMax)
			}
			if strings.Contains(check.Detail, tt.unwantMax) {
				t.Fatalf("catalog_policy detail = %q, should not contain %s", check.Detail, tt.unwantMax)
			}
		})
	}
}

func TestDiagnose_ConverseTextSmokeAllowsToolUseUnsupportedModel(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	mockConverse := &mockConverseStreamClient{output: newConverseDiagnosticTextOutput("req-deepseek-text", "deepseek ok", 20, 8, 0, 0)}
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         bedrockDiagnosticTestConfig("us.deepseek.r1-v1:0", ""),
		Model:          "us.deepseek.r1-v1:0",
		RunSmoke:       true,
		TextSmoke:      true,
		converseClient: mockConverse,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
	}
	if hasBedrockDiagnosticCheck(report, "route", DiagnosticStatusFail) {
		t.Fatalf("route check failed for text-only smoke: %#v", report.Checks)
	}
	request := bedrockDiagnosticRequest(report.Smoke.Requests, "text")
	if request.RequestID != "req-deepseek-text" || !request.UsageObserved {
		t.Fatalf("text request = %#v, want request ID and usage", request)
	}
	if mockConverse.lastInput == nil {
		t.Fatal("ConverseStream() should be called for text-only smoke")
	}
	if mockConverse.lastInput.ToolConfig != nil {
		t.Fatalf("ToolConfig = %#v, want nil for text-only smoke", mockConverse.lastInput.ToolConfig)
	}
}

func TestDiagnose_ConverseToolSmokeRequiresToolUseSupportedModel(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	mockConverse := &mockConverseStreamClient{}
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         bedrockDiagnosticTestConfig("us.deepseek.r1-v1:0", ""),
		Model:          "us.deepseek.r1-v1:0",
		RunSmoke:       true,
		ToolSmoke:      true,
		converseClient: mockConverse,
	})

	if !report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = false, want unsupported tool smoke failure; checks=%#v", report.Checks)
	}
	if !hasBedrockDiagnosticCheck(report, "route", DiagnosticStatusFail) {
		t.Fatalf("missing route fail check: %#v", report.Checks)
	}
	if mockConverse.lastInput != nil {
		t.Fatal("ConverseStream() should not be called for unsupported tool smoke")
	}
}

func TestDiagnose_ToolSmokeSkippedWhenFunctionCallingDisabled(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)
	t.Setenv("BEDROCK_FUNCTION_CALLING", "0")

	mockInvoke := &mockInvokeModelWithResponseStreamClient{err: errors.New("should not call bedrock")}
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       bedrockDiagnosticTestConfig(defaultModel, ""),
		Model:        defaultModel,
		RunSmoke:     true,
		ToolSmoke:    true,
		invokeClient: mockInvoke,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
	}
	request := bedrockDiagnosticRequest(report.Smoke.Requests, "tool")
	if !request.Skipped || !strings.Contains(request.SkipReason, "function calling payloads are disabled") {
		t.Fatalf("tool request = %#v, want skipped function-calling-disabled request", request)
	}
	if mockInvoke.lastInput != nil {
		t.Fatal("InvokeModelWithResponseStream() should not be called when tool smoke is skipped")
	}
	if !hasBedrockDiagnosticCheck(report, "tool_smoke", DiagnosticStatusWarn) {
		t.Fatalf("missing tool_smoke warn check: %#v", report.Checks)
	}
}

func TestDiagnose_RequestIDAndUsageWarningsDoNotFailSmoke(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	mockInvoke := &mockInvokeModelWithResponseStreamClient{output: newClaudeDiagnosticSmokeOutput("", []string{
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}`,
	}, 0, 0)}

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       bedrockDiagnosticTestConfig(defaultModel, ""),
		Model:        defaultModel,
		RunSmoke:     true,
		TextSmoke:    true,
		invokeClient: mockInvoke,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
	}
	request := bedrockDiagnosticRequest(report.Smoke.Requests, "text")
	if request.RequestID != "" || request.UsageObserved {
		t.Fatalf("text request = %#v, want missing request ID and usage", request)
	}
	for _, name := range []string{"text_request_id", "text_usage", "text_cost"} {
		if !hasBedrockDiagnosticCheck(report, name, DiagnosticStatusWarn) {
			t.Fatalf("missing warn check %s: %#v", name, report.Checks)
		}
	}
}

func TestDiagnose_FailedSmokeDoesNotEmitSuccessObservationWarnings(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	mockInvoke := &mockInvokeModelWithResponseStreamClient{err: errors.New("boom")}
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       bedrockDiagnosticTestConfig(defaultModel, ""),
		Model:        defaultModel,
		RunSmoke:     true,
		TextSmoke:    true,
		invokeClient: mockInvoke,
	})

	if !report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = false, want smoke failure; checks=%#v", report.Checks)
	}
	if !hasBedrockDiagnosticCheck(report, "text_smoke", DiagnosticStatusFail) {
		t.Fatalf("missing text_smoke fail check: %#v", report.Checks)
	}
	for _, name := range []string{"text_request_id", "text_usage", "text_cost"} {
		if hasBedrockDiagnosticCheckName(report, name) {
			t.Fatalf("unexpected success-observation check %s for failed smoke: %#v", name, report.Checks)
		}
	}
}

func TestDiagnose_AggregateSmokeUsageRequiresAllRanRequestsObserved(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	mockInvoke := &mockInvokeModelWithResponseStreamClient{outputs: []*bedrockruntime.InvokeModelWithResponseStreamOutput{
		newClaudeDiagnosticSmokeOutput("req-text", []string{
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}`,
		}, 10, 4),
		newClaudeDiagnosticSmokeOutput("req-thinking", []string{
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"thinking ok"}}`,
		}, 0, 0),
	}}
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:        bedrockDiagnosticTestConfig(defaultModel, ""),
		Model:         defaultModel,
		RunSmoke:      true,
		TextSmoke:     true,
		ThinkingSmoke: true,
		invokeClient:  mockInvoke,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
	}
	text := bedrockDiagnosticRequest(report.Smoke.Requests, "text")
	thinking := bedrockDiagnosticRequest(report.Smoke.Requests, "thinking")
	if !text.UsageObserved || thinking.UsageObserved {
		t.Fatalf("text/thinking usage observed = %v/%v, want true/false", text.UsageObserved, thinking.UsageObserved)
	}
	if report.Smoke.UsageObserved {
		t.Fatalf("summary UsageObserved = true, want false when any ran request is missing usage")
	}
	if report.Smoke.Cost.USD <= 0 {
		t.Fatalf("summary cost = %#v, want partial observed cost retained with usage_observed=false", report.Smoke.Cost)
	}
}

func TestDiagnose_PricingUnavailableWarnsWithoutFailingSmoke(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)
	t.Setenv("BEDROCK_FUNCTION_CALLING", "0")

	mockConverse := &mockConverseStreamClient{output: newConverseDiagnosticTextOutput("req-price", "ok", 20, 8, 0, 0)}
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         bedrockDiagnosticTestConfig("amazon.nova-pro-v1:0", "unknown-pricing-model"),
		Model:          "amazon.nova-pro-v1:0",
		CatalogModel:   "unknown-pricing-model",
		RunSmoke:       true,
		TextSmoke:      true,
		converseClient: mockConverse,
	})

	if report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = true; checks=%#v", report.Checks)
	}
	request := bedrockDiagnosticRequest(report.Smoke.Requests, "text")
	if !request.Cost.PricingUnavailable {
		t.Fatalf("Cost = %#v, want pricing unavailable", request.Cost)
	}
	if !hasBedrockDiagnosticCheck(report, "text_cost", DiagnosticStatusWarn) {
		t.Fatalf("missing text_cost warn check: %#v", report.Checks)
	}
}

func TestDiagnose_ToolSmokeFailsWhenToolJSONMissing(t *testing.T) {
	setBedrockDiagnosticTestEnv(t)

	mockInvoke := &mockInvokeModelWithResponseStreamClient{output: newClaudeDiagnosticSmokeOutput("req-tool-missing", []string{
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"no tool call"}}`,
	}, 10, 4)}
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       bedrockDiagnosticTestConfig(defaultModel, ""),
		Model:        defaultModel,
		RunSmoke:     true,
		ToolSmoke:    true,
		invokeClient: mockInvoke,
	})

	if !report.HasFailures() {
		t.Fatalf("Diagnose() HasFailures = false, want tool smoke failure; checks=%#v", report.Checks)
	}
	request := bedrockDiagnosticRequest(report.Smoke.Requests, "tool")
	if !strings.Contains(request.Error, "tool smoke response did not include") {
		t.Fatalf("tool request error = %q, want missing tool error", request.Error)
	}
}

func newClaudeDiagnosticSmokeOutput(requestID string, chunks []string, inputTokens, outputTokens int) *bedrockruntime.InvokeModelWithResponseStreamOutput {
	reader := &fakeResponseStreamReader{
		events: make(chan bedrocktypes.ResponseStream, len(chunks)+3),
	}
	if inputTokens > 0 || outputTokens > 0 {
		reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
			Value: bedrocktypes.PayloadPart{Bytes: []byte(fmt.Sprintf(`{"type":"message_start","message":{"usage":{"input_tokens":%d,"cache_read_input_tokens":2,"cache_creation_input_tokens":1}}}`, inputTokens))},
		}
	}
	for _, chunk := range chunks {
		reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
			Value: bedrocktypes.PayloadPart{Bytes: []byte(chunk)},
		}
	}
	if outputTokens > 0 {
		reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
			Value: bedrocktypes.PayloadPart{Bytes: []byte(fmt.Sprintf(`{"type":"message_delta","usage":{"output_tokens":%d}}`, outputTokens))},
		}
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

func newConverseDiagnosticTextOutput(requestID, text string, inputTokens, outputTokens, cachedTokens, cacheCreationTokens int) *bedrockruntime.ConverseStreamOutput {
	output, _ := newClosedConverseStreamOutput(
		&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
			Value: bedrocktypes.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta:             &bedrocktypes.ContentBlockDeltaMemberText{Value: text},
			},
		},
		&bedrocktypes.ConverseStreamOutputMemberMetadata{
			Value: bedrocktypes.ConverseStreamMetadataEvent{
				Usage: &bedrocktypes.TokenUsage{
					InputTokens:           aws.Int32(int32(inputTokens)),
					OutputTokens:          aws.Int32(int32(outputTokens)),
					CacheReadInputTokens:  aws.Int32(int32(cachedTokens)),
					CacheWriteInputTokens: aws.Int32(int32(cacheCreationTokens)),
				},
			},
		},
	)
	if strings.TrimSpace(requestID) != "" {
		awsmiddleware.SetRequestIDMetadata(&output.ResultMetadata, requestID)
	}
	return output
}

func newConverseDiagnosticToolOutput(requestID string, inputTokens, outputTokens, cachedTokens, cacheCreationTokens int) *bedrockruntime.ConverseStreamOutput {
	output, _ := newClosedConverseStreamOutput(
		&bedrocktypes.ConverseStreamOutputMemberContentBlockStart{
			Value: bedrocktypes.ContentBlockStartEvent{
				ContentBlockIndex: aws.Int32(0),
				Start: &bedrocktypes.ContentBlockStartMemberToolUse{
					Value: bedrocktypes.ToolUseBlockStart{
						ToolUseId: aws.String("toolu_1"),
						Name:      aws.String(diagnosticSmokeToolName),
					},
				},
			},
		},
		&bedrocktypes.ConverseStreamOutputMemberContentBlockDelta{
			Value: bedrocktypes.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta: &bedrocktypes.ContentBlockDeltaMemberToolUse{
					Value: bedrocktypes.ToolUseBlockDelta{Input: aws.String(`{"value":"bedrock-tool-ok"}`)},
				},
			},
		},
		&bedrocktypes.ConverseStreamOutputMemberContentBlockStop{
			Value: bedrocktypes.ContentBlockStopEvent{ContentBlockIndex: aws.Int32(0)},
		},
		&bedrocktypes.ConverseStreamOutputMemberMetadata{
			Value: bedrocktypes.ConverseStreamMetadataEvent{
				Usage: &bedrocktypes.TokenUsage{
					InputTokens:           aws.Int32(int32(inputTokens)),
					OutputTokens:          aws.Int32(int32(outputTokens)),
					CacheReadInputTokens:  aws.Int32(int32(cachedTokens)),
					CacheWriteInputTokens: aws.Int32(int32(cacheCreationTokens)),
				},
			},
		},
	)
	if strings.TrimSpace(requestID) != "" {
		awsmiddleware.SetRequestIDMetadata(&output.ResultMetadata, requestID)
	}
	return output
}

func bedrockDiagnosticTestConfig(model, catalogModel string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:    model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: 64,
	}
	return cfg
}

func bedrockDiagnosticPolicyMaxConfig(model, catalogModel string, providerMax int, override config.ModelOverride) *config.Config {
	cfg := config.DefaultConfig()
	pm := config.ProviderModelConfig{
		DefaultModel:    model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: providerMax,
	}
	if override.CatalogModel != "" || override.MaxOutputTokens > 0 {
		pm.ModelOverrides = map[string]config.ModelOverride{
			model: override,
		}
	}
	cfg.ProviderModels["bedrock"] = pm
	return cfg
}

func bedrockDiagnosticRequest(requests []DiagnosticSmokeRequestResult, name string) DiagnosticSmokeRequestResult {
	for _, request := range requests {
		if request.Name == name {
			return request
		}
	}
	return DiagnosticSmokeRequestResult{}
}

func bedrockDiagnosticCheck(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}

func hasBedrockDiagnosticCheck(report DiagnosticReport, name string, status DiagnosticStatus) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}

func hasBedrockDiagnosticCheckName(report DiagnosticReport, name string) bool {
	for _, check := range report.Checks {
		if check.Name == name {
			return true
		}
	}
	return false
}

func setBedrockDiagnosticTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("BEDROCK_FUNCTION_CALLING", "")
	t.Setenv("XELYON_MODEL", "")
}
