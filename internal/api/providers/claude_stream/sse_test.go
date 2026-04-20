package claudestream

import (
	"testing"
)

func TestParseSSEDataLine(t *testing.T) {
	data, handled := ParseSSEDataLine(`data: {"type":"message_stop"}`)
	if !handled {
		t.Fatal("ParseSSEDataLine(data line) handled = false, want true")
	}
	if data != `{"type":"message_stop"}` {
		t.Fatalf("ParseSSEDataLine() data = %q, want %q", data, `{"type":"message_stop"}`)
	}

	if _, handled := ParseSSEDataLine("event: ping"); handled {
		t.Fatal("ParseSSEDataLine(non-data line) handled = true, want false")
	}
}

func TestDecodeMessageStartUsage(t *testing.T) {
	usage, err := DecodeMessageStartUsage(`{"type":"message_start","message":{"usage":{"input_tokens":5,"output_tokens":1,"cache_read_input_tokens":2,"cache_creation_input_tokens":1}}}`)
	if err != nil {
		t.Fatalf("DecodeMessageStartUsage() error = %v", err)
	}
	if usage.InputTokens != 8 || usage.OutputTokens != 1 || usage.CachedInputTokens != 2 || usage.CacheCreationTokens != 1 {
		t.Fatalf("DecodeMessageStartUsage() = %+v, want input=8 output=1 cached=2 creation=1", usage)
	}
}

func TestUpdateUsageFromMessageDelta(t *testing.T) {
	current := UpdateUsageFromMessageDelta(nil, &StreamUsage{OutputTokens: 7}, false)
	if current == nil || current.OutputTokens != 7 {
		t.Fatalf("UpdateUsageFromMessageDelta(output only) = %+v, want output=7", current)
	}

	current = UpdateUsageFromMessageDelta(current, &StreamUsage{
		InputTokens:              3,
		OutputTokens:             9,
		CacheReadInputTokens:     2,
		CacheCreationInputTokens: 1,
	}, true)
	if current.InputTokens != 6 || current.OutputTokens != 9 || current.CachedInputTokens != 2 || current.CacheCreationTokens != 1 {
		t.Fatalf("UpdateUsageFromMessageDelta(with fallback) = %+v, want input=6 output=9 cached=2 creation=1", current)
	}
}

func TestHandleContentBlockEvents(t *testing.T) {
	toolUses := NewToolUseCollector()
	compaction := NewCompactionCollector()

	HandleContentBlockStart(StreamEvent{
		Type:  "content_block_start",
		Index: 0,
		ContentBlock: &ContentBlock{
			Type: "compaction",
		},
	}, toolUses, compaction)
	HandleContentBlockStart(StreamEvent{
		Type:  "content_block_start",
		Index: 1,
		ContentBlock: &ContentBlock{
			Type: "tool_use",
			ID:   "toolu_1",
			Name: "read_file",
		},
	}, toolUses, compaction)

	if got := HandleContentBlockDelta(StreamEvent{
		Type:  "content_block_delta",
		Index: 0,
		Delta: &Delta{Type: "text_delta", Text: "summary"},
	}, toolUses, compaction, nil); got != "" {
		t.Fatalf("HandleContentBlockDelta(compaction) = %q, want empty", got)
	}
	if got := HandleContentBlockDelta(StreamEvent{
		Type:  "content_block_delta",
		Index: 2,
		Delta: &Delta{Type: "text_delta", Text: "Hello"},
	}, toolUses, compaction, nil); got != "Hello" {
		t.Fatalf("HandleContentBlockDelta(text) = %q, want %q", got, "Hello")
	}

	var spinnerHints []string
	HandleContentBlockDelta(StreamEvent{
		Type:  "content_block_delta",
		Index: 1,
		Delta: &Delta{Type: "input_json_delta", PartialJSON: `{"path":"`},
	}, toolUses, compaction, func(toolName string) {
		spinnerHints = append(spinnerHints, toolName)
	})
	HandleContentBlockDelta(StreamEvent{
		Type:  "content_block_delta",
		Index: 1,
		Delta: &Delta{Type: "input_json_delta", PartialJSON: `/tmp/demo.txt"}`},
	}, toolUses, compaction, func(toolName string) {
		spinnerHints = append(spinnerHints, toolName)
	})

	HandleContentBlockStop(StreamEvent{Type: "content_block_stop", Index: 0}, toolUses, compaction, nil)
	toolJSON := HandleContentBlockStop(StreamEvent{Type: "content_block_stop", Index: 1}, toolUses, compaction, func(id, name string, input map[string]interface{}) (string, error) {
		return id + "|" + name + "|" + input["path"].(string), nil
	})

	if toolJSON != "toolu_1|read_file|/tmp/demo.txt" {
		t.Fatalf("HandleContentBlockStop(tool_use) = %q, want %q", toolJSON, "toolu_1|read_file|/tmp/demo.txt")
	}
	if got := compaction.Output(); got != "summary" {
		t.Fatalf("compaction.Output() = %q, want %q", got, "summary")
	}
	if len(spinnerHints) != 2 || spinnerHints[0] != "read_file" || spinnerHints[1] != "read_file" {
		t.Fatalf("spinner hints = %v, want [read_file read_file]", spinnerHints)
	}
}
