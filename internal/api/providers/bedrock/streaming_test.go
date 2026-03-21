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

func TestProcessChunk_CompactionBlock(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	compactionBlocks := make(map[int]*strings.Builder)
	var toolCallsOutput strings.Builder
	var compactionOutput strings.Builder
	var lastUsage *api.Usage
	spinner := newTestSpinner()

	// content_block_start for compaction
	startData, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]string{
			"type": "compaction",
		},
	})
	text, done := p.processChunk(startData, toolUses, &toolCallsOutput, &lastUsage, compactionBlocks, &compactionOutput, spinner)
	if text != "" || done {
		t.Errorf("expected empty text and done=false for compaction start, got text=%q done=%v", text, done)
	}
	if _, ok := compactionBlocks[0]; !ok {
		t.Fatal("expected compaction block accumulator to be created")
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
	text, _ = p.processChunk(deltaData, toolUses, &toolCallsOutput, &lastUsage, compactionBlocks, &compactionOutput, spinner)
	if text != "" {
		t.Errorf("compaction delta should not produce text output, got: %q", text)
	}

	// content_block_stop for compaction
	stopData, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_stop",
		"index": 0,
	})
	text, done = p.processChunk(stopData, toolUses, &toolCallsOutput, &lastUsage, compactionBlocks, &compactionOutput, spinner)
	if text != "" || done {
		t.Errorf("unexpected output on compaction stop: text=%q done=%v", text, done)
	}
	if !strings.Contains(compactionOutput.String(), "compacted content") {
		t.Errorf("compaction output missing content, got: %q", compactionOutput.String())
	}
}

func TestProcessChunk_MessageDelta_OutputTokens(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	compactionBlocks := make(map[int]*strings.Builder)
	var toolCallsOutput strings.Builder
	var compactionOutput strings.Builder
	var lastUsage *api.Usage
	spinner := newTestSpinner()

	data, _ := json.Marshal(map[string]interface{}{
		"type": "message_delta",
		"usage": map[string]int{
			"output_tokens": 42,
		},
	})
	p.processChunk(data, toolUses, &toolCallsOutput, &lastUsage, compactionBlocks, &compactionOutput, spinner)

	if lastUsage == nil {
		t.Fatal("expected lastUsage to be set")
	}
	if lastUsage.OutputTokens != 42 {
		t.Errorf("OutputTokens = %d, want 42", lastUsage.OutputTokens)
	}
}

func TestProcessChunk_MessageDelta_CacheMetrics(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	compactionBlocks := make(map[int]*strings.Builder)
	var toolCallsOutput strings.Builder
	var compactionOutput strings.Builder
	var lastUsage *api.Usage
	spinner := newTestSpinner()

	data, _ := json.Marshal(map[string]interface{}{
		"type": "message_delta",
		"usage": map[string]int{
			"output_tokens":               50,
			"input_tokens":                100,
			"cache_read_input_tokens":     80,
			"cache_creation_input_tokens": 20,
		},
	})
	p.processChunk(data, toolUses, &toolCallsOutput, &lastUsage, compactionBlocks, &compactionOutput, spinner)

	if lastUsage == nil {
		t.Fatal("expected lastUsage to be set")
	}
	if lastUsage.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", lastUsage.OutputTokens)
	}
	// InputTokens は正規化: input + cache_read + cache_creation
	if lastUsage.InputTokens != 200 {
		t.Errorf("InputTokens = %d, want 200", lastUsage.InputTokens)
	}
	if lastUsage.CachedInputTokens != 80 {
		t.Errorf("CachedInputTokens = %d, want 80", lastUsage.CachedInputTokens)
	}
	if lastUsage.CacheCreationTokens != 20 {
		t.Errorf("CacheCreationTokens = %d, want 20", lastUsage.CacheCreationTokens)
	}
}

func TestProcessChunk_ContentBlockDelta_NilDelta(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	compactionBlocks := make(map[int]*strings.Builder)
	var toolCallsOutput strings.Builder
	var compactionOutput strings.Builder
	var lastUsage *api.Usage
	spinner := newTestSpinner()

	// delta が null の content_block_delta
	data, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
	})
	text, done := p.processChunk(data, toolUses, &toolCallsOutput, &lastUsage, compactionBlocks, &compactionOutput, spinner)
	if text != "" || done {
		t.Errorf("nil delta should produce empty text, got text=%q done=%v", text, done)
	}
}

func TestProcessChunk_ContentBlockStart_NilContentBlock(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	compactionBlocks := make(map[int]*strings.Builder)
	var toolCallsOutput strings.Builder
	var compactionOutput strings.Builder
	var lastUsage *api.Usage
	spinner := newTestSpinner()

	data, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
	})
	text, done := p.processChunk(data, toolUses, &toolCallsOutput, &lastUsage, compactionBlocks, &compactionOutput, spinner)
	if text != "" || done {
		t.Errorf("nil content_block should produce empty text, got text=%q done=%v", text, done)
	}
}

func TestProcessChunk_UnknownEventType(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	compactionBlocks := make(map[int]*strings.Builder)
	var toolCallsOutput strings.Builder
	var compactionOutput strings.Builder
	var lastUsage *api.Usage
	spinner := newTestSpinner()

	data, _ := json.Marshal(map[string]interface{}{
		"type": "unknown_event_type",
	})
	text, done := p.processChunk(data, toolUses, &toolCallsOutput, &lastUsage, compactionBlocks, &compactionOutput, spinner)
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
