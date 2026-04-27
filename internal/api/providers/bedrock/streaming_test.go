package bedrock

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newTestSpinner() *ui.Spinner {
	return ui.NewSpinnerWithWriter(io.Discard)
}

func newTestStreamState() *bedrockStreamState {
	return newBedrockStreamState(newTestSpinner())
}

func TestProcessChunk_CompactionBlock(t *testing.T) {
	p := &Provider{}
	state := newTestStreamState()

	// content_block_start for compaction
	startData, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]string{
			"type": "compaction",
		},
	})
	text, done := p.processChunk(startData, state)
	if text != "" || done {
		t.Errorf("expected empty text and done=false for compaction start, got text=%q done=%v", text, done)
	}

	// content_block_delta for compaction (should not produce text output)
	deltaData, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]string{
			"type": "text_delta",
			"text": "compacted content",
		},
	})
	text, _ = p.processChunk(deltaData, state)
	if text != "" {
		t.Errorf("compaction delta should not produce text output, got: %q", text)
	}

	// content_block_stop for compaction
	stopData, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_stop",
		"index": 0,
	})
	text, done = p.processChunk(stopData, state)
	if text != "" || done {
		t.Errorf("unexpected output on compaction stop: text=%q done=%v", text, done)
	}
	if !strings.Contains(state.compaction.Output(), "compacted content") {
		t.Errorf("compaction output missing content, got: %q", state.compaction.Output())
	}
}

func TestProcessChunk_MessageDelta_OutputTokens(t *testing.T) {
	p := &Provider{}
	state := newTestStreamState()

	data, _ := json.Marshal(map[string]interface{}{
		"type": "message_delta",
		"usage": map[string]int{
			"output_tokens": 42,
		},
	})
	p.processChunk(data, state)

	if state.lastUsage == nil {
		t.Fatal("expected lastUsage to be set")
	}
	if state.lastUsage.OutputTokens != 42 {
		t.Errorf("OutputTokens = %d, want 42", state.lastUsage.OutputTokens)
	}
}

func TestProcessChunk_MessageDelta_CacheMetrics(t *testing.T) {
	p := &Provider{}
	state := newTestStreamState()

	data, _ := json.Marshal(map[string]interface{}{
		"type": "message_delta",
		"usage": map[string]int{
			"output_tokens":               50,
			"input_tokens":                100,
			"cache_read_input_tokens":     80,
			"cache_creation_input_tokens": 20,
		},
	})
	p.processChunk(data, state)

	if state.lastUsage == nil {
		t.Fatal("expected lastUsage to be set")
	}
	if state.lastUsage.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", state.lastUsage.OutputTokens)
	}
	// InputTokens は正規化: input + cache_read + cache_creation
	if state.lastUsage.InputTokens != 200 {
		t.Errorf("InputTokens = %d, want 200", state.lastUsage.InputTokens)
	}
	if state.lastUsage.CachedInputTokens != 80 {
		t.Errorf("CachedInputTokens = %d, want 80", state.lastUsage.CachedInputTokens)
	}
	if state.lastUsage.CacheCreationTokens != 20 {
		t.Errorf("CacheCreationTokens = %d, want 20", state.lastUsage.CacheCreationTokens)
	}
}

func TestProcessChunk_ThinkingBlock(t *testing.T) {
	p := &Provider{}
	state := newTestStreamState()

	startData, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]string{
			"type": "thinking",
		},
	})
	text, done := p.processChunk(startData, state)
	if text != "" || done {
		t.Fatalf("thinking start text=%q done=%v, want empty false", text, done)
	}

	deltaData, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]string{
			"type":     "thinking_delta",
			"thinking": "need a file",
		},
	})
	text, done = p.processChunk(deltaData, state)
	if text != "" || done {
		t.Fatalf("thinking delta text=%q done=%v, want empty false", text, done)
	}

	sigData, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]string{
			"type":      "signature_delta",
			"signature": "sig_1",
		},
	})
	p.processChunk(sigData, state)

	stopData, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_stop",
		"index": 0,
	})
	p.processChunk(stopData, state)

	contentBlocks := state.contentBlocks.Blocks()
	if len(contentBlocks) != 1 {
		t.Fatalf("len(content blocks) = %d, want 1", len(contentBlocks))
	}
	blocks := api.AnthropicThinkingBlocksFromContentBlocks(contentBlocks)
	if len(blocks) != 1 {
		t.Fatalf("len(thinking blocks) = %d, want 1", len(blocks))
	}
	if blocks[0].Type != "thinking" || blocks[0].Thinking != "need a file" || blocks[0].Signature != "sig_1" {
		t.Fatalf("thinking block = %#v, want preserved thinking/signature", blocks[0])
	}
}

func TestProcessChunk_ContentBlockDelta_NilDelta(t *testing.T) {
	p := &Provider{}
	state := newTestStreamState()

	// delta が null の content_block_delta
	data, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
	})
	text, done := p.processChunk(data, state)
	if text != "" || done {
		t.Errorf("nil delta should produce empty text, got text=%q done=%v", text, done)
	}
}

func TestProcessChunk_ContentBlockStart_NilContentBlock(t *testing.T) {
	p := &Provider{}
	state := newTestStreamState()

	data, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
	})
	text, done := p.processChunk(data, state)
	if text != "" || done {
		t.Errorf("nil content_block should produce empty text, got text=%q done=%v", text, done)
	}
}

func TestProcessChunk_UnknownEventType(t *testing.T) {
	p := &Provider{}
	state := newTestStreamState()

	data, _ := json.Marshal(map[string]interface{}{
		"type": "unknown_event_type",
	})
	text, done := p.processChunk(data, state)
	if text != "" || done {
		t.Errorf("unknown event should be ignored, got text=%q done=%v", text, done)
	}
}

func TestIsBedrockCompactionSupported(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"us.anthropic.claude-opus-4-6-20260409-v1:0", true},
		{"us.anthropic.claude-opus-4-5-20250529-v1:0", true},
		{"us.anthropic.claude-sonnet-4-6-20260514-v1:0", true},
		{"us.anthropic.claude-3-haiku-20240307-v1:0", false},
		{"some-other-model", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := isBedrockCompactionSupported(tt.model)
			if got != tt.want {
				t.Errorf("isBedrockCompactionSupported(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
