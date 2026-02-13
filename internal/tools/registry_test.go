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
