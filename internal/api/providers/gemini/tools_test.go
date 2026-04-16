package gemini

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"

	// ツール登録のための blank import
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

// builtinToolCount は DefaultRegistry に登録された組み込みツール数を動的に取得する。
// 新ツール追加時にテストの数値を手動で更新する必要がない。
func builtinToolCount() int {
	return len(tools.DefaultRegistry.GetToolDefinitions())
}

func TestGetGeminiToolDefinitions(t *testing.T) {
	tools := GetGeminiToolDefinitions()

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool config, got %d", len(tools))
	}

	declarations := tools[0].FunctionDeclarations
	if len(declarations) == 0 {
		t.Error("Expected at least one function declaration")
	}

	// DefaultRegistry のツール数と一致すること
	expectedCount := builtinToolCount()
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
		input    *api.GeminiFunctionCall
		expected map[string]any
	}{
		{
			name: "read_file with path",
			input: &api.GeminiFunctionCall{
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
			input: &api.GeminiFunctionCall{
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
			name: "bash with command",
			input: &api.GeminiFunctionCall{
				Name: "bash",
				Args: map[string]any{"command": "git status"},
			},
			expected: map[string]any{
				"tool": "bash",
				"args": map[string]any{"command": "git status"},
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
	fc := &api.GeminiFunctionCall{
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
	// 注: 新ツール追加時はここにも追加が必要
	expectedTools := []string{
		// File Operations (5)
		"read_file", "write_file", "str_replace", "delete_file",
		"list_dir",
		// Search Operations (1)
		"web_search",
		// Development Operations (1)
		"bash",
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
	jsonData := `{"name": "read_file", "args": {"paths": ["/test/file.txt"]}}`

	var fc api.GeminiFunctionCall
	if err := json.Unmarshal([]byte(jsonData), &fc); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if fc.Name != "read_file" {
		t.Errorf("Expected name=read_file, got %s", fc.Name)
	}

	paths, ok := fc.Args["paths"].([]any)
	if !ok {
		t.Fatalf("Expected paths to be []any, got %T", fc.Args["paths"])
	}
	if len(paths) != 1 || paths[0] != "/test/file.txt" {
		t.Errorf("Expected paths=[/test/file.txt], got %v", paths)
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
	var parsed []api.GeminiToolConfig
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
			decl := api.ConvertMCPToolToGeminiDeclaration(tc.toolName, tc.description, tc.inputSchema, os.Stderr)

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
	decl := api.ConvertMCPToolToGeminiDeclaration("mcp_test", "Test", json.RawMessage(`{invalid json`), os.Stderr)

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

	decl := api.ConvertMCPToolToGeminiDeclaration("mcp_test_array", "Test array tool", inputSchema, os.Stderr)

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

	decl := api.ConvertMCPToolToGeminiDeclaration("mcp_test_no_items", "Test without items", inputSchema, os.Stderr)

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

	decl := api.ConvertMCPToolToGeminiDeclaration("mcp_test_empty_type", "Test with empty items type", inputSchema, os.Stderr)

	valuesProp := decl.Parameters.Properties["values"]
	if valuesProp.Items == nil {
		t.Fatal("Expected values.items to be set")
	}
	// type が空の場合はデフォルトで string
	if valuesProp.Items.Type != "string" {
		t.Errorf("Expected values.items.type=string (default), got %s", valuesProp.Items.Type)
	}
}

// ===== convertToGeminiSchema テスト（組み込みツールの変換） =====

func TestConvertToGeminiSchema_ArrayType(t *testing.T) {
	// array型のプロパティが正しく変換されることを確認
	params := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"files": map[string]interface{}{
				"type":        "array",
				"description": "List of files",
				"items": map[string]interface{}{
					"type":        "string",
					"description": "File path",
				},
			},
			"numbers": map[string]interface{}{
				"type":        "array",
				"description": "List of numbers",
				"items": map[string]interface{}{
					"type": "integer",
				},
			},
		},
		"required": []interface{}{"files"},
	}

	schema := convertToGeminiSchema(params)

	if schema == nil {
		t.Fatal("Expected schema to be set")
	}

	// filesプロパティの確認
	filesProp, ok := schema.Properties["files"]
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

	// numbersプロパティの確認
	numbersProp, ok := schema.Properties["numbers"]
	if !ok {
		t.Fatal("Expected 'numbers' property to exist")
	}
	if numbersProp.Items == nil {
		t.Fatal("Expected numbers.items to be set")
	}
	if numbersProp.Items.Type != "integer" {
		t.Errorf("Expected numbers.items.type=integer, got %s", numbersProp.Items.Type)
	}
}

func TestConvertToGeminiSchema_ArrayWithoutItems(t *testing.T) {
	// array型でitemsが指定されていない場合、デフォルトでstring型を設定
	params := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"tags": map[string]interface{}{
				"type":        "array",
				"description": "List of tags",
			},
		},
	}

	schema := convertToGeminiSchema(params)

	tagsProp, ok := schema.Properties["tags"]
	if !ok {
		t.Fatal("Expected 'tags' property to exist")
	}
	if tagsProp.Items == nil {
		t.Fatal("Expected tags.items to be set (default)")
	}
	if tagsProp.Items.Type != "string" {
		t.Errorf("Expected tags.items.type=string (default), got %s", tagsProp.Items.Type)
	}
}

func TestConvertToGeminiSchema_ArrayWithEnum(t *testing.T) {
	// array型のitemsにenumがある場合
	params := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"statuses": map[string]interface{}{
				"type":        "array",
				"description": "List of statuses",
				"items": map[string]interface{}{
					"type": "string",
					"enum": []interface{}{"open", "closed", "pending"},
				},
			},
		},
	}

	schema := convertToGeminiSchema(params)

	statusesProp := schema.Properties["statuses"]
	if statusesProp.Items == nil {
		t.Fatal("Expected statuses.items to be set")
	}
	if statusesProp.Items.Type != "string" {
		t.Errorf("Expected statuses.items.type=string, got %s", statusesProp.Items.Type)
	}
	if len(statusesProp.Items.Enum) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(statusesProp.Items.Enum))
	}
}

