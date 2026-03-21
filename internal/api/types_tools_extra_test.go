package api

import (
	"encoding/json"
	"testing"
)

func TestConvertMCPToolToToolDefinition(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		description string
		inputSchema []byte
		wantName    string
		wantDesc    string
		wantParams  bool // パラメータが設定されるか
	}{
		{
			name:        "basic tool with schema",
			toolName:    "read_file",
			description: "Read a file from disk",
			inputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string","description":"File path"}},"required":["path"]}`),
			wantName:    "read_file",
			wantDesc:    "Read a file from disk",
			wantParams:  true,
		},
		{
			name:        "tool with empty schema",
			toolName:    "list_tools",
			description: "List available tools",
			inputSchema: []byte{},
			wantName:    "list_tools",
			wantDesc:    "List available tools",
			wantParams:  false,
		},
		{
			name:        "tool with null schema",
			toolName:    "ping",
			description: "Ping the server",
			inputSchema: []byte("null"),
			wantName:    "ping",
			wantDesc:    "Ping the server",
			wantParams:  false,
		},
		{
			name:        "tool with invalid JSON schema",
			toolName:    "broken",
			description: "Broken tool",
			inputSchema: []byte("{invalid json}"),
			wantName:    "broken",
			wantDesc:    "Broken tool",
			wantParams:  false,
		},
		{
			name:        "tool with complex schema",
			toolName:    "edit_file",
			description: "Edit a file",
			inputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"},"line":{"type":"number"}},"required":["path","content"]}`),
			wantName:    "edit_file",
			wantDesc:    "Edit a file",
			wantParams:  true,
		},
		{
			name:        "tool with nil schema",
			toolName:    "no_schema",
			description: "No schema tool",
			inputSchema: nil,
			wantName:    "no_schema",
			wantDesc:    "No schema tool",
			wantParams:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertMCPToolToToolDefinition(tt.toolName, tt.description, tt.inputSchema)

			if got.Name != tt.wantName {
				t.Errorf("ConvertMCPToolToToolDefinition() Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.Description != tt.wantDesc {
				t.Errorf("ConvertMCPToolToToolDefinition() Description = %q, want %q", got.Description, tt.wantDesc)
			}
			if tt.wantParams && got.Parameters == nil {
				t.Error("ConvertMCPToolToToolDefinition() Parameters should not be nil")
			}
			if !tt.wantParams && got.Parameters != nil {
				t.Errorf("ConvertMCPToolToToolDefinition() Parameters should be nil, got %v", got.Parameters)
			}

			// スキーマが正しくパースされた場合、properties が含まれることを確認
			if tt.wantParams {
				if _, ok := got.Parameters["properties"]; !ok {
					t.Error("ConvertMCPToolToToolDefinition() Parameters should contain 'properties'")
				}
			}

			// JSON にマーシャルできることを確認
			if _, err := json.Marshal(got); err != nil {
				t.Errorf("ConvertMCPToolToToolDefinition() result should be JSON serializable: %v", err)
			}
		})
	}
}
