package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestIsSameToolCall(t *testing.T) {
	tests := []struct {
		name string
		tc1  *tools.ToolCall
		tc2  *tools.ToolCall
		want bool
	}{
		{
			name: "both nil",
			tc1:  nil,
			tc2:  nil,
			want: false,
		},
		{
			name: "first nil",
			tc1:  nil,
			tc2:  &tools.ToolCall{Tool: "test"},
			want: false,
		},
		{
			name: "second nil",
			tc1:  &tools.ToolCall{Tool: "test"},
			tc2:  nil,
			want: false,
		},
		{
			name: "different tool names",
			tc1:  &tools.ToolCall{Tool: "tool1"},
			tc2:  &tools.ToolCall{Tool: "tool2"},
			want: false,
		},
		{
			name: "same tool, no args",
			tc1:  &tools.ToolCall{Tool: "test", Args: map[string]string{}},
			tc2:  &tools.ToolCall{Tool: "test", Args: map[string]string{}},
			want: true,
		},
		{
			name: "same tool, same args",
			tc1: &tools.ToolCall{
				Tool: "read_file",
				Args: map[string]string{"path": "/test/file.txt"},
			},
			tc2: &tools.ToolCall{
				Tool: "read_file",
				Args: map[string]string{"path": "/test/file.txt"},
			},
			want: true,
		},
		{
			name: "same tool, different args",
			tc1: &tools.ToolCall{
				Tool: "read_file",
				Args: map[string]string{"path": "/test/file1.txt"},
			},
			tc2: &tools.ToolCall{
				Tool: "read_file",
				Args: map[string]string{"path": "/test/file2.txt"},
			},
			want: false,
		},
		{
			name: "same tool, different number of args",
			tc1: &tools.ToolCall{
				Tool: "test",
				Args: map[string]string{"a": "1"},
			},
			tc2: &tools.ToolCall{
				Tool: "test",
				Args: map[string]string{"a": "1", "b": "2"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSameToolCall(tt.tc1, tt.tc2)
			if got != tt.want {
				t.Errorf("isSameToolCall() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "deepseek-chat",
			model: "deepseek-chat",
			want:  "DeepSeek V3 (balanced)",
		},
		{
			name:  "deepseek-coder",
			model: "deepseek-coder",
			want:  "DeepSeek Coder (code-focused)",
		},
		{
			name:  "deepseek-reasoner",
			model: "deepseek-reasoner",
			want:  "DeepSeek R1 (reasoning)",
		},
		{
			name:  "claude",
			model: "claude",
			want:  "Claude (Vertex AI)",
		},
		{
			name:  "unknown model",
			model: "gpt-4",
			want:  "gpt-4",
		},
		{
			name:  "custom model",
			model: "my-custom-model",
			want:  "my-custom-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelDisplayName(tt.model)
			if got != tt.want {
				t.Errorf("modelDisplayName(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

// TestNewAgent is skipped because NewAgent requires a valid provider
// and performs significant initialization (MCP, storage, etc.)
// These are better tested via integration tests.
func TestNewAgent(t *testing.T) {
	t.Skip("NewAgent requires a valid provider and performs I/O operations")
}
