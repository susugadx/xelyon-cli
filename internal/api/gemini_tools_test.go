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

// ===== MCP統合テスト =====

func TestConvertMCPToolToGeminiDeclaration(t *testing.T) {
	testCases := []struct {
		name         string
		toolName     string
		description  string
		inputSchema  json.RawMessage
		expectParams bool
		expectPath   bool
		expectReq    []string
	}{
		{
			name:        "full schema",
			toolName:    "mcp_github_get_issue",
			description: "Get issue details from GitHub",
			inputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"owner": {"type": "string", "description": "Repository owner"},
					"repo": {"type": "string", "description": "Repository name"},
					"issue_number": {"type": "integer", "description": "Issue number"}
				},
				"required": ["owner", "repo", "issue_number"]
			}`),
			expectParams: true,
			expectPath:   false,
			expectReq:    []string{"owner", "repo", "issue_number"},
		},
		{
			name:         "empty schema",
			toolName:     "mcp_server_simple",
			description:  "Simple tool with no params",
			inputSchema:  json.RawMessage(`{}`),
			expectParams: false,
			expectPath:   false,
			expectReq:    nil,
		},
		{
			name:         "null schema",
			toolName:     "mcp_server_null",
			description:  "Tool with null schema",
			inputSchema:  json.RawMessage(`null`),
			expectParams: false,
			expectPath:   false,
			expectReq:    nil,
		},
		{
			name:        "with enum",
			toolName:    "mcp_server_enum",
			description: "Tool with enum parameter",
			inputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"action": {"type": "string", "enum": ["start", "stop", "restart"]}
				}
			}`),
			expectParams: true,
			expectPath:   false,
			expectReq:    nil,
		},
		{
			name:        "with array type and items",
			toolName:    "mcp_server_array",
			description: "Tool with array parameter",
			inputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"files": {"type": "array", "items": {"type": "string"}, "description": "List of files"}
				},
				"required": ["files"]
			}`),
			expectParams: true,
			expectPath:   false,
			expectReq:    []string{"files"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decl := ConvertMCPToolToGeminiDeclaration(tc.toolName, tc.description, tc.inputSchema)

			if decl.Name != tc.toolName {
				t.Errorf("Expected name=%s, got %s", tc.toolName, decl.Name)
			}
			if decl.Description != tc.description {
				t.Errorf("Expected description=%s, got %s", tc.description, decl.Description)
			}

			if tc.expectParams {
				if decl.Parameters == nil {
					t.Fatal("Expected parameters to be set")
				}
				if decl.Parameters.Type != "object" {
					t.Errorf("Expected type=object, got %s", decl.Parameters.Type)
				}
			} else {
				// 空のスキーマの場合、Parametersはnilまたは空
				if decl.Parameters != nil && len(decl.Parameters.Properties) > 0 {
					t.Errorf("Expected no parameters, got %+v", decl.Parameters)
				}
			}

			if tc.expectReq != nil && decl.Parameters != nil {
				if len(decl.Parameters.Required) != len(tc.expectReq) {
					t.Errorf("Expected %d required params, got %d", len(tc.expectReq), len(decl.Parameters.Required))
				}
			}
		})
	}
}

func TestConvertMCPToolToGeminiDeclaration_InvalidJSON(t *testing.T) {
	// 不正なJSONでもエラーにならない（空のパラメータで続行）
	decl := ConvertMCPToolToGeminiDeclaration("mcp_test", "Test", json.RawMessage(`{invalid json`))

	if decl.Name != "mcp_test" {
		t.Errorf("Expected name=mcp_test, got %s", decl.Name)
	}
	if decl.Parameters != nil {
		t.Errorf("Expected nil parameters for invalid JSON, got %+v", decl.Parameters)
	}
}

func TestConvertMCPToolToGeminiDeclaration_ArrayWithItems(t *testing.T) {
	// array型のitemsが正しく変換されることを確認
	inputSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"files": {
				"type": "array",
				"items": {"type": "string", "description": "File path"},
				"description": "List of files to process"
			},
			"options": {
				"type": "array",
				"items": {"type": "integer"},
				"description": "Numeric options"
			}
		},
		"required": ["files"]
	}`)

	decl := ConvertMCPToolToGeminiDeclaration("mcp_test_array", "Test array tool", inputSchema)

	if decl.Parameters == nil {
		t.Fatal("Expected parameters to be set")
	}

	// filesプロパティの確認
	filesProp, ok := decl.Parameters.Properties["files"]
	if !ok {
		t.Fatal("Expected 'files' property to exist")
	}
	if filesProp.Type != "array" {
		t.Errorf("Expected files.type=array, got %s", filesProp.Type)
	}
	if filesProp.Items == nil {
		t.Fatal("Expected files.items to be set")
	}
	if filesProp.Items.Type != "string" {
		t.Errorf("Expected files.items.type=string, got %s", filesProp.Items.Type)
	}
	if filesProp.Items.Description != "File path" {
		t.Errorf("Expected files.items.description='File path', got '%s'", filesProp.Items.Description)
	}

	// optionsプロパティの確認
	optionsProp, ok := decl.Parameters.Properties["options"]
	if !ok {
		t.Fatal("Expected 'options' property to exist")
	}
	if optionsProp.Items == nil {
		t.Fatal("Expected options.items to be set")
	}
	if optionsProp.Items.Type != "integer" {
		t.Errorf("Expected options.items.type=integer, got %s", optionsProp.Items.Type)
	}

	// JSONシリアライズで items が含まれることを確認
	jsonData, err := json.Marshal(decl)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, `"items"`) {
		t.Errorf("Expected JSON to contain 'items', got: %s", jsonStr)
	}
}

