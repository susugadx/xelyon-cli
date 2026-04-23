package tools

import (
	"io"
	"reflect"
	"testing"
)

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
	plan := defaultParseToolCallExecutionPlan()
	options := parseRunOptions{
		registry:    newXMLTestRegistry(t),
		startFinder: newDefaultJSONToolCallStartFinder(),
		plan:        plan,
		logger:      newParseDebugLogger(false, io.Discard),
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
	plan := defaultParseToolCallExecutionPlan()
	options := parseRunOptions{
		registry:    newXMLTestRegistry(t),
		startFinder: newDefaultJSONToolCallStartFinder(),
		plan:        plan,
		logger:      newParseDebugLogger(false, io.Discard),
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

func TestResolveParseToolCallPhases_UsesDefaultWhenEmpty(t *testing.T) {
	got := resolveParseToolCallPhases(nil)
	if len(got) != 2 {
		t.Fatalf("resolveParseToolCallPhases(nil) returned %d phases, want 2", len(got))
	}
}

func TestResolveParseToolCallPhases_UsesInjectedWhenProvided(t *testing.T) {
	custom := []parseToolCallPhase{
		fixedParseToolCallPhase{tool: "custom"},
	}
	got := resolveParseToolCallPhases(custom)
	if !reflect.DeepEqual(got, custom) {
		t.Fatalf("resolveParseToolCallPhases(custom) = %#v, want %#v", got, custom)
	}
}

func TestParseToolCalls_UsesInjectedPhases(t *testing.T) {
	plan := defaultParseToolCallExecutionPlan()
	plan.phases = []parseToolCallPhase{
		parseToolCallPhaseFunc(func(_ parseToolCallContext, _ []*ToolCall) []*ToolCall {
			return []*ToolCall{{Tool: "list_dir", Args: map[string]string{"path": "."}}}
		}),
	}

	options := parseRunOptions{
		registry:    newXMLTestRegistry(t),
		startFinder: newDefaultJSONToolCallStartFinder(),
		plan:        plan,
		logger:      newParseDebugLogger(false, io.Discard),
	}

	got := parseToolCalls(`{"tool":"read_file","args":{"path":"main.go"}}`, options)
	if len(got) != 1 {
		t.Fatalf("parseToolCalls() returned %d calls, want 1", len(got))
	}
	if got[0].Tool != "list_dir" {
		t.Fatalf("Tool = %q, want 'list_dir'", got[0].Tool)
	}
}

func TestParseToolCallExecutionPlan_ResolveCodeBlockRanges_UsesPlanPolicy(t *testing.T) {
	plan := defaultParseToolCallExecutionPlan()
	plan.codeBlockPolicy = markdownCodeBlockPolicy{unclosedFence: markdownUnclosedFencePolicyIgnore}

	got := plan.ResolveCodeBlockRanges("```json\n{\"tool\":\"read_file\"}")
	if len(got) != 0 {
		t.Fatalf("ResolveCodeBlockRanges() returned %d ranges, want 0", len(got))
	}
}

func TestParseToolCallExecutionPlanBuilder_DefaultBuild(t *testing.T) {
	plan := newParseToolCallExecutionPlanBuilder().Build()
	if len(plan.ResolvePhases()) != 2 {
		t.Fatalf("default builder phases = %d, want 2", len(plan.ResolvePhases()))
	}
}

func TestParseToolCallExecutionPlanBuilder_WithPhases(t *testing.T) {
	customPhase := fixedParseToolCallPhase{tool: "custom"}
	plan := newParseToolCallExecutionPlanBuilder().
		WithPhases([]parseToolCallPhase{customPhase}).
		Build()

	got := plan.ResolvePhases()
	if len(got) != 1 {
		t.Fatalf("custom builder phases = %d, want 1", len(got))
	}
	if !reflect.DeepEqual(got, []parseToolCallPhase{customPhase}) {
		t.Fatalf("custom builder phases = %#v, want %#v", got, []parseToolCallPhase{customPhase})
	}
}

func TestParseToolCallExecutionPlanBuilder_WithPhases_CopiesInput(t *testing.T) {
	input := []parseToolCallPhase{
		fixedParseToolCallPhase{tool: "before"},
	}
	plan := newParseToolCallExecutionPlanBuilder().WithPhases(input).Build()

	input[0] = fixedParseToolCallPhase{tool: "after"}

	got := plan.ResolvePhases()
	if len(got) != 1 {
		t.Fatalf("ResolvePhases() returned %d phases, want 1", len(got))
	}

	phase, ok := got[0].(fixedParseToolCallPhase)
	if !ok {
		t.Fatalf("ResolvePhases()[0] type = %T, want fixedParseToolCallPhase", got[0])
	}
	if phase.tool != "before" {
		t.Fatalf("ResolvePhases()[0].tool = %q, want before", phase.tool)
	}
}

func TestParseToolCallExecutionPlan_ResolvePhases_ReturnsCopy(t *testing.T) {
	plan := newParseToolCallExecutionPlanBuilder().
		WithPhases([]parseToolCallPhase{fixedParseToolCallPhase{tool: "one"}}).
		Build()

	first := plan.ResolvePhases()
	first[0] = fixedParseToolCallPhase{tool: "mutated"}

	second := plan.ResolvePhases()
	phase, ok := second[0].(fixedParseToolCallPhase)
	if !ok {
		t.Fatalf("ResolvePhases()[0] type = %T, want fixedParseToolCallPhase", second[0])
	}
	if phase.tool != "one" {
		t.Fatalf("ResolvePhases() should return defensive copy, got %q", phase.tool)
	}
}
