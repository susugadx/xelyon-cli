package claudestream

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProcessLineEvent_EmptyLineIsSkipped(t *testing.T) {
	state := newRunnerOutputState(context.Background(), nil)
	result := state.processLineEvent("", func(_ StreamEvent, _ string) (string, bool, error) {
		t.Fatal("handler should not be called for empty line")
		return "", false, nil
	}, false)

	if !result.skip {
		t.Fatal("processLineEvent() skip = false, want true")
	}
}

func TestProcessLineEvent_NonDataLineIsSkipped(t *testing.T) {
	state := newRunnerOutputState(context.Background(), nil)
	result := state.processLineEvent("event: ping", func(_ StreamEvent, _ string) (string, bool, error) {
		t.Fatal("handler should not be called for non-data line")
		return "", false, nil
	}, false)

	if !result.skip {
		t.Fatal("processLineEvent() skip = false, want true")
	}
}

func TestProcessLineEvent_DecodeErrorHonorsIgnoreFlag(t *testing.T) {
	state := newRunnerOutputState(context.Background(), nil)
	line := "data: {not-json"

	ignored := state.processLineEvent(line, func(_ StreamEvent, _ string) (string, bool, error) {
		t.Fatal("handler should not be called when decode fails")
		return "", false, nil
	}, true)
	if !ignored.skip {
		t.Fatal("processLineEvent(ignore=true) skip = false, want true")
	}

	strict := state.processLineEvent(line, func(_ StreamEvent, _ string) (string, bool, error) {
		t.Fatal("handler should not be called when decode fails")
		return "", false, nil
	}, false)
	if strict.err == nil {
		t.Fatal("processLineEvent(ignore=false) error = nil, want decode error")
	}
	if !strict.decodeErr {
		t.Fatal("processLineEvent(ignore=false) decodeErr = false, want true")
	}
}

func TestProcessLineEvent_HandlerResultIsPropagated(t *testing.T) {
	state := newRunnerOutputState(context.Background(), nil)
	line := `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`

	result := state.processLineEvent(line, func(event StreamEvent, data string) (string, bool, error) {
		if event.Type != "content_block_delta" {
			t.Fatalf("event.Type = %q, want %q", event.Type, "content_block_delta")
		}
		if !strings.Contains(data, `"type":"content_block_delta"`) {
			t.Fatalf("data = %q, want content_block_delta JSON", data)
		}
		return "Hello", false, nil
	}, false)
	if result.err != nil {
		t.Fatalf("processLineEvent() error = %v, want nil", result.err)
	}
	if result.decodeErr {
		t.Fatal("processLineEvent() decodeErr = true, want false")
	}
	if result.skip {
		t.Fatal("processLineEvent() skip = true, want false")
	}
	if result.done {
		t.Fatal("processLineEvent() done = true, want false")
	}
	if result.textDelta != "Hello" {
		t.Fatalf("processLineEvent() textDelta = %q, want %q", result.textDelta, "Hello")
	}
}

func TestProcessLineEvent_HandlerErrorIsPropagated(t *testing.T) {
	state := newRunnerOutputState(context.Background(), nil)
	line := `data: {"type":"message_delta"}`
	wantErr := errors.New("handler failed")

	result := state.processLineEvent(line, func(_ StreamEvent, _ string) (string, bool, error) {
		return "", false, wantErr
	}, false)
	if result.err == nil {
		t.Fatal("processLineEvent() error = nil, want handler error")
	}
	if !errors.Is(result.err, wantErr) {
		t.Fatalf("processLineEvent() error = %v, want %v", result.err, wantErr)
	}
	if result.decodeErr {
		t.Fatal("processLineEvent() decodeErr = true, want false for handler error")
	}
}

func TestProcessLineEvent_DoneIsPropagated(t *testing.T) {
	state := newRunnerOutputState(context.Background(), nil)
	line := `data: {"type":"message_stop"}`

	result := state.processLineEvent(line, func(_ StreamEvent, _ string) (string, bool, error) {
		return "", true, nil
	}, false)
	if result.err != nil {
		t.Fatalf("processLineEvent() error = %v, want nil", result.err)
	}
	if !result.done {
		t.Fatal("processLineEvent() done = false, want true")
	}
}
