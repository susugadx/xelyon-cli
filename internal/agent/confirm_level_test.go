package agent

import (
	"testing"
)

func TestShouldConfirmTool_All(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		expected bool
	}{
		{"read_file", "read_file", true},
		{"write_file", "write_file", true},
		{"delete_file", "delete_file", true},
		{"bash", "bash", true},
		{"search_code", "search_code", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldConfirmTool(tt.tool, "all")
			if result != tt.expected {
				t.Errorf("ShouldConfirmTool(%s, 'all') = %v, want %v", tt.tool, result, tt.expected)
			}
		})
	}
}

func TestShouldConfirmTool_Dangerous(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		expected bool
	}{
		// SafetyHigh (read-only) tools should not require confirmation
		{"read_file", "read_file", false},
		{"search_code", "search_code", false},
		{"search_file", "search_file", false},
		{"list_dir", "list_dir", false},

		// SafetyMedium tools should not require confirmation
		{"write_file", "write_file", false},
		{"str_replace", "str_replace", false},

		// SafetyLow (dangerous) tools should require confirmation
		{"delete_file", "delete_file", true},
		{"bash", "bash", true},
		{"move_file", "move_file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldConfirmTool(tt.tool, "dangerous")
			if result != tt.expected {
				t.Errorf("ShouldConfirmTool(%s, 'dangerous') = %v, want %v", tt.tool, result, tt.expected)
			}
		})
	}
}

func TestShouldConfirmTool_None(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		expected bool
	}{
		{"read_file", "read_file", false},
		{"write_file", "write_file", false},
		{"delete_file", "delete_file", false},
		{"bash", "bash", false},
		{"any_tool", "any_tool", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldConfirmTool(tt.tool, "none")
			if result != tt.expected {
				t.Errorf("ShouldConfirmTool(%s, 'none') = %v, want %v", tt.tool, result, tt.expected)
			}
		})
	}
}

func TestShouldConfirmTool_DefaultLevel(t *testing.T) {
	// デフォルト（不明なレベル）は "dangerous" と同じ動作
	tests := []struct {
		name     string
		tool     string
		expected bool
	}{
		{"read_file", "read_file", false},
		{"delete_file", "delete_file", true},
		{"bash", "bash", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldConfirmTool(tt.tool, "unknown_level")
			if result != tt.expected {
				t.Errorf("ShouldConfirmTool(%s, 'unknown_level') = %v, want %v", tt.tool, result, tt.expected)
			}
		})
	}
}

func TestIsDangerousTool(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		expected bool
	}{
		{"read_file is not dangerous", "read_file", false},
		{"write_file is not dangerous", "write_file", false},
		{"delete_file is dangerous", "delete_file", true},
		{"bash is dangerous", "bash", true},
		{"move_file is dangerous", "move_file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDangerousTool(tt.tool)
			if result != tt.expected {
				t.Errorf("IsDangerousTool(%s) = %v, want %v", tt.tool, result, tt.expected)
			}
		})
	}
}

func TestValidateConfirmLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected bool
	}{
		{"all is valid", "all", true},
		{"dangerous is valid", "dangerous", true},
		{"none is valid", "none", true},
		{"empty is invalid", "", false},
		{"unknown is invalid", "unknown", false},
		{"ALL is invalid (case sensitive)", "ALL", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateConfirmLevel(tt.level)
			if result != tt.expected {
				t.Errorf("ValidateConfirmLevel(%s) = %v, want %v", tt.level, result, tt.expected)
			}
		})
	}
}
