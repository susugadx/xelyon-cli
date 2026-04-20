package claudestream

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func runCancelCase(t *testing.T, mode CancelMode) (string, error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	reader, writer := io.Pipe()
	resp := &http.Response{
		Body: io.NopCloser(reader),
	}

	go func() {
		_, _ = io.WriteString(writer, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`+"\n\n")
		_, _ = io.WriteString(writer, `data: {"type":"message_delta","usage":{"output_tokens":1}}`+"\n\n")
		<-ctx.Done()
		_ = writer.Close()
	}()

	seenText := false

	return RunStreamingResponse(ctx, resp, ui.NewSpinnerWithWriter(io.Discard), func(event StreamEvent, _ string) (string, bool, error) {
		if event.Type == "content_block_delta" {
			seenText = true
			return event.Delta.Text, false, nil
		}
		if seenText && event.Type == "message_delta" {
			cancel()
		}
		return "", false, nil
	}, RunnerOptions{
		CancelMode:        mode,
		WarnOnPartial:     false,
		IgnoreDecodeError: false,
		EnableIdleTimeout: false,
	})
}

func TestRunStreamingResponse_CancelModePartialAsSuccess(t *testing.T) {
	got, err := runCancelCase(t, CancelModePartialAsSuccess)
	if err != nil {
		t.Fatalf("RunStreamingResponse() error = %v, want nil", err)
	}
	if got != "Hello" {
		t.Fatalf("RunStreamingResponse() = %q, want %q", got, "Hello")
	}
}

func TestRunStreamingResponse_CancelModePartialAsError(t *testing.T) {
	got, err := runCancelCase(t, CancelModePartialAsError)
	if err != context.Canceled {
		t.Fatalf("RunStreamingResponse() error = %v, want %v", err, context.Canceled)
	}
	if got != "Hello" {
		t.Fatalf("RunStreamingResponse() = %q, want %q", got, "Hello")
	}
}
