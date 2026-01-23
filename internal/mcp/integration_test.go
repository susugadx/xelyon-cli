package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPToolWrapper_Name(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		toolName   string
		expected   string
	}{
		{
			name:       "simple names",
			serverName: "filesystem",
			toolName:   "read_file",
			expected:   "mcp_filesystem_read_file",
		},
		{
			name:       "names with hyphens",
			serverName: "my-server",
			toolName:   "my-tool",
			expected:   "mcp_my_server_my_tool",
		},
		{
			name:       "names with special characters",
			serverName: "server@1.0",
			toolName:   "tool!",
			expected:   "mcp_server_1_0_tool_",
		},
		{
			name:       "names with spaces",
			serverName: "my server",
			toolName:   "my tool",
			expected:   "mcp_my_server_my_tool",
		},
		{
			name:       "names with slashes",
			serverName: "server/path",
			toolName:   "tool/action",
			expected:   "mcp_server_path_tool_action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := &MCPToolWrapper{
				serverName: tt.serverName,
				toolName:   tt.toolName,
			}

			result := wrapper.Name()
			if result != tt.expected {
				t.Errorf("Name() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestMCPToolWrapper_FormatResult(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName: "test_tool",
	}

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "empty result",
			input:    "",
			contains: "Tool executed successfully (no output)",
		},
		{
			name:     "short result",
			input:    "success",
			contains: "success",
		},
		{
			name:     "multiline result (under limit)",
			input:    "line1\nline2\nline3",
			contains: "line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapper.formatResult(tt.input)
			if tt.contains != "" && result != tt.contains {
				t.Errorf("formatResult() = %q, want %q", result, tt.contains)
			}
		})
	}
}

func TestMCPToolWrapper_FormatResult_NoTruncation(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName: "test_tool",
	}

	// 25行の出力を生成
	// NOTE: MCP出力の切り詰めはtoken_guard.goで一元管理するため、ここでは行わない
	var lines []string
	for i := 1; i <= 25; i++ {
		lines = append(lines, "line")
	}
	input := ""
	for i, line := range lines {
		if i > 0 {
			input += "\n"
		}
		input += line
	}

	result := wrapper.formatResult(input)

	// truncationされていないこと（全行が含まれる）
	if result != input {
		t.Errorf("Expected result to equal input (no truncation), got different output")
	}

	// "output truncated"が含まれていないこと
	if strings.Contains(result, "output truncated") {
		t.Errorf("Expected result to NOT contain 'output truncated'")
	}
}

func TestMCPToolWrapper_ValidateArgs_EmptySchema(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: nil,
	}

	args := map[string]string{
		"param1": "value1",
	}

	err := wrapper.validateArgs(args)
	if err != nil {
		t.Errorf("validateArgs with empty schema should not error, got: %v", err)
	}
}

func TestMCPToolWrapper_ValidateArgs_NullSchema(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: json.RawMessage("null"),
	}

	args := map[string]string{
		"param1": "value1",
	}

	err := wrapper.validateArgs(args)
	if err != nil {
		t.Errorf("validateArgs with null schema should not error, got: %v", err)
	}
}

func TestMCPToolWrapper_ValidateArgs_InvalidJSON(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: json.RawMessage("{invalid json"),
	}

	args := map[string]string{
		"param1": "value1",
	}

	// 不正なJSONの場合は警告のみで成功するべき
	err := wrapper.validateArgs(args)
	if err != nil {
		t.Errorf("validateArgs with invalid JSON should warn but not error, got: %v", err)
	}
}

func TestMCPToolWrapper_ValidateArgs_ValidSchema(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":     "string",
				"required": true,
			},
			"optional": map[string]any{
				"type": "string",
			},
		},
	}

	schemaBytes, _ := json.Marshal(schema)
	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: schemaBytes,
	}

	tests := []struct {
		name    string
		args    map[string]string
		wantErr bool
	}{
		{
			name: "valid args with required param",
			args: map[string]string{
				"path": "/test/path",
			},
			wantErr: false,
		},
		{
			name: "valid args with all params",
			args: map[string]string{
				"path":     "/test/path",
				"optional": "value",
			},
			wantErr: false,
		},
		{
			name:    "missing required param",
			args:    map[string]string{},
			wantErr: true, // Should error because 'path' is required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapper.validateArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_RegisterToToolRegistry(t *testing.T) {
	manager := NewManager()

	// モックツールを追加
	manager.tools = []MCPTool{
		{
			ServerName:  "test-server",
			Name:        "test-tool",
			Description: "A test tool",
			InputSchema: json.RawMessage(`{"type": "object"}`),
		},
	}

	// RegisterToToolRegistry を直接テストするのは難しいので、
	// ツールが正しくフォーマットされることを確認
	wrapper := &MCPToolWrapper{
		manager:     manager,
		serverName:  "test-server",
		toolName:    "test-tool",
		desc:        "A test tool",
		inputSchema: json.RawMessage(`{"type": "object"}`),
	}

	expectedName := "mcp_test_server_test_tool"
	if wrapper.Name() != expectedName {
		t.Errorf("Wrapper name = %q, want %q", wrapper.Name(), expectedName)
	}
}

func TestMCPToolWrapper_Run_ValidationError(t *testing.T) {
	manager := NewManager()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"required_param": map[string]any{
				"type":     "string",
				"required": true,
			},
		},
	}
	schemaBytes, _ := json.Marshal(schema)

	wrapper := &MCPToolWrapper{
		manager:     manager,
		serverName:  "test-server",
		toolName:    "test-tool",
		desc:        "A test tool",
		inputSchema: schemaBytes,
	}

	// 必須パラメータなしで実行
	result, change, err := wrapper.Run(map[string]string{})

	if err == nil {
		t.Error("Run() should return error for missing required parameter")
	}

	if change != nil {
		t.Error("Run() should not return FileChange on validation error")
	}

	if !strings.Contains(result, "Validation Error") {
		t.Errorf("Result should contain 'Validation Error', got: %s", result)
	}
}

