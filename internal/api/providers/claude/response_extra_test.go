package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func streamingResponseForTest(body string) *http.Response {
	return &http.Response{
		Header: http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:   io.NopCloser(strings.NewReader(body)),
	}
}

func jsonResponseForTest(t *testing.T, payload any) *http.Response {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	return &http.Response{
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   io.NopCloser(bytes.NewReader(data)),
	}
}

func marshalStreamEvent(t *testing.T, event StreamEvent) string {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return "data: " + string(data)
}

func TestChatWithTools_SetsThinkingConfigByModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"

	adaptiveReq, _ := captureClaudeRequest(t, cfg, "claude-opus-4-6")
	if adaptiveReq.Thinking == nil || adaptiveReq.Thinking.Type != "adaptive" {
		t.Fatalf("adaptive Thinking = %+v, want adaptive type", adaptiveReq.Thinking)
	}
	if adaptiveReq.OutputConfig == nil || adaptiveReq.OutputConfig.Effort != "max" {
		t.Fatalf("adaptive OutputConfig = %+v, want effort=max", adaptiveReq.OutputConfig)
	}

	enabledReq, _ := captureClaudeRequest(t, cfg, "claude-3-5-sonnet")
	if enabledReq.Thinking == nil || enabledReq.Thinking.Type != "enabled" {
		t.Fatalf("legacy Thinking = %+v, want enabled type", enabledReq.Thinking)
	}
	if enabledReq.Thinking.BudgetTokens != LevelToBudgetTokens("xhigh") {
		t.Fatalf("BudgetTokens = %d, want %d", enabledReq.Thinking.BudgetTokens, LevelToBudgetTokens("xhigh"))
	}
	if enabledReq.OutputConfig != nil {
		t.Fatalf("legacy OutputConfig = %+v, want nil", enabledReq.OutputConfig)
	}
}

func TestHandleStreamingResponse_TracksCompactionToolUseAndUsage(t *testing.T) {
	p := New("test-key")

	var gotUsage api.Usage
	var callbackCount int
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
		callbackCount++
	})

	var out bytes.Buffer
	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(strings.NewReader(""), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	body := strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":5,"cache_read_input_tokens":2,"cache_creation_input_tokens":1}}}`,
		marshalStreamEvent(t, StreamEvent{
			Type:  "content_block_start",
			Index: 0,
			ContentBlock: &ContentBlock{
				Type: "compaction",
			},
		}),
		marshalStreamEvent(t, StreamEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: &Delta{Type: "text_delta", Text: "trimmed "},
		}),
		marshalStreamEvent(t, StreamEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: &Delta{Type: "text_delta", Text: "context"},
		}),
		marshalStreamEvent(t, StreamEvent{Type: "content_block_stop", Index: 0}),
		marshalStreamEvent(t, StreamEvent{
			Type:  "content_block_delta",
			Index: 1,
			Delta: &Delta{Type: "text_delta", Text: "Hello"},
		}),
		marshalStreamEvent(t, StreamEvent{
			Type:  "content_block_start",
			Index: 2,
			ContentBlock: &ContentBlock{
				Type: "tool_use",
				ID:   "toolu_01TEST",
				Name: "read_file",
			},
		}),
		marshalStreamEvent(t, StreamEvent{
			Type:  "content_block_delta",
			Index: 2,
			Delta: &Delta{Type: "input_json_delta", PartialJSON: `{"path":"`},
		}),
		marshalStreamEvent(t, StreamEvent{
			Type:  "content_block_delta",
			Index: 2,
			Delta: &Delta{Type: "input_json_delta", PartialJSON: `/tmp/a.txt"}`},
		}),
		marshalStreamEvent(t, StreamEvent{Type: "content_block_stop", Index: 2}),
		marshalStreamEvent(t, StreamEvent{
			Type: "message_delta",
			Usage: &StreamUsage{
				InputTokens:              6,
				OutputTokens:             11,
				CacheReadInputTokens:     3,
				CacheCreationInputTokens: 2,
			},
		}),
		marshalStreamEvent(t, StreamEvent{Type: "message_stop"}),
	}, "\n\n") + "\n\n"

	result, err := p.handleStreamingResponse(ctx, streamingResponseForTest(body), ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}

	for _, fragment := range []string{
		"[COMPACTION]\ntrimmed context\n[/COMPACTION]\nHello",
		`"tool":"read_file"`,
		`"path":"/tmp/a.txt"`,
		`"id":"toolu_01TEST"`,
	} {
		if !strings.Contains(result, fragment) {
			t.Fatalf("handleStreamingResponse() missing %q in %q", fragment, result)
		}
	}

	if callbackCount != 1 {
		t.Fatalf("usage callback count = %d, want 1", callbackCount)
	}
	if gotUsage.InputTokens != 11 || gotUsage.OutputTokens != 11 || gotUsage.CachedInputTokens != 3 || gotUsage.CacheCreationTokens != 2 {
		t.Fatalf("usage callback = %+v, want input=11 output=11 cached=3 created=2", gotUsage)
	}
}

