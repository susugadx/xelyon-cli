package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestBuildChatResponsesRequest_StreamCapabilityByModel(t *testing.T) {
	tests := []struct {
		model      string
		wantStream bool
	}{
		{model: "gpt-5.5", wantStream: true},
		{model: "gpt-5.5-2026-04-23", wantStream: true},
		{model: "gpt-5.5-pro", wantStream: false},
		{model: "gpt-5.5-pro-2026-04-23", wantStream: false},
		{model: "gpt-5.4", wantStream: true},
		{model: "gpt-5.4-pro", wantStream: true},
		{model: "gpt-5.3-codex", wantStream: true},
		{model: "gpt-5.2-codex", wantStream: true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			req := New("test-key").buildChatResponsesRequest(
				config.WithContext(context.Background(), config.DefaultConfig()),
				"system",
				[]api.Message{{Role: "user", Content: "hi"}},
				tt.model,
			)
			if req.Stream != tt.wantStream {
				t.Fatalf("Stream = %v, want %v", req.Stream, tt.wantStream)
			}

			raw := marshalResponsesRequestMap(t, req)
			if tt.wantStream {
				if raw["stream"] != true {
					t.Fatalf("JSON stream = %#v, want true", raw["stream"])
				}
				return
			}
			if raw["stream"] == true {
				t.Fatalf("JSON stream = true, want false or omitted")
			}
		})
	}
}

func TestBuildChatResponsesRequest_GPT55ReasoningXHighAndMaxOutput(t *testing.T) {
	tests := []string{"gpt-5.5", "gpt-5.5-pro"}
	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Thinking.Enabled = true
			cfg.Thinking.Level = "xhigh"
			ctx := config.WithContext(context.Background(), cfg)

			req := New("test-key").buildChatResponsesRequest(ctx, "system", []api.Message{{Role: "user", Content: "hi"}}, model)
			if req.Reasoning == nil || req.Reasoning.Effort != "xhigh" {
				t.Fatalf("Reasoning = %#v, want effort xhigh", req.Reasoning)
			}
			if req.MaxOutputTokens != 128000 {
				t.Fatalf("MaxOutputTokens = %d, want 128000", req.MaxOutputTokens)
			}
		})
	}
}

func TestBuildChatResponsesRequest_GPT55ThinkingOffOmitsReasoning(t *testing.T) {
	tests := []string{"gpt-5.5", "gpt-5.5-pro"}
	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			req := New("test-key").buildChatResponsesRequest(
				config.WithContext(context.Background(), config.DefaultConfig()),
				"system",
				[]api.Message{{Role: "user", Content: "hi"}},
				model,
			)
			if req.Reasoning != nil {
				t.Fatalf("Reasoning = %#v, want nil", req.Reasoning)
			}
		})
	}
}

func TestBuildChatResponsesRequest_CodexReasoningFallbackStillLow(t *testing.T) {
	tests := []struct {
		model             string
		wantMaxOutput     int
		checkMaxOutputSet bool
	}{
		{model: "gpt-5.2-codex"},
		{model: "gpt-5.3-codex", wantMaxOutput: 128000, checkMaxOutputSet: true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			req := New("test-key").buildChatResponsesRequest(
				config.WithContext(context.Background(), config.DefaultConfig()),
				"system",
				[]api.Message{{Role: "user", Content: "hi"}},
				tt.model,
			)
			if !req.Stream {
				t.Fatalf("Stream = false, want true for %s", tt.model)
			}
			if req.Reasoning == nil || req.Reasoning.Effort != "low" {
				t.Fatalf("Reasoning = %#v, want low fallback", req.Reasoning)
			}
			if tt.checkMaxOutputSet && req.MaxOutputTokens != tt.wantMaxOutput {
				t.Fatalf("MaxOutputTokens = %d, want %d", req.MaxOutputTokens, tt.wantMaxOutput)
			}
		})
	}
}

func TestBuildChatResponsesRequest_ToolUseDisabledOmitsToolFields(t *testing.T) {
	ctx := api.WithToolUseDisabled(config.WithContext(context.Background(), config.DefaultConfig()))
	p := New("test-key")
	p.SetMCPTools([]api.ToolDefinition{{Name: "custom_lookup", Description: "custom lookup"}})
	p.SetToolChoice("custom_lookup")

	req := p.buildChatResponsesRequest(ctx, "system", []api.Message{{Role: "user", Content: "hi"}}, "gpt-5.4")
	raw := marshalResponsesRequestMap(t, req)
	if _, ok := raw["tools"]; ok {
		t.Fatalf("tools should be omitted when tool use is disabled: %#v", raw["tools"])
	}
	if _, ok := raw["tool_choice"]; ok {
		t.Fatalf("tool_choice should be omitted when tool use is disabled: %#v", raw["tool_choice"])
	}
}

