package bedrock

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type fakeResponseStreamReader struct {
	events chan bedrocktypes.ResponseStream
	err    error
	closed bool
}

func (f *fakeResponseStreamReader) Events() <-chan bedrocktypes.ResponseStream {
	return f.events
}

func (f *fakeResponseStreamReader) Close() error {
	f.closed = true
	return nil
}

func (f *fakeResponseStreamReader) Err() error {
	return f.err
}

func TestProvider_HandleEventStream_CombinesTextToolCallsAndUsage(t *testing.T) {
	reader := &fakeResponseStreamReader{
		events: make(chan bedrocktypes.ResponseStream, 8),
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{
			Bytes: []byte(`{"type":"message_start","message":{"usage":{"input_tokens":5,"cache_read_input_tokens":2}}}`),
		},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{
			Bytes: []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`),
		},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{
			Bytes: []byte(`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"read_file"}}`),
		},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{
			Bytes: []byte(`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"/tmp/file.txt\"}"}}`),
		},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{
			Bytes: []byte(`{"type":"content_block_stop","index":1}`),
		},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{
			Bytes: []byte(`{"type":"message_delta","usage":{"output_tokens":7}}`),
		},
	}
	reader.events <- &bedrocktypes.ResponseStreamMemberChunk{
		Value: bedrocktypes.PayloadPart{
			Bytes: []byte(`{"type":"message_stop"}`),
		},
	}

	output := &bedrockruntime.InvokeModelWithResponseStreamOutput{}
	stream := bedrockruntime.NewInvokeModelWithResponseStreamEventStream(func(es *bedrockruntime.InvokeModelWithResponseStreamEventStream) {
		es.Reader = reader
	})
	setUnexported(output, "eventStream", stream)

	var usage api.Usage
	p := &Provider{
		usageCallback: func(u api.Usage) {
			usage = u
		},
	}

	var out bytes.Buffer
	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(bytes.NewReader(nil), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	content, err := p.handleEventStream(ctx, output, ui.NewSpinnerWithWriter(io.Discard))
	if err != nil {
		t.Fatalf("handleEventStream() error = %v", err)
	}
	if !strings.Contains(content, "Hello") {
		t.Fatalf("content = %q, want streamed text", content)
	}
	if !strings.Contains(content, `"tool":"read_file"`) {
		t.Fatalf("content = %q, want tool call JSON", content)
	}
	if !strings.Contains(content, `"/tmp/file.txt"`) {
		t.Fatalf("content = %q, want tool args", content)
	}
	if usage.InputTokens != 7 {
		t.Fatalf("usage.InputTokens = %d, want 7", usage.InputTokens)
	}
	if usage.OutputTokens != 7 {
		t.Fatalf("usage.OutputTokens = %d, want 7", usage.OutputTokens)
	}
	if !reader.closed {
		t.Fatal("response stream reader should be closed")
	}
}

func setUnexported(target any, fieldName string, value any) {
	field := reflect.ValueOf(target).Elem().FieldByName(fieldName)
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}
