package claudestream

import (
	"testing"
)

func TestToolUseCollector_StopAndEncode(t *testing.T) {
	collector := NewToolUseCollector()
	collector.Start(0, "toolu_1", "read_file")

	var spinnerHints []string
	collector.AppendInputDelta(0, `{"pa`, func(toolName string) {
		spinnerHints = append(spinnerHints, toolName)
	})
	collector.AppendInputDelta(0, `th":"/tmp/demo.txt"}`, func(toolName string) {
		spinnerHints = append(spinnerHints, toolName)
	})

	got := collector.StopAndEncode(0, func(id, name string, input map[string]interface{}) (string, error) {
		return id + "|" + name + "|" + input["path"].(string), nil
	})
	if got != "toolu_1|read_file|/tmp/demo.txt" {
		t.Fatalf("StopAndEncode() = %q, want %q", got, "toolu_1|read_file|/tmp/demo.txt")
	}
	if len(spinnerHints) != 2 || spinnerHints[0] != "read_file" || spinnerHints[1] != "read_file" {
		t.Fatalf("spinner hints = %v, want [read_file read_file]", spinnerHints)
	}
}

func TestCompactionCollector(t *testing.T) {
	collector := NewCompactionCollector()
	collector.Start(1)
	if !collector.AppendText(1, "sum") {
		t.Fatal("AppendText(compaction index) = false, want true")
	}
	if collector.AppendText(2, "ignored") {
		t.Fatal("AppendText(non-compaction index) = true, want false")
	}
	collector.Stop(1)
	if got := collector.Output(); got != "sum" {
		t.Fatalf("Output() = %q, want %q", got, "sum")
	}
}

func TestContentBlockCollector_PreservesInterleavedOrder(t *testing.T) {
	collector := NewContentBlockCollector()

	collector.Start(0, &ContentBlock{Type: "thinking"})
	collector.AppendDelta(0, &Delta{Type: "thinking_delta", Thinking: "need a"})
	collector.AppendDelta(0, &Delta{Type: "signature_delta", Signature: "sig_a"})
	collector.Stop(0)

	collector.Start(1, &ContentBlock{Type: "tool_use", ID: "toolu_a", Name: "read_file"})
	collector.AppendDelta(1, &Delta{Type: "input_json_delta", PartialJSON: `{"path":"a.txt"}`})
	collector.Stop(1)

	collector.Start(2, &ContentBlock{Type: "thinking"})
	collector.AppendDelta(2, &Delta{Type: "thinking_delta", Thinking: "need b"})
	collector.AppendDelta(2, &Delta{Type: "signature_delta", Signature: "sig_b"})
	collector.Stop(2)

	collector.Start(3, &ContentBlock{Type: "tool_use", ID: "toolu_b", Name: "read_file"})
	collector.AppendDelta(3, &Delta{Type: "input_json_delta", PartialJSON: `{"path":"b.txt"}`})
	collector.Stop(3)

	blocks := collector.Blocks()
	if len(blocks) != 4 {
		t.Fatalf("len(Blocks()) = %d, want 4", len(blocks))
	}
	wantTypes := []string{"thinking", "tool_use", "thinking", "tool_use"}
	for i, want := range wantTypes {
		if blocks[i].Type != want {
			t.Fatalf("blocks[%d].Type = %q, want %q; blocks=%#v", i, blocks[i].Type, want, blocks)
		}
	}
	if blocks[1].ID != "toolu_a" || blocks[1].Input["path"] != "a.txt" {
		t.Fatalf("blocks[1] = %#v, want first tool_use", blocks[1])
	}
	if blocks[2].Thinking != "need b" || blocks[2].Signature != "sig_b" {
		t.Fatalf("blocks[2] = %#v, want second thinking", blocks[2])
	}
	if blocks[3].ID != "toolu_b" || blocks[3].Input["path"] != "b.txt" {
		t.Fatalf("blocks[3] = %#v, want second tool_use", blocks[3])
	}
}
