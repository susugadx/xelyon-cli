package bedrock

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleEventStream_ClosedStreamFlushesCompactionAndToolCalls(t *testing.T) {
	reader := &fakeResponseStreamReader{
		events: make(chan bedrocktypes.ResponseStream, 7),
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"message_start","message":{"usage":{"input_tokens":5,"cache_read_input_tokens":2}}}`)},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"content_block_start","index":0,"content_block":{"type":"compaction"}}`)},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"summary"}}`)},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"content_block_stop","index":0}`)},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"read_file"}}`)},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"/tmp/demo.txt\"}"}}`)},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"content_block_stop","index":1}`)},
	}
	close(reader.events)

	var usage api.Usage
	p := &Provider{usageCallback: func(u api.Usage) {
		usage = u
	}}

	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	got, err := p.handleEventStream(ctx, newBedrockStreamOutput(reader), ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleEventStream() error = %v", err)
	}
	if !strings.Contains(got, "[COMPACTION]\nsummary\n[/COMPACTION]\n") {
		t.Fatalf("result = %q, want compaction wrapper", got)
	}
	if !strings.Contains(got, `"tool":"read_file"`) {
		t.Fatalf("result = %q, want tool JSON", got)
	}
	if usage.InputTokens != 7 || usage.CachedInputTokens != 2 {
		t.Fatalf("usage = %+v, want input=7 cached=2", usage)
	}
}

func TestHandleEventStream_StreamErrorAfterClose(t *testing.T) {
	reader := &fakeResponseStreamReader{
		events: make(chan bedrocktypes.ResponseStream),
		err:    errors.New("stream boom"),
	}
	close(reader.events)

	p := &Provider{}
	_, err := p.handleEventStream(context.Background(), newBedrockStreamOutput(reader), ui.NewSpinnerWithWriter(io.Discard))
	if err == nil || !strings.Contains(err.Error(), "bedrock stream error") {
		t.Fatalf("handleEventStream() error = %v, want wrapped stream error", err)
	}
}

func TestHandleEventStream_ContextCanceledReturnsPartialContent(t *testing.T) {
	reader := &fakeResponseStreamReader{
		events: make(chan bedrocktypes.ResponseStream),
	}

	sent := make(chan struct{})
	go func() {
		reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
			Value: bedrocktypes.PayloadPart{Bytes: []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`)},
		}
		close(sent)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sent
		cancel()
	}()

	p := &Provider{}
	got, err := p.handleEventStream(ctx, newBedrockStreamOutput(reader), ui.NewSpinnerWithWriter(io.Discard))
	if err != context.Canceled {
		t.Fatalf("handleEventStream() error = %v, want %v", err, context.Canceled)
	}
	if got != "Hello" {
		t.Fatalf("handleEventStream() = %q, want partial content", got)
	}
}
