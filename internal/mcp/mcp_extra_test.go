package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestManager_SetOutput(t *testing.T) {
	manager := NewManager()

	var buf bytes.Buffer
	manager.SetOutput(&buf)

	// out() が設定した writer を返すことを確認
	w := manager.out()
	if w != &buf {
		t.Fatal("out() should return the writer set by SetOutput")
	}

	// 実際に書き込めることを確認
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if buf.String() != "hello" {
		t.Errorf("expected %q, got %q", "hello", buf.String())
	}
}

func TestManager_SetOutput_Nil(t *testing.T) {
	manager := NewManager()

	// nil を設定
	manager.SetOutput(nil)

	// out() は io.Discard を返すべき
	w := manager.out()
	if w != io.Discard {
		t.Fatal("out() should return io.Discard when output is nil")
	}
}

func TestMCPToolWrapper_Description_Empty(t *testing.T) {
	wrapper := &MCPToolWrapper{
		serverName: "test-server",
		toolName:   "test-tool",
		desc:       "",
	}

	result := wrapper.Description()

	expected := "MCP tool: test-tool from server test-server"
	if result != expected {
		t.Errorf("Description() = %q, want %q", result, expected)
	}
}

func TestMCPToolWrapper_Description_Provided(t *testing.T) {
	wrapper := &MCPToolWrapper{
		serverName: "test-server",
		toolName:   "test-tool",
		desc:       "Reads a file from the filesystem",
	}

	result := wrapper.Description()

	if result != "Reads a file from the filesystem" {
		t.Errorf("Description() = %q, want %q", result, "Reads a file from the filesystem")
	}
}

func TestMCPToolWrapper_Parameters_ValidSchema(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path",
			},
			"encoding": map[string]any{
				"type":        "string",
				"description": "File encoding",
			},
		},
		"required": []any{"path"},
	}
	schemaBytes, _ := json.Marshal(schema)

	wrapper := &MCPToolWrapper{
		inputSchema: schemaBytes,
	}

	params := wrapper.Parameters()

	// type フィールドが保持されていること
	if params["type"] != "object" {
		t.Errorf("expected type=object, got %v", params["type"])
	}

	// properties が存在すること
	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties to be map, got %T", params["properties"])
	}

	if _, ok := props["path"]; !ok {
		t.Error("expected 'path' property")
	}
	if _, ok := props["encoding"]; !ok {
		t.Error("expected 'encoding' property")
	}

	// required が保持されていること
	req, ok := params["required"].([]any)
	if !ok {
		t.Fatalf("expected required to be []any, got %T", params["required"])
	}
	if len(req) != 1 || req[0] != "path" {
		t.Errorf("expected required=[path], got %v", req)
	}
}

func TestMCPToolWrapper_Parameters_EmptySchema(t *testing.T) {
	tests := []struct {
		name        string
		inputSchema json.RawMessage
	}{
		{"nil schema", nil},
		{"empty bytes", json.RawMessage{}},
		{"null string", json.RawMessage("null")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := &MCPToolWrapper{
				inputSchema: tt.inputSchema,
			}

			params := wrapper.Parameters()

			if params["type"] != "object" {
				t.Errorf("expected type=object, got %v", params["type"])
			}

			props, ok := params["properties"].(map[string]any)
			if !ok {
				t.Fatalf("expected properties to be map, got %T", params["properties"])
			}
			if len(props) != 0 {
				t.Errorf("expected empty properties, got %v", props)
			}

			if params["additionalProperties"] != false {
				t.Errorf("expected additionalProperties=false, got %v", params["additionalProperties"])
			}
		})
	}
}

func TestMCPToolWrapper_Parameters_InvalidJSON(t *testing.T) {
	wrapper := &MCPToolWrapper{
		inputSchema: json.RawMessage("{not valid json!!"),
	}

	params := wrapper.Parameters()

	// 不正な JSON の場合はデフォルトの空スキーマが返るべき
	if params["type"] != "object" {
		t.Errorf("expected type=object, got %v", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties to be map, got %T", params["properties"])
	}
	if len(props) != 0 {
		t.Errorf("expected empty properties, got %v", props)
	}

	if params["additionalProperties"] != false {
		t.Errorf("expected additionalProperties=false, got %v", params["additionalProperties"])
	}
}

func TestMCPToolWrapper_FormatResult_Empty(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName: "test_tool",
	}

	result := wrapper.formatResult("")

	expected := "Tool executed successfully (no output)"
	if result != expected {
		t.Errorf("formatResult(\"\") = %q, want %q", result, expected)
	}
}

func TestMCPToolWrapper_FormatResult_NonEmpty(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName: "test_tool",
	}

	input := "file contents here\nline 2\nline 3"
	result := wrapper.formatResult(input)

	if result != input {
		t.Errorf("formatResult() = %q, want %q", result, input)
	}
}

func TestRegisterToToolRegistry_EmptyTools(t *testing.T) {
	manager := NewManager()
	// tools は空のスライス（NewManager のデフォルト）

	registry := tools.NewRegistry()

	// パニックしないことを確認
	manager.RegisterToToolRegistry(registry)

	// 何も登録されていないことを確認
	defs := registry.GetToolDefinitions()
	if len(defs) != 0 {
		t.Errorf("expected 0 tool definitions, got %d", len(defs))
	}
}

func TestManager_HealthStatus_NoSessions(t *testing.T) {
	manager := NewManager()

	status := manager.HealthStatus()

	if status == nil {
		t.Fatal("HealthStatus() should return non-nil map")
	}
	if len(status) != 0 {
		t.Errorf("expected empty map, got %d entries", len(status))
	}
}

func TestRegisterToToolRegistry_MultipleTools(t *testing.T) {
	manager := NewManager()
	manager.tools = []MCPTool{
		{
			ServerName:  "server-a",
			Name:        "tool-one",
			Description: "First tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			ServerName:  "server-a",
			Name:        "tool-two",
			Description: "Second tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}}}`),
		},
		{
			ServerName:  "server-b",
			Name:        "tool-three",
			Description: "",
			InputSchema: nil,
		},
	}

	registry := tools.NewRegistry()
	manager.RegisterToToolRegistry(registry)

	defs := registry.GetToolDefinitions()
	if len(defs) != 3 {
		t.Fatalf("expected 3 tool definitions, got %d", len(defs))
	}

	// 各ツールが正しい名前で登録されているか確認
	expectedNames := map[string]bool{
		"mcp_server_a_tool_one":   true,
		"mcp_server_a_tool_two":   true,
		"mcp_server_b_tool_three": true,
	}
	for _, def := range defs {
		if !expectedNames[def.Name] {
			t.Errorf("unexpected tool name %q", def.Name)
		}
	}
}

func TestRegisterToToolRegistry_ToolLookup(t *testing.T) {
	manager := NewManager()
	manager.tools = []MCPTool{
		{
			ServerName:  "myserver",
			Name:        "greet",
			Description: "Say hello",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
		},
	}

	registry := tools.NewRegistry()
	manager.RegisterToToolRegistry(registry)

	// GetTool で取得して Description と Parameters を確認
	tool := registry.GetTool("mcp_myserver_greet")
	if tool == nil {
		t.Fatal("expected tool to be registered as mcp_myserver_greet")
	}

	if tool.Description() != "Say hello" {
		t.Errorf("Description() = %q, want %q", tool.Description(), "Say hello")
	}

	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("expected type=object, got %v", params["type"])
	}
}
