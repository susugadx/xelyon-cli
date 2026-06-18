package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	toolsreg "github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"

	// ツール登録のための blank import
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
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
	tools := openairesponses.BuildToolDefinitionsWithContext(context.Background(), nil)

	// ツール数の確認（Registry に登録されている数）
	expectedCount := len(toolsreg.DefaultRegistry.GetToolDefinitions())
	if len(tools) != expectedCount {
		t.Errorf("BuildToolDefinitionsWithContext() returned %d tools, want %d", len(tools), expectedCount)
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
	requiredTools := []string{"read_file", "write_file", "str_replace", "bash", "list_dir"}
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
	mcpTools := []api.ToolDefinition{
		{
			Name:        "mcp_custom_tool",
			Description: "A custom MCP tool",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	tools := openairesponses.BuildToolDefinitionsWithContext(context.Background(), mcpTools)

	expectedCount := len(toolsreg.DefaultRegistry.GetToolDefinitions()) + 1
	if len(tools) != expectedCount {
		t.Errorf("BuildToolDefinitionsWithContext() with MCP returned %d tools, want %d", len(tools), expectedCount)
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
	tools := openairesponses.BuildToolDefinitionsWithContext(context.Background(), nil)

	// read_file ツールを探す
	var readFileTool *openairesponses.Tool
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

	if len(props) != 3 {
		t.Fatalf("read_file.Parameters.properties has %d entries, want 3", len(props))
	}
	pathsProp, ok := props["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("read_file.Parameters.properties.paths not found")
	}
	if pathsProp["type"] != "array" {
		t.Errorf("read_file.Parameters.properties.paths.type = %v, want array", pathsProp["type"])
	}
	detailProp, ok := props["detail"].(map[string]interface{})
	if !ok {
		t.Fatal("read_file.Parameters.properties.detail not found")
	}
	if detailProp["type"] != "string" {
		t.Errorf("read_file.Parameters.properties.detail.type = %v, want string", detailProp["type"])
	}

	// paths と targets は排他的なため required は未設定
	if _, hasRequired := params["required"]; hasRequired {
		t.Fatal("read_file.Parameters should not have required (paths and targets are mutually exclusive)")
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
					Arguments: `{"paths":["/test.txt"]}`,
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
					Name:      "bash",
					Arguments: `{}`,
				},
			},
			wantID:   "call_xyz789",
			wantTool: "bash",
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
			result, err := openairesponses.ConvertToolCallToToolJSON(tt.toolCall)

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

// TestResponsesUsageJSONDeserialization は ResponsesUsage の JSON デシリアライズで
// input_tokens_details.cached_tokens が正しくマッピングされることを確認
func TestResponsesUsageJSONDeserialization(t *testing.T) {
	jsonData := `{"input_tokens": 1000, "output_tokens": 200, "input_tokens_details": {"cached_tokens": 800}}`

	var usage ResponsesUsage
	if err := json.Unmarshal([]byte(jsonData), &usage); err != nil {
		t.Fatalf("Failed to unmarshal ResponsesUsage: %v", err)
	}

	if usage.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000", usage.InputTokens)
	}
	if usage.OutputTokens != 200 {
		t.Errorf("OutputTokens = %d, want 200", usage.OutputTokens)
	}
	if usage.InputTokensDetails == nil {
		t.Fatal("InputTokensDetails is nil, want non-nil")
	}
	if usage.InputTokensDetails.CachedTokens != 800 {
		t.Errorf("CachedTokens = %d, want 800", usage.InputTokensDetails.CachedTokens)
	}
}

// TestResponsesUsageWithoutCachedTokens は input_tokens_details が省略された場合に
// InputTokensDetails が nil のままであることを確認
func TestResponsesUsageWithoutCachedTokens(t *testing.T) {
	jsonData := `{"input_tokens": 500, "output_tokens": 100}`

	var usage ResponsesUsage
	if err := json.Unmarshal([]byte(jsonData), &usage); err != nil {
		t.Fatalf("Failed to unmarshal ResponsesUsage: %v", err)
	}

	if usage.InputTokens != 500 {
		t.Errorf("InputTokens = %d, want 500", usage.InputTokens)
	}
	if usage.OutputTokens != 100 {
		t.Errorf("OutputTokens = %d, want 100", usage.OutputTokens)
	}
	if usage.InputTokensDetails != nil {
		t.Errorf("InputTokensDetails = %+v, want nil", usage.InputTokensDetails)
	}
}

// TestResponsesUsageWithReasoningTokens は output_tokens_details.reasoning_tokens が
// 正しくデシリアライズされることを確認
func TestResponsesUsageWithReasoningTokens(t *testing.T) {
	jsonData := `{"input_tokens": 1000, "output_tokens": 500, "input_tokens_details": {"cached_tokens": 200}, "output_tokens_details": {"reasoning_tokens": 300}}`

	var usage ResponsesUsage
	if err := json.Unmarshal([]byte(jsonData), &usage); err != nil {
		t.Fatalf("Failed to unmarshal ResponsesUsage: %v", err)
	}

	if usage.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000", usage.InputTokens)
	}
	if usage.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want 500", usage.OutputTokens)
	}
	if usage.OutputTokensDetails == nil {
		t.Fatal("OutputTokensDetails is nil, want non-nil")
	}
	if usage.OutputTokensDetails.ReasoningTokens != 300 {
		t.Errorf("ReasoningTokens = %d, want 300", usage.OutputTokensDetails.ReasoningTokens)
	}
	if usage.InputTokensDetails == nil {
		t.Fatal("InputTokensDetails is nil, want non-nil")
	}
	if usage.InputTokensDetails.CachedTokens != 200 {
		t.Errorf("CachedTokens = %d, want 200", usage.InputTokensDetails.CachedTokens)
	}
}

