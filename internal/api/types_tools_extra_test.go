package api

import (
	"encoding/json"
	"testing"
)

func TestConvertMCPToolToToolDefinition(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		description  string
		inputSchema  []byte
		wantName     string
		wantDesc     string
		wantFallback bool
	}{
		{
			name:        "basic tool with schema",
			toolName:    "read_file",
			description: "Read a file from disk",
			inputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string","description":"File path"}},"required":["path"]}`),
			wantName:    "read_file",
			wantDesc:    "Read a file from disk",
		},
		{
			name:         "tool with empty schema",
			toolName:     "list_tools",
			description:  "List available tools",
			inputSchema:  []byte{},
			wantName:     "list_tools",
			wantDesc:     "List available tools",
			wantFallback: true,
		},
		{
			name:         "tool with null schema",
			toolName:     "ping",
			description:  "Ping the server",
			inputSchema:  []byte("null"),
			wantName:     "ping",
			wantDesc:     "Ping the server",
			wantFallback: true,
		},
		{
			name:         "tool with invalid JSON schema",
			toolName:     "broken",
			description:  "Broken tool",
			inputSchema:  []byte("{invalid json}"),
			wantName:     "broken",
			wantDesc:     "Broken tool",
			wantFallback: true,
		},
		{
			name:         "tool with empty object schema",
			toolName:     "empty_object",
			description:  "Empty object tool",
			inputSchema:  []byte(`{}`),
			wantName:     "empty_object",
			wantDesc:     "Empty object tool",
			wantFallback: true,
		},
		{
			name:        "tool with complex schema",
			toolName:    "edit_file",
			description: "Edit a file",
			inputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"},"line":{"type":"number"}},"required":["path","content"]}`),
			wantName:    "edit_file",
			wantDesc:    "Edit a file",
		},
		{
			name:         "tool with nil schema",
			toolName:     "no_schema",
			description:  "No schema tool",
			inputSchema:  nil,
			wantName:     "no_schema",
			wantDesc:     "No schema tool",
			wantFallback: true,
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
			if got.Parameters == nil {
				t.Error("ConvertMCPToolToToolDefinition() Parameters should not be nil")
			}

			if tt.wantFallback {
				if got.Parameters["type"] != "object" {
					t.Errorf("fallback Parameters[type] = %#v, want object", got.Parameters["type"])
				}
				props, ok := got.Parameters["properties"].(map[string]interface{})
				if !ok || len(props) != 0 {
					t.Errorf("fallback Parameters[properties] = %#v, want empty map", got.Parameters["properties"])
				}
				if got.Parameters["additionalProperties"] != false {
					t.Errorf("fallback Parameters[additionalProperties] = %#v, want false", got.Parameters["additionalProperties"])
				}
			} else if _, ok := got.Parameters["properties"]; !ok {
				t.Error("ConvertMCPToolToToolDefinition() Parameters should contain 'properties'")
			}

			// JSON にマーシャルできることを確認
			if _, err := json.Marshal(got); err != nil {
				t.Errorf("ConvertMCPToolToToolDefinition() result should be JSON serializable: %v", err)
			}
		})
	}
}