func TestConvertMCPToolToGeminiDeclaration_ArrayWithoutItems(t *testing.T) {
	// array型でitemsが指定されていない場合、デフォルトでstring型を設定
	inputSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"description": "List of tags"
			}
		}
	}`)

	decl := ConvertMCPToolToGeminiDeclaration("mcp_test_no_items", "Test without items", inputSchema)

	if decl.Parameters == nil {
		t.Fatal("Expected parameters to be set")
	}

	tagsProp, ok := decl.Parameters.Properties["tags"]
	if !ok {
		t.Fatal("Expected 'tags' property to exist")
	}
	if tagsProp.Type != "array" {
		t.Errorf("Expected tags.type=array, got %s", tagsProp.Type)
	}
	// items が nil でないことを確認（デフォルト設定）
	if tagsProp.Items == nil {
		t.Fatal("Expected tags.items to be set (default)")
	}
	// デフォルトで string 型
	if tagsProp.Items.Type != "string" {
		t.Errorf("Expected tags.items.type=string (default), got %s", tagsProp.Items.Type)
	}
}

func TestConvertMCPToolToGeminiDeclaration_ArrayWithEmptyItemsType(t *testing.T) {
	// items.typeが空の場合、デフォルトでstring型を設定
	inputSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"values": {
				"type": "array",
				"items": {"description": "Some value"}
			}
		}
	}`)

	decl := ConvertMCPToolToGeminiDeclaration("mcp_test_empty_type", "Test with empty items type", inputSchema)

	valuesProp := decl.Parameters.Properties["values"]
	if valuesProp.Items == nil {
		t.Fatal("Expected values.items to be set")
	}
	// type が空の場合はデフォルトで string
	if valuesProp.Items.Type != "string" {
		t.Errorf("Expected values.items.type=string (default), got %s", valuesProp.Items.Type)
	}
}

func TestGetCombinedToolDefinitions(t *testing.T) {
	// MCPツールなしの場合
	t.Run("no MCP tools", func(t *testing.T) {
		tools := GetCombinedToolDefinitions(nil)

		if len(tools) != 1 {
			t.Fatalf("Expected 1 tool config, got %d", len(tools))
		}

		// 組み込みツール35個のみ
		expectedCount := 35
		if len(tools[0].FunctionDeclarations) != expectedCount {
			t.Errorf("Expected %d declarations, got %d", expectedCount, len(tools[0].FunctionDeclarations))
		}
	})

	// MCPツールありの場合
	t.Run("with MCP tools", func(t *testing.T) {
		mcpTools := []GeminiFunctionDeclaration{
			{Name: "mcp_github_get_issue", Description: "Get issue"},
			{Name: "mcp_github_list_repos", Description: "List repos"},
		}

		tools := GetCombinedToolDefinitions(mcpTools)

		if len(tools) != 1 {
			t.Fatalf("Expected 1 tool config, got %d", len(tools))
		}

		// 組み込み35 + MCP2
		expectedCount := 35 + 2
		if len(tools[0].FunctionDeclarations) != expectedCount {
			t.Errorf("Expected %d declarations, got %d", expectedCount, len(tools[0].FunctionDeclarations))
		}

		// MCPツールが含まれていることを確認
		foundMCP := 0
		for _, d := range tools[0].FunctionDeclarations {
			if strings.HasPrefix(d.Name, "mcp_") {
				foundMCP++
			}
		}
		if foundMCP != 2 {
			t.Errorf("Expected 2 MCP tools, found %d", foundMCP)
		}
	})
}

func TestGetCombinedToolDefinitions_JSONSerializable(t *testing.T) {
	mcpTools := []GeminiFunctionDeclaration{
		{
			Name:        "mcp_test_tool",
			Description: "Test MCP tool",
			Parameters: &GeminiParameterSchema{
				Type: "object",
				Properties: map[string]GeminiPropertyDef{
					"arg1": {Type: "string", Description: "First argument"},
				},
				Required: []string{"arg1"},
			},
		},
	}

	tools := GetCombinedToolDefinitions(mcpTools)

	// JSONにシリアライズできることを確認
	jsonData, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("Failed to marshal combined tools: %v", err)
	}

	// 再パースできることを確認
	var parsed []GeminiToolConfig
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal combined tools: %v", err)
	}

	// MCPツールが含まれていることを確認
	found := false
	for _, d := range parsed[0].FunctionDeclarations {
		if d.Name == "mcp_test_tool" {
			found = true
			if d.Parameters == nil {
				t.Error("MCP tool parameters should not be nil")
			}
			break
		}
	}
	if !found {
		t.Error("MCP tool not found after JSON round-trip")
	}
}