// TestResponsesUsageWithoutReasoningTokens は output_tokens_details が省略された場合に
// OutputTokensDetails が nil のままであることを確認
func TestResponsesUsageWithoutReasoningTokens(t *testing.T) {
	jsonData := `{"input_tokens": 500, "output_tokens": 100}`

	var usage ResponsesUsage
	if err := json.Unmarshal([]byte(jsonData), &usage); err != nil {
		t.Fatalf("Failed to unmarshal ResponsesUsage: %v", err)
	}

	if usage.OutputTokensDetails != nil {
		t.Errorf("OutputTokensDetails = %+v, want nil", usage.OutputTokensDetails)
	}
}

// TestResponsesUsageReasoningTokensToAPIUsage は reasoning_tokens が api.Usage.ThinkingTokens に
// 移され、OutputTokens からは除外されることを確認
func TestResponsesUsageReasoningTokensToAPIUsage(t *testing.T) {
	tests := []struct {
		name               string
		usage              ResponsesUsage
		wantOutputTokens   int
		wantThinkingTokens int
	}{
		{
			name: "reasoning_tokens あり",
			usage: ResponsesUsage{
				InputTokens:  1000,
				OutputTokens: 500,
				OutputTokensDetails: &ResponsesOutputDetails{
					ReasoningTokens: 300,
				},
			},
			wantOutputTokens:   200,
			wantThinkingTokens: 300,
		},
		{
			name: "OutputTokensDetails nil",
			usage: ResponsesUsage{
				InputTokens:  1000,
				OutputTokens: 500,
			},
			wantOutputTokens:   500,
			wantThinkingTokens: 0,
		},
		{
			name: "reasoning_tokens ゼロ",
			usage: ResponsesUsage{
				InputTokens:  1000,
				OutputTokens: 500,
				OutputTokensDetails: &ResponsesOutputDetails{
					ReasoningTokens: 0,
				},
			},
			wantOutputTokens:   500,
			wantThinkingTokens: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiUsage := openairesponses.UsageToAPIUsage(&tt.usage)
			if apiUsage.OutputTokens != tt.wantOutputTokens {
				t.Errorf("OutputTokens = %d, want %d", apiUsage.OutputTokens, tt.wantOutputTokens)
			}
			if apiUsage.ThinkingTokens != tt.wantThinkingTokens {
				t.Errorf("ThinkingTokens = %d, want %d", apiUsage.ThinkingTokens, tt.wantThinkingTokens)
			}
		})
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

func TestHandleResponsesStreaming_TextToolCallsAndUsage(t *testing.T) {
	var out strings.Builder
	ctx := uiruntime.WithRuntime(context.Background(), uiruntime.NewRuntime(strings.NewReader(""), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	var gotUsage api.Usage
	p := New("test-key")
	p.SetUsageCallback(func(u api.Usage) {
		gotUsage = u
	})

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.created","response":{"id":"resp_123","status":"in_progress"}}`,
			``,
			`data: {"type":"response.output_text.delta","delta":"Hello "}`,
			``,
			`data: {"type":"response.output_item.added","item":{"type":"function_call","name":"read_file","call_id":"call_1"}}`,
			``,
			`data: {"type":"response.function_call_arguments.delta","item":{"call_id":"call_1"},"delta":"{\"path\":\"main.go\"}"}`,
			``,
			`data: {"type":"response.completed","response":{"usage":{"input_tokens":10,"output_tokens":4,"input_tokens_details":{"cached_tokens":3},"output_tokens_details":{"reasoning_tokens":2}}}}`,
			``,
			`data: [DONE]`,
		}, "\n"))),
	}

	content, responseID, err := p.handleResponsesStreaming(ctx, resp, uiruntime.NewSpinnerWithRuntime(uiruntime.RuntimeFromContext(ctx)))
	if err != nil {
		t.Fatalf("handleResponsesStreaming() error = %v", err)
	}
	if responseID != "resp_123" {
		t.Fatalf("responseID = %q, want %q", responseID, "resp_123")
	}
	if !strings.Contains(content, "Hello ") {
		t.Fatalf("content = %q, want text delta", content)
	}
	if !strings.Contains(content, `"tool":"read_file"`) || !strings.Contains(content, `"path":"main.go"`) {
		t.Fatalf("content = %q, want tool call JSON", content)
	}
	if gotUsage.InputTokens != 10 || gotUsage.OutputTokens != 2 || gotUsage.CachedInputTokens != 3 || gotUsage.ThinkingTokens != 2 {
		t.Fatalf("usage = %+v, want input=10 output=2 cached=3 thinking=2", gotUsage)
	}
}

