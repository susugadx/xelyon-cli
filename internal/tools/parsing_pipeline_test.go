package tools

import (
	"io"
	"testing"
)

type parseToolCallPhaseFunc func(ctx parseToolCallContext, current []*ToolCall) []*ToolCall

func (f parseToolCallPhaseFunc) Run(ctx parseToolCallContext, current []*ToolCall) []*ToolCall {
	return f(ctx, current)
}

func TestRunParseToolCallPhases_RunsInOrderAndPassesCurrent(t *testing.T) {
	var order []string

	phase1 := parseToolCallPhaseFunc(func(_ parseToolCallContext, current []*ToolCall) []*ToolCall {
		order = append(order, "phase1")
		if len(current) != 0 {
			t.Fatalf("phase1 received %d current calls, want 0", len(current))
		}
		return []*ToolCall{{Tool: "read_file"}}
	})

	phase2 := parseToolCallPhaseFunc(func(_ parseToolCallContext, current []*ToolCall) []*ToolCall {
		order = append(order, "phase2")
		if len(current) != 1 || current[0].Tool != "read_file" {
			t.Fatalf("phase2 received unexpected current: %+v", current)
		}
		return append(current, &ToolCall{Tool: "list_dir"})
	})

	got := runParseToolCallPhases(parseToolCallContext{}, []parseToolCallPhase{phase1, phase2})
	if len(order) != 2 || order[0] != "phase1" || order[1] != "phase2" {
		t.Fatalf("phase order = %v, want [phase1 phase2]", order)
	}
	if len(got) != 2 || got[0].Tool != "read_file" || got[1].Tool != "list_dir" {
		t.Fatalf("runParseToolCallPhases() result = %+v, want [read_file, list_dir]", got)
	}
}

func TestDefaultParseToolCallPhases_JSONTakesPriority(t *testing.T) {
	options := parseRunOptions{
		registry:        newXMLTestRegistry(t),
		startFinder:     newDefaultJSONToolCallStartFinder(),
		codeBlockPolicy: defaultMarkdownCodeBlockPolicy(),
		logger:          newParseDebugLogger(false, io.Discard),
	}
	ctx := newParseToolCallContext(`{"tool": "read_file", "args": {"path": "main.go"}}
<list_dir><path>.</path></list_dir>`, options)

	got := runParseToolCallPhases(ctx, defaultParseToolCallPhases())
	if len(got) != 1 {
		t.Fatalf("runParseToolCallPhases() returned %d calls, want 1", len(got))
	}
	if got[0].Tool != "read_file" {
		t.Fatalf("Tool = %q, want 'read_file'", got[0].Tool)
	}
}

func TestDefaultParseToolCallPhases_XMLFallbackWhenJSONEmpty(t *testing.T) {
	options := parseRunOptions{
		registry:        newXMLTestRegistry(t),
		startFinder:     newDefaultJSONToolCallStartFinder(),
		codeBlockPolicy: defaultMarkdownCodeBlockPolicy(),
		logger:          newParseDebugLogger(false, io.Discard),
	}
	ctx := newParseToolCallContext(`{"tool": "read_file", "args": {"path": "main.go"}
<list_dir><path>.</path></list_dir>`, options)

	got := runParseToolCallPhases(ctx, defaultParseToolCallPhases())
	if len(got) != 1 {
		t.Fatalf("runParseToolCallPhases() returned %d calls, want 1", len(got))
	}
	if got[0].Tool != "list_dir" {
		t.Fatalf("Tool = %q, want 'list_dir'", got[0].Tool)
	}
}
