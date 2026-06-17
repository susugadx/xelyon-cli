package tools

import (
	"reflect"
	"testing"
)

// MockTool はテスト用のモックツール
type MockTool struct {
	name string
}

func (m *MockTool) Name() string        { return m.name }
func (m *MockTool) Description() string { return "Mock description" }
func (m *MockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (m *MockTool) Run(_ ExecutionContext, args map[string]string) (string, *FileChange, error) {
	return "Success", nil, nil
}

type StructuredMockTool struct {
	name   string
	result ToolRunResult
}

func (m *StructuredMockTool) Name() string        { return m.name }
func (m *StructuredMockTool) Description() string { return "Structured mock description" }
func (m *StructuredMockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (m *StructuredMockTool) Run(_ ExecutionContext, args map[string]string) (string, *FileChange, error) {
	return m.result.Output, m.result.Change, nil
}
func (m *StructuredMockTool) RunResult(_ ExecutionContext, args map[string]string) (ToolRunResult, error) {
	return m.result, nil
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	tool := &MockTool{name: "test_tool"}

	// Test Register and GetTool
	r.Register(tool)
	got := r.GetTool("test_tool")
	if got != tool {
		t.Errorf("GetTool() = %v, want %v", got, tool)
	}

	// Test HasTool
	if !r.HasTool("test_tool") {
		t.Errorf("HasTool() = false, want true")
	}
	if r.HasTool("unknown") {
		t.Errorf("HasTool() = true, want false")
	}

	// Test GetToolDefinitions
	defs := r.GetToolDefinitions()
	if len(defs) != 1 {
		t.Errorf("GetToolDefinitions() length = %d, want 1", len(defs))
	}
	if defs[0].Name != "test_tool" {
		t.Errorf("GetToolDefinitions()[0].Name = %s, want test_tool", defs[0].Name)
	}
	if !reflect.DeepEqual(defs[0].Parameters, tool.Parameters()) {
		t.Errorf("GetToolDefinitions() parameters mismatch")
	}
}

func TestRegistry_ExcludedTools(t *testing.T) {
	r := NewRegistry()
	r.Register(&MockTool{name: "read_file"})
	r.Register(&MockTool{name: "ask_user_question"})

	// 除外なし: 全2ツール
	defs := r.GetToolDefinitions()
	if len(defs) != 2 {
		t.Errorf("GetToolDefinitions() without exclusion = %d, want 2", len(defs))
	}

	// planning 系を除外
	r.SetExcludedTools([]string{"ask_user_question"})
	defs = r.GetToolDefinitions()
	if len(defs) != 1 {
		t.Errorf("GetToolDefinitions() with exclusion = %d, want 1", len(defs))
	}
	if defs[0].Name != "read_file" {
		t.Errorf("GetToolDefinitions()[0].Name = %s, want read_file", defs[0].Name)
	}

	// HasTool は除外に関係なく true
	if !r.HasTool("ask_user_question") {
		t.Error("HasTool(ask_user_question) should be true even when excluded")
	}

	output, change := r.ExecuteWithContext(ExecutionContext{}, &ToolCall{Tool: "ask_user_question"})
	if change != nil {
		t.Fatalf("ExecuteWithContext() change = %+v, want nil", change)
	}
	if output != "Error: tool not available in current mode: ask_user_question" {
		t.Fatalf("ExecuteWithContext() output = %q, want excluded-tool error", output)
	}

	output, change = r.ExecuteWithContext(ExecutionContext{}, &ToolCall{Tool: "read_file"})
	if change != nil {
		t.Fatalf("ExecuteWithContext() change = %+v, want nil", change)
	}
	if output != "Success" {
		t.Fatalf("ExecuteWithContext() output = %q, want Success", output)
	}

	// ClearExcludedTools で復帰
	r.ClearExcludedTools()
	defs = r.GetToolDefinitions()
	if len(defs) != 2 {
		t.Errorf("GetToolDefinitions() after clear = %d, want 2", len(defs))
	}

	output, change = r.ExecuteWithContext(ExecutionContext{}, &ToolCall{Tool: "ask_user_question"})
	if change != nil {
		t.Fatalf("ExecuteWithContext() after clear change = %+v, want nil", change)
	}
	if output != "Success" {
		t.Fatalf("ExecuteWithContext() after clear output = %q, want Success", output)
	}
}

func TestRegistry_GetToolDefinitions_SortedByName(t *testing.T) {
	r := NewRegistry()
	r.Register(&MockTool{name: "z_tool"})
	r.Register(&MockTool{name: "a_tool"})
	r.Register(&MockTool{name: "m_tool"})

	defs := r.GetToolDefinitions()
	if len(defs) != 3 {
		t.Fatalf("GetToolDefinitions() length = %d, want 3", len(defs))
	}

	got := []string{defs[0].Name, defs[1].Name, defs[2].Name}
	want := []string{"a_tool", "m_tool", "z_tool"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetToolDefinitions() names = %v, want %v", got, want)
	}
}

func TestRegistry_GetExcludedTools_SortedByName(t *testing.T) {
	r := NewRegistry()
	r.SetExcludedTools([]string{"z_tool", "a_tool", "m_tool"})

	got := r.GetExcludedTools()
	want := []string{"a_tool", "m_tool", "z_tool"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetExcludedTools() = %v, want %v", got, want)
	}
}

func TestRegistry_GetAPIToolDefinitions_AppliesExcludedToolsAndSortsByName(t *testing.T) {
	r := NewRegistry()
	r.Register(&MockTool{name: "z_tool"})
	r.Register(&MockTool{name: "a_tool"})
	r.Register(&MockTool{name: "m_tool"})
	r.SetExcludedTools([]string{"m_tool"})

	defs := r.GetAPIToolDefinitions()
	got := make([]string, 0, len(defs))
	for _, def := range defs {
		got = append(got, def.Name)
		if def.Description == "" {
			t.Fatalf("GetAPIToolDefinitions() description for %q is empty", def.Name)
		}
		if !reflect.DeepEqual(def.Parameters, map[string]interface{}{"type": "object"}) {
			t.Fatalf("GetAPIToolDefinitions() parameters for %q = %#v, want mock object schema", def.Name, def.Parameters)
		}
	}
	want := []string{"a_tool", "z_tool"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetAPIToolDefinitions() names = %v, want %v", got, want)
	}
}

func TestRegistry_ExecuteDetailedWithContext_ClonesStructuredObservations(t *testing.T) {
	observation := &RuntimeObservation{
		TouchedFiles: []ObservationPath{{Path: "pkg/source.go", ResolvedPath: "/repo/pkg/source.go"}},
	}
	groupObservation := &RuntimeObservation{
		Evidence: []ObservationEvidence{{
			Path:      "pkg/source.go",
			StartLine: 10,
			Excerpt:   "target()",
		}},
	}
	r := NewRegistry()
	r.Register(&StructuredMockTool{
		name: "structured_tool",
		result: ToolRunResult{
			Output:      "ok",
			Observation: observation,
			ObservationGroups: map[string]*RuntimeObservation{
				"pattern": groupObservation,
			},
		},
	})

	result := r.ExecuteDetailedWithContext(ExecutionContext{}, &ToolCall{Tool: "structured_tool"})

	if result.Observation == nil || result.Observation == observation {
		t.Fatalf("ExecuteDetailedWithContext() observation pointer = %#v, want cloned observation", result.Observation)
	}
	if result.ObservationGroups["pattern"] == nil || result.ObservationGroups["pattern"] == groupObservation {
		t.Fatalf("ExecuteDetailedWithContext() observation group pointer = %#v, want cloned observation", result.ObservationGroups["pattern"])
	}

	observation.TouchedFiles[0].Path = "mutated.go"
	groupObservation.Evidence[0].Excerpt = "mutated()"

	if got := result.Observation.TouchedFiles[0].Path; got != "pkg/source.go" {
		t.Fatalf("result observation path = %q, want original path", got)
	}
	if got := result.ObservationGroups["pattern"].Evidence[0].Excerpt; got != "target()" {
		t.Fatalf("result observation group excerpt = %q, want original excerpt", got)
	}
}