func TestBuildChatResponsesRequest_StoreFalseSendsFullHistoryWithoutPreviousResponseID(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Responses.Store = false
	ctx := config.WithContext(context.Background(), cfg)
	ctx = api.WithCompactedInputItems(ctx, []api.InputItem{{Type: "compacted", Data: "compact-data"}})

	p := New("test-key")
	p.SetResponseID("resp_old")
	req := p.buildChatResponsesRequest(ctx, "system", []api.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "answer"},
		{Role: "user", Content: "next"},
	}, "gpt-5.2-codex")

	if req.Store {
		t.Fatal("Store = true, want false")
	}
	if req.PreviousResponseID != "" {
		t.Fatalf("PreviousResponseID = %q, want empty", req.PreviousResponseID)
	}
	inputItems, ok := req.Input.([]openairesponses.InputItem)
	if !ok {
		t.Fatalf("Input type = %T, want []openairesponses.InputItem", req.Input)
	}
	if len(inputItems) != 5 {
		t.Fatalf("Input length = %d, want developer plus compacted item plus full history", len(inputItems))
	}
	if inputItems[0].Role != "developer" || inputItems[0].Content != "system" {
		t.Fatalf("Input[0] = %#v, want developer system message", inputItems[0])
	}
	if inputItems[1].Type != "compacted" || inputItems[1].Data != "compact-data" {
		t.Fatalf("Input[1] = %#v, want compacted item from context", inputItems[1])
	}
}

func TestBuildChatResponsesRequest_IncludesServerCompactionContextManagementOnPreviousResponseChain(t *testing.T) {
	ctx := config.WithContext(context.Background(), config.DefaultConfig())
	p := New("test-key")
	p.SetResponseID("resp_old")

	req := p.buildChatResponsesRequest(ctx, "system", []api.Message{
		{Role: "user", Content: "hi"},
	}, "gpt-5.5")

	raw := marshalResponsesRequestMap(t, req)
	contextManagementRaw, ok := raw["context_management"].([]any)
	if !ok || len(contextManagementRaw) != 1 {
		t.Fatalf("context_management = %#v, want one compaction item", raw["context_management"])
	}
	compaction, ok := contextManagementRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("context_management[0] type = %T, want map", contextManagementRaw[0])
	}
	if compaction["type"] != "compaction" {
		t.Fatalf("context_management[0].type = %#v, want compaction", compaction["type"])
	}
	threshold, ok := compaction["compact_threshold"].(float64)
	if !ok {
		t.Fatalf("compact_threshold type = %T, want float64(JSON number)", compaction["compact_threshold"])
	}
	if int(threshold) < 1000 {
		t.Fatalf("compact_threshold = %d, want >= 1000", int(threshold))
	}
	if int(threshold) == 0 {
		t.Fatal("compact_threshold = 0, want resolved non-zero value")
	}
}

func TestBuildChatResponsesRequest_OmitsServerCompactionWhenDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Responses.ServerCompaction.Enabled = false
	ctx := config.WithContext(context.Background(), cfg)
	p := New("test-key")
	p.SetResponseID("resp_old")

	req := p.buildChatResponsesRequest(ctx, "system", []api.Message{
		{Role: "user", Content: "hi"},
	}, "gpt-5.5")

	raw := marshalResponsesRequestMap(t, req)
	if _, ok := raw["context_management"]; ok {
		t.Fatalf("context_management should be omitted when disabled: %#v", raw["context_management"])
	}
}

func TestBuildChatResponsesRequest_OmitsServerCompactionWhenContextWindowUnknown(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
	})
	ctx := config.WithContext(context.Background(), cfg)
	p := New("test-key")
	p.SetResponseID("resp_old")

	req := p.buildChatResponsesRequest(ctx, "system", []api.Message{
		{Role: "user", Content: "hi"},
	}, "corp-gpt-deployment")

	raw := marshalResponsesRequestMap(t, req)
	if _, ok := raw["context_management"]; ok {
		t.Fatalf("context_management should be omitted when context window is unknown: %#v", raw["context_management"])
	}
}

func TestLongRunningResponsesHTTPClientDisablesHeaderTimeout(t *testing.T) {
	baseTransport := &http.Transport{ResponseHeaderTimeout: 60 * time.Second}
	baseClient := &http.Client{
		Timeout:   3 * time.Minute,
		Transport: baseTransport,
	}

	got := newLongRunningResponsesHTTPClient(baseClient)
	if got.Timeout != baseClient.Timeout {
		t.Fatalf("Timeout = %v, want %v", got.Timeout, baseClient.Timeout)
	}
	transport, ok := got.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport type = %T, want *http.Transport", got.Transport)
	}
	if transport.ResponseHeaderTimeout != 0 {
		t.Fatalf("ResponseHeaderTimeout = %v, want 0", transport.ResponseHeaderTimeout)
	}
	if baseTransport.ResponseHeaderTimeout != 60*time.Second {
		t.Fatalf("base ResponseHeaderTimeout mutated to %v", baseTransport.ResponseHeaderTimeout)
	}
}

func marshalResponsesRequestMap(t *testing.T, req ResponsesRequest) map[string]any {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	return raw
}
