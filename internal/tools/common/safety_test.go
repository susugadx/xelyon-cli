package common

import "testing"

func TestGetToolSafety(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     ToolSafety
	}{
		{
			name:     "read_file is SafetyHigh",
			toolName: "read_file",
			want:     SafetyHigh,
		},
		{
			name:     "search_code is SafetyHigh",
			toolName: "search_code",
			want:     SafetyHigh,
		},
		{
			name:     "write_file is SafetyMedium",
			toolName: "write_file",
			want:     SafetyMedium,
		},
		{
			name:     "delete_file is SafetyLow",
			toolName: "delete_file",
			want:     SafetyLow,
		},
		{
			name:     "bash is SafetyLow",
			toolName: "bash",
			want:     SafetyLow,
		},
		{
			name:     "web_search is SafetyMedium",
			toolName: "web_search",
			want:     SafetyMedium,
		},
		{
			name:     "ask_user_question is SafetyHigh",
			toolName: "ask_user_question",
			want:     SafetyHigh,
		},
		{
			name:     "spawn_agent is SafetyHigh",
			toolName: "spawn_agent",
			want:     SafetyHigh,
		},
		{
			name:     "wait_agent is SafetyHigh",
			toolName: "wait_agent",
			want:     SafetyHigh,
		},
		{
			name:     "unknown tool defaults to SafetyMedium",
			toolName: "unknown_tool",
			want:     SafetyMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetToolSafety(tt.toolName)
			if got != tt.want {
				t.Errorf("GetToolSafety(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestIsAutoApprovable(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		autoApprove bool
		want        bool
	}{
		{
			name:        "auto-approve disabled",
			toolName:    "write_file",
			autoApprove: false,
			want:        false,
		},
		{
			name:        "SafetyHigh tool with auto-approve",
			toolName:    "read_file",
			autoApprove: true,
			want:        true,
		},
		{
			name:        "SafetyMedium tool with auto-approve",
			toolName:    "write_file",
			autoApprove: true,
			want:        true,
		},
		{
			name:        "SafetyLow tool with auto-approve (should approve)",
			toolName:    "delete_file",
			autoApprove: true,
			want:        true,
		},
		{
			name:        "bash with auto-approve (should approve)",
			toolName:    "bash",
			autoApprove: true,
			want:        true,
		},
		{
			name:        "git_push with auto-approve (should approve)",
			toolName:    "git_push",
			autoApprove: true,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAutoApprovable(tt.toolName, tt.autoApprove)
			if got != tt.want {
				t.Errorf("IsAutoApprovable(%q, %v) = %v, want %v", tt.toolName, tt.autoApprove, got, tt.want)
			}
		})
	}
}

func TestGetSafetyDescription(t *testing.T) {
	tests := []struct {
		name   string
		safety ToolSafety
		want   string
	}{
		{
			name:   "SafetyHigh",
			safety: SafetyHigh,
			want:   "Safe (read-only)",
		},
		{
			name:   "SafetyMedium",
			safety: SafetyMedium,
			want:   "Moderate (reversible changes)",
		},
		{
			name:   "SafetyLow",
			safety: SafetyLow,
			want:   "Dangerous (destructive operation)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSafetyDescription(tt.safety)
			if got != tt.want {
				t.Errorf("GetSafetyDescription(%v) = %q, want %q", tt.safety, got, tt.want)
			}
		})
	}
}