func TestHandleStreamingResponse_UsageWithoutMessageStart(t *testing.T) {
	p := New("test-key")

	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	body := strings.Join([]string{
		marshalStreamEvent(t, StreamEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: &Delta{Type: "text_delta", Text: "Fallback"},
		}),
		marshalStreamEvent(t, StreamEvent{
			Type: "message_delta",
			Usage: &StreamUsage{
				InputTokens:              4,
				OutputTokens:             9,
				CacheReadInputTokens:     1,
				CacheCreationInputTokens: 2,
			},
		}),
		marshalStreamEvent(t, StreamEvent{Type: "message_stop"}),
	}, "\n\n") + "\n\n"

	ctx := api.WithAssistantUpdateMode(context.Background(), api.AssistantUpdatesOff)
	result, err := p.handleStreamingResponse(ctx, streamingResponseForTest(body), ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v", err)
	}
	if result != "Fallback" {
		t.Fatalf("handleStreamingResponse() = %q, want %q", result, "Fallback")
	}
	if gotUsage.InputTokens != 7 || gotUsage.OutputTokens != 9 {
		t.Fatalf("usage callback = %+v, want input=7 output=9", gotUsage)
	}
}

func TestHandleStreamingResponse_ContextCanceledReturnsPartialWithoutError(t *testing.T) {
	p := New("test-key")

	ctx, cancel := context.WithCancel(context.Background())
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	var errOut bytes.Buffer
	ctx = ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), io.Discard, &errOut))

	reader, writer := io.Pipe()
	resp := &http.Response{
		Header: http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:   io.NopCloser(reader),
	}

	written := make(chan struct{})
	go func() {
		_, _ = io.WriteString(writer, marshalStreamEvent(t, StreamEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: &Delta{Type: "text_delta", Text: "Hello"},
		})+"\n\n")
		close(written)
		<-ctx.Done()
		_ = writer.Close()
	}()

	go func() {
		<-written
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	result, err := p.handleStreamingResponse(ctx, resp, ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleStreamingResponse() error = %v, want nil", err)
	}
	if result != "Hello" {
		t.Fatalf("handleStreamingResponse() = %q, want %q", result, "Hello")
	}
	if !strings.Contains(errOut.String(), "Partial result returned.") {
		t.Fatalf("stderr = %q, want partial warning", errOut.String())
	}
}

func TestHandleNonStreamingResponse_EmitsTextToolJSONAndUsage(t *testing.T) {
	p := New("test-key")

	var gotUsage api.Usage
	p.SetUsageCallback(func(usage api.Usage) {
		gotUsage = usage
	})

	var out bytes.Buffer
	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(strings.NewReader(""), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesVerbose)

	resp := jsonResponseForTest(t, Response{
		Content: []Content{
			{Type: "text", Text: "I'll open that file."},
			{Type: "tool_use", ID: "toolu_02TEST", Name: "read_file", Input: map[string]any{"path": "/tmp/readme.md"}},
		},
		Usage: StreamUsage{
			InputTokens:              4,
			OutputTokens:             7,
			CacheReadInputTokens:     3,
			CacheCreationInputTokens: 1,
		},
	})

	result, err := p.handleNonStreamingResponse(ctx, resp, ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleNonStreamingResponse() error = %v", err)
	}

	if !strings.Contains(result, "I'll open that file.") || !strings.Contains(result, `"tool":"read_file"`) {
		t.Fatalf("handleNonStreamingResponse() = %q, want text and tool JSON", result)
	}
	if !strings.Contains(out.String(), "I'll open that file.") {
		t.Fatalf("expected streamed text output, got %q", out.String())
	}
	if gotUsage.InputTokens != 8 || gotUsage.OutputTokens != 7 || gotUsage.CachedInputTokens != 3 || gotUsage.CacheCreationTokens != 1 {
		t.Fatalf("usage callback = %+v, want input=8 output=7 cached=3 created=1", gotUsage)
	}
}

func TestHandleNonStreamingResponse_NoContentError(t *testing.T) {
	p := New("test-key")

	_, err := p.handleNonStreamingResponse(context.Background(), jsonResponseForTest(t, Response{}), ui.NewSpinnerWithWriter(io.Discard))
	if err == nil || !strings.Contains(err.Error(), "no response from API") {
		t.Fatalf("handleNonStreamingResponse() error = %v, want no response error", err)
	}
}

func TestProviderIsFunctionCallingEnabled(t *testing.T) {
	if !New("test-key").IsFunctionCallingEnabled() {
		t.Fatal("IsFunctionCallingEnabled() = false, want true")
	}
}
