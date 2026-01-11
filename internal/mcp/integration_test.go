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

func TestMCPToolWrapper_FormatResult_Truncation(t *testing.T) {
	wrapper := &MCPToolWrapper{
		toolName: "test_tool",
	}

	// 25行の出力を生成（limitは20行）
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

	// "output truncated"が含まれているべき
	if !strings.Contains(result, "output truncated") {
		t.Errorf("Expected result to contain 'output truncated', got: %s", result)
	}

	// "5 more lines"が含まれているべき (25 - 20 = 5)
	if !strings.Contains(result, "5 more lines") {
		t.Errorf("Expected result to mention '5 more lines', got: %s", result)
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