func TestHandleResponsesStreaming_PreservesFunctionCallOutputOrder(t *testing.T) {
	var out strings.Builder
	ctx := uiruntime.WithRuntime(context.Background(), uiruntime.NewRuntime(strings.NewReader(""), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	p := New("test-key")
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.output_item.added","item":{"type":"function_call","name":"second","call_id":"call_2"}}`,
			``,
			`data: {"type":"response.output_item.added","item":{"type":"function_call","name":"first","call_id":"call_1"}}`,
			``,
			`data: {"type":"response.function_call_arguments.done","item":{"call_id":"call_2","arguments":"{\"path\":\"/tmp/2\"}"}}`,
			``,
			`data: {"type":"response.function_call_arguments.done","item":{"call_id":"call_1","arguments":"{\"path\":\"/tmp/1\"}"}}`,
			``,
			`data: {"type":"response.completed"}`,
			``,
			`data: [DONE]`,
		}, "\n"))),
	}

	content, _, err := p.handleResponsesStreaming(ctx, resp, uiruntime.NewSpinnerWithRuntime(uiruntime.RuntimeFromContext(ctx)))
	if err != nil {
		t.Fatalf("handleResponsesStreaming() error = %v", err)
	}

	idxCall2 := strings.Index(content, `"id":"call_2"`)
	idxCall1 := strings.Index(content, `"id":"call_1"`)
	if idxCall2 == -1 || idxCall1 == -1 {
		t.Fatalf("content = %q, want both call_2 and call_1", content)
	}
	if idxCall2 > idxCall1 {
		t.Fatalf("content = %q, want call_2 before call_1", content)
	}
}

func TestHandleResponsesStreaming_ErrorEvent(t *testing.T) {
	var out strings.Builder
	ctx := uiruntime.WithRuntime(context.Background(), uiruntime.NewRuntime(strings.NewReader(""), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	p := New("test-key")
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"error","error":{"message":"quota exceeded"}}`,
			``,
			`data: [DONE]`,
		}, "\n"))),
	}

	_, _, err := p.handleResponsesStreaming(ctx, resp, uiruntime.NewSpinnerWithRuntime(uiruntime.RuntimeFromContext(ctx)))
	if err == nil || !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("handleResponsesStreaming() error = %v, want quota exceeded", err)
	}
}

func TestHandleResponsesStreaming_DefaultDebugFollowsEnvWhenNotSpecified(t *testing.T) {
	t.Setenv("XELYON_DEBUG_OPENAI", "1")
	var out strings.Builder
	ctx := uiruntime.WithRuntime(context.Background(), uiruntime.NewRuntime(strings.NewReader(""), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.completed"}`,
			``,
			`data: [DONE]`,
		}, "\n"))),
	}

	var debugOut strings.Builder
	_, _, err := openairesponses.HandleStreaming(ctx, resp, uiruntime.NewSpinnerWithRuntime(uiruntime.RuntimeFromContext(ctx)), openairesponses.StreamingOptions{
		DebugWriter: &debugOut,
	})
	if err != nil {
		t.Fatalf("HandleResponsesStreaming() error = %v", err)
	}
	if !strings.Contains(debugOut.String(), "[DEBUG OpenAI Responses] SSE line:") {
		t.Fatalf("debug output = %q, want env-based debug logs", debugOut.String())
	}
}

func TestHandleResponsesStreaming_DebugOverrideFalseDisablesEnvDebug(t *testing.T) {
	t.Setenv("XELYON_DEBUG_OPENAI", "1")
	var out strings.Builder
	ctx := uiruntime.WithRuntime(context.Background(), uiruntime.NewRuntime(strings.NewReader(""), &out, &out))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.completed"}`,
			``,
			`data: [DONE]`,
		}, "\n"))),
	}

	var debugOut strings.Builder
	debugEnabled := false
	_, _, err := openairesponses.HandleStreaming(ctx, resp, uiruntime.NewSpinnerWithRuntime(uiruntime.RuntimeFromContext(ctx)), openairesponses.StreamingOptions{
		DebugOverride: &debugEnabled,
		DebugWriter:   &debugOut,
	})
	if err != nil {
		t.Fatalf("HandleResponsesStreaming() error = %v", err)
	}
	if strings.Contains(debugOut.String(), "[DEBUG ") {
		t.Fatalf("debug output = %q, want no debug logs with explicit override=false", debugOut.String())
	}
}
