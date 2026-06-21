package mcp

import "testing"

func TestShouldIncludeTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		filter   *ToolsFilter
		want     bool
	}{
		{"nil filter", "any_tool", nil, true},
		{"empty filter", "any_tool", &ToolsFilter{}, true},
		{"include match", "create_issue", &ToolsFilter{Include: []string{"create_issue", "list_issues"}}, true},
		{"include no match", "delete_repo", &ToolsFilter{Include: []string{"create_issue", "list_issues"}}, false},
		{"exclude match", "delete_repo", &ToolsFilter{Exclude: []string{"delete_repo"}}, false},
		{"exclude no match", "create_issue", &ToolsFilter{Exclude: []string{"delete_repo"}}, true},
		{
			"include takes precedence",
			"create_issue",
			&ToolsFilter{
				Include: []string{"create_issue"},
				Exclude: []string{"create_issue"},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldIncludeTool(tt.toolName, tt.filter)
			if got != tt.want {
				t.Errorf("shouldIncludeTool(%q, %+v) = %v, want %v",
					tt.toolName, tt.filter, got, tt.want)
			}
		})
	}
}
