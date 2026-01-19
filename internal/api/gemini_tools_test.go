package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetGeminiToolDefinitions(t *testing.T) {
	tools := GetGeminiToolDefinitions()

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool config, got %d", len(tools))
	}

	declarations := tools[0].FunctionDeclarations
	if len(declarations) == 0 {
		t.Error("Expected at least one function declaration")
	}

	// 35ツールが定義されていることを確認
	expectedCount := 35
	if len(declarations) != expectedCount {
		t.Errorf("Expected %d tool definitions, got %d", expectedCount, len(declarations))
	}
}

func TestToolDefinitionsHaveRequiredFields(t *testing.T) {
	tools := GetGeminiToolDefinitions()
	declarations := tools[0].FunctionDeclarations

	for _, decl := range declarations {
		if decl.Name == "" {
			t.Error("Found tool definition with empty name")
		}
		if decl.Description == "" {
			t.Errorf("Tool %s has empty description", decl.Name)
		}
	}
}

func TestConvertFunctionCallToToolJSON(t *testing.T) {
	testCases := []struct {
		name     string
		input    *GeminiFunctionCall
		expected map[string]any
	}{
		{
			name: "read_file with path",
			input: &GeminiFunctionCall{
				Name: "read_file",
				Args: map[string]any{"path": "/path/to/file"},
			},
			expected: map[string]any{
				"tool": "read_file",
				"args": map[string]any{"path": "/path/to/file"},
			},
		},
		{
			name: "str_replace with multiple args",
			input: &GeminiFunctionCall{
				Name: "str_replace",
				Args: map[string]any{
					"path":    "/path/to/file",
					"old_str": "foo",
					"new_str": "bar",
				},
			},
			expected: map[string]any{
				"tool": "str_replace",
				"args": map[string]any{
					"path":    "/path/to/file",
					"old_str": "foo",
					"new_str": "bar",
				},
			},
		},
		{
			name: "git_status with no args",
			input: &GeminiFunctionCall{
				Name: "git_status",
				Args: map[string]any{},
			},
			expected: map[string]any{
				"tool": "git_status",
				"args": map[string]any{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertFunctionCallToToolJSON(tc.input)

			// JSONとしてパース
			var parsed map[string]any
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Fatalf("Failed to parse result as JSON: %v", err)
			}

			// tool名を確認
			if parsed["tool"] != tc.expected["tool"] {
				t.Errorf("Expected tool=%v, got %v", tc.expected["tool"], parsed["tool"])
			}

			// args を確認
			expectedArgs := tc.expected["args"].(map[string]any)
			parsedArgs := parsed["args"].(map[string]any)

			for key, expectedVal := range expectedArgs {
				if parsedArgs[key] != expectedVal {
					t.Errorf("Expected args[%s]=%v, got %v", key, expectedVal, parsedArgs[key])
				}
			}
		})
	}
}

func TestConvertFunctionCallToToolJSON_KeyOrder(t *testing.T) {
	// JSON出力で "tool" が "args" より前に来ることを確認
	// ParseToolCalls は {"tool" パターンで検索するため重要
	fc := &GeminiFunctionCall{
		Name: "write_file",
		Args: map[string]any{"path": "test.txt", "content": "hello"},
	}

	result := convertFunctionCallToToolJSON(fc)

	// "tool" が "args" より前に来ていることを確認
	toolIdx := strings.Index(result, `"tool"`)
	argsIdx := strings.Index(result, `"args"`)

	if toolIdx == -1 {
		t.Fatal("\"tool\" key not found in JSON output")
	}
	if argsIdx == -1 {
		t.Fatal("\"args\" key not found in JSON output")
	}
	if toolIdx >= argsIdx {
		t.Errorf("\"tool\" should come before \"args\" in JSON output. Got: %s", result)
	}

	// {"tool" で始まることを確認（ParseToolCallsのパターンマッチに必要）
	if !strings.HasPrefix(result, `{"tool"`) {
		t.Errorf("JSON should start with {\"tool\", got: %s", result)
	}
}

func TestAllBuiltinToolsHaveDefinitions(t *testing.T) {
	// 期待されるツール名のリスト
	expectedTools := []string{
		// File Operations
		"read_file", "write_file", "str_replace", "append_file", "prepend_file",
		"insert_after", "insert_before", "copy_file", "move_file", "delete_file",
		"delete_lines", "list_dir", "create_dir", "restore_backup", "list_backups",
		// Git Operations
		"git_status", "git_diff", "git_log", "git_add", "git_commit",
		"git_push", "git_branch", "git_checkout", "git_stash",
		// Search Operations
		"search_code", "search_file", "web_search", "ast_grep", "grep_replace",
		// Development Operations
		"run_test", "format", "lint", "diff_files", "http_request", "bash",
	}

	definedNames := GetToolDefinitionNames()
	definedMap := make(map[string]bool)
	for _, name := range definedNames {
		definedMap[name] = true
	}

	for _, expected := range expectedTools {
		if !definedMap[expected] {
			t.Errorf("Missing tool definition for: %s", expected)
		}
	}
}

func TestGeminiFunctionCallStructure(t *testing.T) {
	// JSON からパースできることを確認
	jsonData := `{"name": "read_file", "args": {"path": "/test/file.txt", "start_line": 1}}`

	var fc GeminiFunctionCall
	if err := json.Unmarshal([]byte(jsonData), &fc); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if fc.Name != "read_file" {
		t.Errorf("Expected name=read_file, got %s", fc.Name)
	}

	if fc.Args["path"] != "/test/file.txt" {
		t.Errorf("Expected path=/test/file.txt, got %v", fc.Args["path"])
	}

	// start_line は数値として解析される
	if fc.Args["start_line"] != float64(1) {
		t.Errorf("Expected start_line=1, got %v (type: %T)", fc.Args["start_line"], fc.Args["start_line"])
	}
}

func TestGeminiToolConfigMarshal(t *testing.T) {
	tools := GetGeminiToolDefinitions()

	// JSON にシリアライズできることを確認
	jsonData, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("Failed to marshal tool definitions: %v", err)
	}

	// 空でないことを確認
	if len(jsonData) < 100 {
		t.Error("Tool definitions JSON seems too short")
	}

	// 再パースできることを確認
	var parsed []GeminiToolConfig
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal tool definitions: %v", err)
	}

	if len(parsed) != 1 {
		t.Errorf("Expected 1 tool config after unmarshal, got %d", len(parsed))
	}
}