func TestConvertToGeminiSchema_JSONSerializable(t *testing.T) {
	// array型を含むスキーマがJSONにシリアライズできることを確認
	params := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"assignees": map[string]interface{}{
				"type":        "array",
				"description": "List of assignees",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"required": []interface{}{"assignees"},
	}

	schema := convertToGeminiSchema(params)

	jsonData, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, `"items"`) {
		t.Errorf("Expected JSON to contain 'items', got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"type":"string"`) {
		t.Errorf("Expected JSON to contain items type, got: %s", jsonStr)
	}
}

func TestGetCombinedToolDefinitions(t *testing.T) {
	// MCPツールなしの場合
	t.Run("no MCP tools", func(t *testing.T) {
		tools := GetCombinedToolDefinitions(nil)

		if len(tools) != 1 {
			t.Fatalf("Expected 1 tool config, got %d", len(tools))
		}

		// 組み込みツールのみ
		expectedCount := builtinToolCount()
		if len(tools[0].FunctionDeclarations) != expectedCount {
			t.Errorf("Expected %d declarations, got %d", expectedCount, len(tools[0].FunctionDeclarations))
		}
	})

	// MCPツールありの場合
	t.Run("with MCP tools", func(t *testing.T) {
		mcpTools := []api.ToolDefinition{
			{Name: "mcp_github_get_issue", Description: "Get issue", Parameters: nil},
			{Name: "mcp_github_list_repos", Description: "List repos", Parameters: nil},
		}

		tools := GetCombinedToolDefinitions(mcpTools)

		if len(tools) != 1 {
			t.Fatalf("Expected 1 tool config, got %d", len(tools))
		}

		// 組み込み + MCP2
		expectedCount := builtinToolCount() + 2
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

	// 重複するMCPツールがある場合（最初のものが優先される）
	t.Run("with duplicate MCP tools", func(t *testing.T) {
		mcpTools := []api.ToolDefinition{
			{Name: "mcp_github_create_or_update_file", Description: "First description", Parameters: nil},
			{Name: "mcp_github_create_or_update_file", Description: "Second description (duplicate)", Parameters: nil},
			{Name: "mcp_github_list_repos", Description: "List repos", Parameters: nil},
		}

		tools := GetCombinedToolDefinitions(mcpTools)

		// 重複が除去されて 組み込み + 2 になる
		expectedCount := builtinToolCount() + 2
		if len(tools[0].FunctionDeclarations) != expectedCount {
			t.Errorf("Expected %d declarations (duplicates removed), got %d", expectedCount, len(tools[0].FunctionDeclarations))
		}

		// 重複ツールが1つだけ含まれていることを確認
		count := 0
		for _, d := range tools[0].FunctionDeclarations {
			if d.Name == "mcp_github_create_or_update_file" {
				count++
				// 最初の description が使われる
				if d.Description != "First description" {
					t.Errorf("Expected first description to be kept, got %q", d.Description)
				}
			}
		}
		if count != 1 {
			t.Errorf("Expected exactly 1 mcp_github_create_or_update_file, found %d", count)
		}
	})
}

func TestGetCombinedToolDefinitions_JSONSerializable(t *testing.T) {
	mcpTools := []api.ToolDefinition{
		{
			Name:        "mcp_test_tool",
			Description: "Test MCP tool",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"arg1": map[string]interface{}{"type": "string", "description": "First argument"},
				},
				"required": []string{"arg1"},
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
	var parsed []api.GeminiToolConfig
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
