package openai

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// TestConvertHistoryToResponsesInput は History を Responses API 形式に変換するテスト
func TestConvertHistoryToResponsesInput(t *testing.T) {
	tests := []struct {
		name    string
		history []api.Message
		want    []struct {
			Type      string
			Role      string
			Content   string
			CallID    string
			Name      string
			Arguments string
			Output    string
		}
	}{
		{
			name:    "empty history",
			history: []api.Message{},
			want:    nil,
		},
		{
			name: "single user message",
			history: []api.Message{
				{Role: "user", Content: "Hello"},
			},
			want: []struct {
				Type      string
				Role      string
				Content   string
				CallID    string
				Name      string
				Arguments string
				Output    string
			}{
				{Type: "message", Role: "user", Content: "Hello"},
			},
		},
		{
			name: "assistant message with ToolCalls",
			history: []api.Message{
				{
					Role:    "assistant",
					Content: "",
					ToolCalls: []api.OpenAIToolCall{
						{
							ID:   "call_abc123",
							Type: "function",
							Function: api.OpenAIToolCallFunction{
								Name:      "read_file",
								Arguments: `{"path":"/test.txt"}`,
							},
						},
					},
				},
			},
			want: []struct {
				Type      string
				Role      string
				Content   string
				CallID    string
				Name      string
				Arguments string
				Output    string
			}{
				{Type: "function_call", CallID: "call_abc123", Name: "read_file", Arguments: `{"path":"/test.txt"}`},
			},
		},
		{
			name: "tool message (function_call_output)",
			history: []api.Message{
				{
					Role:       "tool",
					Content:    "file content here",
					ToolCallID: "call_abc123",
				},
			},
			want: []struct {
				Type      string
				Role      string
				Content   string
				CallID    string
				Name      string
				Arguments string
				Output    string
			}{
				{Type: "function_call_output", CallID: "call_abc123", Output: "file content here"},
			},
		},
		{
			name: "multi-turn conversation with function calling",
			history: []api.Message{
				{Role: "user", Content: "Read the config file"},
				{
					Role: "assistant",
					ToolCalls: []api.OpenAIToolCall{
						{
							ID:   "call_001",
							Type: "function",
							Function: api.OpenAIToolCallFunction{
								Name:      "read_file",
								Arguments: `{"path":"config.yaml"}`,
							},
						},
					},
				},
				{Role: "tool", ToolCallID: "call_001", Content: "key: value"},
				{Role: "assistant", Content: "The config file contains key: value"},
				{Role: "user", Content: "Thanks!"},
			},
			want: []struct {
				Type      string
				Role      string
				Content   string
				CallID    string
				Name      string
				Arguments string
				Output    string
			}{
				{Type: "message", Role: "user", Content: "Read the config file"},
				{Type: "function_call", CallID: "call_001", Name: "read_file", Arguments: `{"path":"config.yaml"}`},
				{Type: "function_call_output", CallID: "call_001", Output: "key: value"},
				{Type: "message", Role: "assistant", Content: "The config file contains key: value"},
				{Type: "message", Role: "user", Content: "Thanks!"},
			},
		},
		{
			name: "multiple tool calls in single assistant message",
			history: []api.Message{
				{
					Role: "assistant",
					ToolCalls: []api.OpenAIToolCall{
						{
							ID:   "call_001",
							Type: "function",
							Function: api.OpenAIToolCallFunction{
								Name:      "read_file",
								Arguments: `{"path":"file1.txt"}`,
							},
						},
						{
							ID:   "call_002",
							Type: "function",
							Function: api.OpenAIToolCallFunction{
								Name:      "read_file",
								Arguments: `{"path":"file2.txt"}`,
							},
						},
					},
				},
			},
			want: []struct {
				Type      string
				Role      string
				Content   string
				CallID    string
				Name      string
				Arguments string
				Output    string
			}{
				{Type: "function_call", CallID: "call_001", Name: "read_file", Arguments: `{"path":"file1.txt"}`},
				{Type: "function_call", CallID: "call_002", Name: "read_file", Arguments: `{"path":"file2.txt"}`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertHistoryToResponsesInput(tt.history)

			if len(tt.want) == 0 {
				if len(got) != 0 {
					t.Errorf("convertHistoryToResponsesInput() returned %d items, want 0", len(got))
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("convertHistoryToResponsesInput() returned %d items, want %d", len(got), len(tt.want))
			}

			for i, want := range tt.want {
				if got[i].Type != want.Type {
					t.Errorf("Item[%d].Type = %q, want %q", i, got[i].Type, want.Type)
				}
				if got[i].Role != want.Role {
					t.Errorf("Item[%d].Role = %q, want %q", i, got[i].Role, want.Role)
				}
				// Content は string または []InputContentPart なので型アサーション
				if content, ok := got[i].Content.(string); ok {
					if content != want.Content {
						t.Errorf("Item[%d].Content = %q, want %q", i, content, want.Content)
					}
				} else if want.Content != "" {
					t.Errorf("Item[%d].Content type is not string", i)
				}
				if got[i].CallID != want.CallID {
					t.Errorf("Item[%d].CallID = %q, want %q", i, got[i].CallID, want.CallID)
				}
				if got[i].Name != want.Name {
					t.Errorf("Item[%d].Name = %q, want %q", i, got[i].Name, want.Name)
				}
				if got[i].Arguments != want.Arguments {
					t.Errorf("Item[%d].Arguments = %q, want %q", i, got[i].Arguments, want.Arguments)
				}
				if got[i].Output != want.Output {
					t.Errorf("Item[%d].Output = %q, want %q", i, got[i].Output, want.Output)
				}
			}
		})
	}
}

// TestGetResponsesToolDefinitions は Responses API 用ツール定義のテスト
func TestGetResponsesToolDefinitions(t *testing.T) {
	// MCPツールなし
	tools := GetResponsesToolDefinitions(nil)

	// ツール数の確認（toolDefinitions に定義されている数）
	expectedCount := len(toolDefinitions)
	if len(tools) != expectedCount {
		t.Errorf("GetResponsesToolDefinitions() returned %d tools, want %d", len(tools), expectedCount)
	}

	// 各ツールの形式を確認
	for i, tool := range tools {
		if tool.Type != "function" {
			t.Errorf("Tool[%d].Type = %q, want 'function'", i, tool.Type)
		}
		if tool.Name == "" {
			t.Errorf("Tool[%d].Name is empty", i)
		}
		if tool.Description == "" {
			t.Errorf("Tool[%d] (%s).Description is empty", i, tool.Name)
		}
		if tool.Parameters == nil {
			t.Errorf("Tool[%d] (%s).Parameters is nil", i, tool.Name)
		}
	}

	// 特定のツールが存在することを確認
	requiredTools := []string{"read_file", "write_file", "str_replace", "bash", "search_code"}
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, required := range requiredTools {
		if !toolNames[required] {
			t.Errorf("Required tool %q not found in tool definitions", required)
		}
	}
}

// TestGetResponsesToolDefinitions_WithMCPTools は MCP ツール込みのテスト
func TestGetResponsesToolDefinitions_WithMCPTools(t *testing.T) {
	mcpTools := []api.OpenAIToolFunction{
		{
			Name:        "mcp_custom_tool",
			Description: "A custom MCP tool",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	tools := GetResponsesToolDefinitions(mcpTools)

	expectedCount := len(toolDefinitions) + 1
	if len(tools) != expectedCount {
		t.Errorf("GetResponsesToolDefinitions() with MCP returned %d tools, want %d", len(tools), expectedCount)
	}

	// MCP ツールが含まれていることを確認
	found := false
	for _, tool := range tools {
		if tool.Name == "mcp_custom_tool" {
			found = true
			if tool.Type != "function" {
				t.Errorf("MCP tool Type = %q, want 'function'", tool.Type)
			}
			break
		}
	}
	if !found {
		t.Error("MCP tool 'mcp_custom_tool' not found in tool definitions")
	}
}

// TestResponsesToolFormat は ResponsesTool の JSON 形式をテスト
func TestResponsesToolFormat(t *testing.T) {
	tools := GetResponsesToolDefinitions(nil)

	// read_file ツールを探す
	var readFileTool *ResponsesTool
	for i := range tools {
		if tools[i].Name == "read_file" {
			readFileTool = &tools[i]
			break
		}
	}

	if readFileTool == nil {
		t.Fatal("read_file tool not found")
	}

	// 形式を確認
	if readFileTool.Type != "function" {
		t.Errorf("read_file.Type = %q, want 'function'", readFileTool.Type)
	}
	if readFileTool.Description == "" {
		t.Error("read_file.Description is empty")
	}

	// Parameters の形式確認
	params := readFileTool.Parameters
	if params == nil {
		t.Fatal("read_file.Parameters is nil")
	}

	if params["type"] != "object" {
		t.Errorf("read_file.Parameters.type = %v, want 'object'", params["type"])
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("read_file.Parameters.properties is not a map")
	}

	if _, ok := props["path"]; !ok {
		t.Error("read_file.Parameters.properties.path not found")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("read_file.Parameters.required is not []string")
	}
	if len(required) == 0 || required[0] != "path" {
		t.Errorf("read_file.Parameters.required = %v, want ['path']", required)
	}
}

// TestConvertToolCallToToolJSON は OpenAI tool_call を内部形式に変換するテスト
func TestConvertToolCallToToolJSON(t *testing.T) {
	tests := []struct {
		name      string
		toolCall  *api.OpenAIToolCall
		wantID    string
		wantTool  string
		wantError bool
	}{
		{
			name: "valid tool call",
			toolCall: &api.OpenAIToolCall{
				ID:   "call_abc123",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"/test.txt"}`,
				},
			},
			wantID:   "call_abc123",
			wantTool: "read_file",
		},
		{
			name: "empty arguments",
			toolCall: &api.OpenAIToolCall{
				ID:   "call_xyz789",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "run_test",
					Arguments: `{}`,
				},
			},
			wantID:   "call_xyz789",
			wantTool: "run_test",
		},
		{
			name: "invalid JSON arguments",
			toolCall: &api.OpenAIToolCall{
				ID:   "call_invalid",
				Type: "function",
				Function: api.OpenAIToolCallFunction{
					Name:      "test",
					Arguments: `{invalid json}`,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToolCallToToolJSON(tt.toolCall)

			if tt.wantError {
				if err == nil {
					t.Error("ConvertToolCallToToolJSON() should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("ConvertToolCallToToolJSON() error = %v", err)
			}

			// JSON をパースして確認
			if result == "" {
				t.Error("ConvertToolCallToToolJSON() returned empty string")
			}

			// ID が含まれていることを確認
			if tt.wantID != "" && !contains(result, tt.wantID) {
				t.Errorf("Result should contain ID %q: %s", tt.wantID, result)
			}

			// ツール名が含まれていることを確認
			if !contains(result, tt.wantTool) {
				t.Errorf("Result should contain tool %q: %s", tt.wantTool, result)
			}
		})
	}
}

// contains は文字列に部分文字列が含まれるかチェック
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestResponsesAPIInputFormat は Responses API の input 配列形式をテスト
func TestResponsesAPIInputFormat(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "Can you read a file?"},
		{
			Role: "assistant",
			ToolCalls: []api.OpenAIToolCall{
				{
					ID:   "call_file",
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"test.go"}`,
					},
				},
			},
		},
		{Role: "tool", ToolCallID: "call_file", Content: "package main"},
		{Role: "assistant", Content: "The file contains a Go package."},
	}

	input := convertHistoryToResponsesInput(history)

	// 期待される順序と type を確認
	expectedTypes := []string{
		"message",              // user: Hello
		"message",              // assistant: Hi there!
		"message",              // user: Can you read a file?
		"function_call",        // assistant tool call
		"function_call_output", // tool result
		"message",              // assistant: The file contains...
	}

	if len(input) != len(expectedTypes) {
		t.Fatalf("Input length = %d, want %d", len(input), len(expectedTypes))
	}

	for i, expectedType := range expectedTypes {
		if input[i].Type != expectedType {
			t.Errorf("Input[%d].Type = %q, want %q", i, input[i].Type, expectedType)
		}
	}

	// function_call の詳細確認
	if input[3].CallID != "call_file" {
		t.Errorf("function_call CallID = %q, want 'call_file'", input[3].CallID)
	}
	if input[3].Name != "read_file" {
		t.Errorf("function_call Name = %q, want 'read_file'", input[3].Name)
	}

	// function_call_output の詳細確認
	if input[4].CallID != "call_file" {
		t.Errorf("function_call_output CallID = %q, want 'call_file'", input[4].CallID)
	}
	if input[4].Output != "package main" {
		t.Errorf("function_call_output Output = %q, want 'package main'", input[4].Output)
	}
}

// TestResponsesAPI_FunctionCall_ArgumentsDeltaAccumulation_Note:
// function_call_arguments.delta の累積処理は実際の streaming でのみテスト可能
// (response.output_item.added → delta (複数) → done → response.completed)
// 単体テストの場合は、実装ロジックレビューで確認:
// - callID がない場合: len(functionCalls) == 1 なら累積
// - callID がある場合: 通常通り累積
// - done イベントで Arguments が空文字列なら累積値を保持
// - done イベントで Arguments があれば上書き
