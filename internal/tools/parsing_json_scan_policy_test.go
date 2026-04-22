package tools

import (
	"io"
	"testing"
)

func TestDefaultJSONToolCallScanErrorPolicy_IncompleteJSONObjectStops(t *testing.T) {
	policy := defaultJSONToolCallScanErrorPolicy()
	got := policy.Decide(jsonToolCallScanError{kind: jsonToolCallScanErrorIncompleteJSONObject, start: 10})
	if got != jsonToolCallScanDecisionStop {
		t.Fatalf("Decide(incomplete) = %v, want %v", got, jsonToolCallScanDecisionStop)
	}
}

func TestJSONToolCallScanner_ContinuePolicySkipsIncompleteCandidate(t *testing.T) {
	input := `{"tool": "read_file", "args": {"path": "a.go"}}
{"tool": "str_replace", "args": {"path": "a.go", "old_str": "x", "new_str": "y"
{"tool": "bash", "args": {"command": "echo ok"}}`

	logger := newParseDebugLogger(false, io.Discard)
	scanner := newJSONToolCallScanner(input, nil, logger)
	scanner.errorPolicy = jsonToolCallScanErrorPolicy{onIncompleteJSONObject: jsonToolCallScanDecisionContinue}
	decoder := newJSONToolCallDecoder(logger)

	var tools []string
	for {
		candidate, ok := scanner.Next()
		if !ok {
			break
		}
		if tc, ok := decoder.Decode(candidate); ok {
			tools = append(tools, tc.Tool)
		}
	}

	if len(tools) != 2 {
		t.Fatalf("decoded tools = %v, want 2 tools", tools)
	}
	if tools[0] != "read_file" || tools[1] != "bash" {
		t.Fatalf("decoded tools = %v, want [read_file bash]", tools)
	}
}
