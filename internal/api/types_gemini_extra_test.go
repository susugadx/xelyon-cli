package api

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestConvertPropertyDef(t *testing.T) {
	tests := []struct {
		name     string
		propMap  map[string]any
		wantType string
		wantDesc string
		wantEnum []string
		hasItems bool
	}{
		{
			name: "string property",
			propMap: map[string]any{
				"type":        "string",
				"description": "A file path",
			},
			wantType: "string",
			wantDesc: "A file path",
		},
		{
			name: "number property",
			propMap: map[string]any{
				"type":        "number",
				"description": "Line number",
			},
			wantType: "number",
			wantDesc: "Line number",
		},
		{
			name: "boolean property",
			propMap: map[string]any{
				"type":        "boolean",
				"description": "Verbose mode",
			},
			wantType: "boolean",
			wantDesc: "Verbose mode",
		},
		{
			name: "array property with items",
			propMap: map[string]any{
				"type":        "array",
				"description": "List of files",
				"items": map[string]any{
					"type": "string",
				},
			},
			wantType: "array",
			wantDesc: "List of files",
			hasItems: true,
		},
		{
			name: "array property without items defaults to string",
			propMap: map[string]any{
				"type":        "array",
				"description": "Generic list",
			},
			wantType: "array",
			wantDesc: "Generic list",
			hasItems: true,
		},
		{
			name: "array property with items missing type defaults to string",
			propMap: map[string]any{
				"type": "array",
				"items": map[string]any{
					"description": "item without type",
				},
			},
			wantType: "array",
			hasItems: true,
		},
		{
			name: "property with enum",
			propMap: map[string]any{
				"type": "string",
				"enum": []any{"low", "medium", "high"},
			},
			wantType: "string",
			wantEnum: []string{"low", "medium", "high"},
		},
		{
			name:     "empty property map",
			propMap:  map[string]any{},
			wantType: "",
		},
		{
			name: "object property (no special handling)",
			propMap: map[string]any{
				"type":        "object",
				"description": "Nested object",
			},
			wantType: "object",
			wantDesc: "Nested object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertPropertyDef(tt.propMap)

			if got.Type != tt.wantType {
				t.Errorf("convertPropertyDef() Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Description != tt.wantDesc {
				t.Errorf("convertPropertyDef() Description = %q, want %q", got.Description, tt.wantDesc)
			}
			if tt.wantEnum != nil {
				if len(got.Enum) != len(tt.wantEnum) {
					t.Errorf("convertPropertyDef() Enum length = %d, want %d", len(got.Enum), len(tt.wantEnum))
				} else {
					for i, v := range tt.wantEnum {
						if got.Enum[i] != v {
							t.Errorf("convertPropertyDef() Enum[%d] = %q, want %q", i, got.Enum[i], v)
						}
					}
				}
			}
			if tt.hasItems {
				if got.Items == nil {
					t.Error("convertPropertyDef() Items should not be nil for array type")
				} else if got.Items.Type == "" {
					t.Error("convertPropertyDef() Items.Type should not be empty")
				}
			}
			if !tt.hasItems && got.Items != nil {
				t.Errorf("convertPropertyDef() Items should be nil for non-array type, got %+v", got.Items)
			}
		})
	}
}

func TestConvertMCPToolToGeminiDeclaration(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		description string
		inputSchema json.RawMessage
		wantName    string
		wantDesc    string
		wantReq     []string // expected required fields
	}{
		{
			name:        "basic tool with properties",
			toolName:    "read_file",
			description: "Read a file",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"File path"}},"required":["path"]}`),
			wantName:    "read_file",
			wantDesc:    "Read a file",
			wantReq:     []string{"path"},
		},
		{
			name:        "tool with empty schema",
			toolName:    "list",
			description: "List items",
			inputSchema: json.RawMessage{},
			wantName:    "list",
			wantDesc:    "List items",
		},
		{
			name:        "tool with null schema",
			toolName:    "ping",
			description: "Ping server",
			inputSchema: json.RawMessage("null"),
			wantName:    "ping",
			wantDesc:    "Ping server",
		},
		{
			name:        "tool with invalid JSON",
			toolName:    "broken",
			description: "Broken tool",
			inputSchema: json.RawMessage("{invalid}"),
			wantName:    "broken",
			wantDesc:    "Broken tool",
		},
		{
			name:        "tool with multiple properties and required",
			toolName:    "edit_file",
			description: "Edit a file",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"},"line":{"type":"number"}},"required":["path","content"]}`),
			wantName:    "edit_file",
			wantDesc:    "Edit a file",
			wantReq:     []string{"path", "content"},
		},
		{
			name:        "tool with array property",
			toolName:    "multi_read",
			description: "Read multiple files",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"paths":{"type":"array","items":{"type":"string"}}},"required":["paths"]}`),
			wantName:    "multi_read",
			wantDesc:    "Read multiple files",
			wantReq:     []string{"paths"},
		},
		{
			name:        "tool with no properties in schema",
			toolName:    "empty_schema",
			description: "Empty schema tool",
			inputSchema: json.RawMessage(`{"type":"object"}`),
			wantName:    "empty_schema",
			wantDesc:    "Empty schema tool",
		},
	}

	debugOut := &bytes.Buffer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debugOut.Reset()
			got := ConvertMCPToolToGeminiDeclaration(tt.toolName, tt.description, tt.inputSchema, debugOut)

			if got.Name != tt.wantName {
				t.Errorf("ConvertMCPToolToGeminiDeclaration() Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.Description != tt.wantDesc {
				t.Errorf("ConvertMCPToolToGeminiDeclaration() Description = %q, want %q", got.Description, tt.wantDesc)
			}
			if got.Parameters == nil {
				t.Error("ConvertMCPToolToGeminiDeclaration() Parameters should not be nil")
			}

			if got.Parameters != nil {
				// Type が "object" であることを確認
				if got.Parameters.Type != "object" {
					t.Errorf("ConvertMCPToolToGeminiDeclaration() Parameters.Type = %q, want %q", got.Parameters.Type, "object")
				}

				// required フィールドを確認
				if tt.wantReq != nil {
					if len(got.Parameters.Required) != len(tt.wantReq) {
						t.Errorf("ConvertMCPToolToGeminiDeclaration() Required length = %d, want %d", len(got.Parameters.Required), len(tt.wantReq))
					} else {
						for i, r := range tt.wantReq {
							if got.Parameters.Required[i] != r {
								t.Errorf("ConvertMCPToolToGeminiDeclaration() Required[%d] = %q, want %q", i, got.Parameters.Required[i], r)
							}
						}
					}
				}
			}

			// JSON にマーシャルできることを確認
			if _, err := json.Marshal(got); err != nil {
				t.Errorf("ConvertMCPToolToGeminiDeclaration() result should be JSON serializable: %v", err)
			}
		})
	}
}