func TestMCPToolWrapper_Run_CallToolError(t *testing.T) {
	manager := NewManager()

	wrapper := &MCPToolWrapper{
		manager:     manager,
		serverName:  "nonexistent-server",
		toolName:    "test-tool",
		desc:        "A test tool",
		inputSchema: nil,
	}

	// 接続していないサーバーに対して実行
	result, change, err := wrapper.Run(map[string]string{})

	if err == nil {
		t.Error("Run() should return error for non-connected server")
	}

	if change != nil {
		t.Error("Run() should not return FileChange on error")
	}

	if !strings.Contains(result, "Error:") {
		t.Errorf("Result should contain 'Error:', got: %s", result)
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_EmptySchema(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: nil,
	}

	args := map[string]string{
		"param1": "value1",
		"param2": "123",
	}

	result := wrapper.convertArgsWithSchema(args)

	if result["param1"] != "value1" {
		t.Errorf("Expected param1='value1', got %v", result["param1"])
	}
	if result["param2"] != "123" {
		t.Errorf("Expected param2='123' (string), got %v", result["param2"])
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_NullSchema(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: json.RawMessage("null"),
	}

	args := map[string]string{
		"param1": "value1",
	}

	result := wrapper.convertArgsWithSchema(args)

	if result["param1"] != "value1" {
		t.Errorf("Expected param1='value1', got %v", result["param1"])
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_InvalidJSON(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: json.RawMessage("{invalid json"),
	}

	args := map[string]string{
		"param1": "value1",
	}

	result := wrapper.convertArgsWithSchema(args)

	// Invalid JSON should fall back to string values
	if result["param1"] != "value1" {
		t.Errorf("Expected param1='value1', got %v", result["param1"])
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_IntegerType(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{
				"type": "integer",
			},
		},
	}
	schemaBytes, _ := json.Marshal(schema)

	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: schemaBytes,
	}

	args := map[string]string{
		"count": "42",
	}

	result := wrapper.convertArgsWithSchema(args)

	if val, ok := result["count"].(int64); !ok || val != 42 {
		t.Errorf("Expected count=42 (int64), got %T %v", result["count"], result["count"])
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_NumberType(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"price": map[string]any{
				"type": "number",
			},
		},
	}
	schemaBytes, _ := json.Marshal(schema)

	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: schemaBytes,
	}

	args := map[string]string{
		"price": "19.99",
	}

	result := wrapper.convertArgsWithSchema(args)

	if val, ok := result["price"].(float64); !ok || val != 19.99 {
		t.Errorf("Expected price=19.99 (float64), got %T %v", result["price"], result["price"])
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_BooleanType(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"enabled": map[string]any{
				"type": "boolean",
			},
		},
	}
	schemaBytes, _ := json.Marshal(schema)

	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: schemaBytes,
	}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"1", "1", true},
		{"0", "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]string{
				"enabled": tt.input,
			}
			result := wrapper.convertArgsWithSchema(args)

			if val, ok := result["enabled"].(bool); !ok || val != tt.want {
				t.Errorf("Expected enabled=%v (bool), got %T %v", tt.want, result["enabled"], result["enabled"])
			}
		})
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_StringType(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
		},
	}
	schemaBytes, _ := json.Marshal(schema)

	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: schemaBytes,
	}

	args := map[string]string{
		"name": "test-value",
	}

	result := wrapper.convertArgsWithSchema(args)

	// String type should remain as string
	if result["name"] != "test-value" {
		t.Errorf("Expected name='test-value', got %v", result["name"])
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_MixedTypes(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
			"count": map[string]any{
				"type": "integer",
			},
			"enabled": map[string]any{
				"type": "boolean",
			},
		},
	}
	schemaBytes, _ := json.Marshal(schema)

	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: schemaBytes,
	}

	args := map[string]string{
		"name":    "test",
		"count":   "10",
		"enabled": "true",
	}

	result := wrapper.convertArgsWithSchema(args)

	if result["name"] != "test" {
		t.Errorf("Expected name='test', got %v", result["name"])
	}
	if val, ok := result["count"].(int64); !ok || val != 10 {
		t.Errorf("Expected count=10 (int64), got %T %v", result["count"], result["count"])
	}
	if val, ok := result["enabled"].(bool); !ok || val != true {
		t.Errorf("Expected enabled=true (bool), got %T %v", result["enabled"], result["enabled"])
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_InvalidValue(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{
				"type": "integer",
			},
		},
	}
	schemaBytes, _ := json.Marshal(schema)

	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: schemaBytes,
	}

	args := map[string]string{
		"count": "not-a-number",
	}

	result := wrapper.convertArgsWithSchema(args)

	// Invalid value should remain as string
	if result["count"] != "not-a-number" {
		t.Errorf("Expected count='not-a-number' (string fallback), got %T %v", result["count"], result["count"])
	}
}

func TestMCPToolWrapper_ConvertArgsWithSchema_UnknownProperty(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"known": map[string]any{
				"type": "string",
			},
		},
	}
	schemaBytes, _ := json.Marshal(schema)

	wrapper := &MCPToolWrapper{
		toolName:    "test_tool",
		inputSchema: schemaBytes,
	}

	args := map[string]string{
		"known":   "value",
		"unknown": "123",
	}

	result := wrapper.convertArgsWithSchema(args)

	// Unknown property should remain as string
	if result["unknown"] != "123" {
		t.Errorf("Expected unknown='123' (string), got %v", result["unknown"])
	}
}
