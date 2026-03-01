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
func (m *MockTool) Run(args map[string]string) (string, *FileChange, error) {
	return "Success", nil, nil
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
	r.Register(&MockTool{name: "create_plan"})
	r.Register(&MockTool{name: "update_plan"})
	r.Register(&MockTool{name: "ask_user_question"})

	// 除外なし: 全4ツール
	defs := r.GetToolDefinitions()
	if len(defs) != 4 {
		t.Errorf("GetToolDefinitions() without exclusion = %d, want 4", len(defs))
	}

	// planning 系を除外
	r.SetExcludedTools([]string{"create_plan", "update_plan", "ask_user_question"})
	defs = r.GetToolDefinitions()
	if len(defs) != 1 {
		t.Errorf("GetToolDefinitions() with exclusion = %d, want 1", len(defs))
	}
	if defs[0].Name != "read_file" {
		t.Errorf("GetToolDefinitions()[0].Name = %s, want read_file", defs[0].Name)
	}

	// HasTool は除外に関係なく true
	if !r.HasTool("create_plan") {
		t.Error("HasTool(create_plan) should be true even when excluded")
	}

	// ClearExcludedTools で復帰
	r.ClearExcludedTools()
	defs = r.GetToolDefinitions()
	if len(defs) != 4 {
		t.Errorf("GetToolDefinitions() after clear = %d, want 4", len(defs))
	}
}
