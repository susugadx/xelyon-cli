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
