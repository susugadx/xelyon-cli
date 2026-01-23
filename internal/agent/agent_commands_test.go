package agent

import (
	"testing"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{
			name: "less than 1000",
			n:    999,
			want: "999",
		},
		{
			name: "exactly 1000",
			n:    1000,
			want: "1,000",
		},
		{
			name: "ten thousand",
			n:    10000,
			want: "10,000",
		},
		{
			name: "hundred thousand",
			n:    100000,
			want: "100,000",
		},
		{
			name: "one million",
			n:    1000000,
			want: "1,000,000",
		},
		{
			name: "arbitrary large number",
			n:    1234567,
			want: "1,234,567",
		},
		{
			name: "zero",
			n:    0,
			want: "0",
		},
		{
			name: "small number",
			n:    42,
			want: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNumber(tt.n)
			if got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single command",
			input: "/help",
			want:  []string{"/help"},
		},
		{
			name:  "command with argument",
			input: "/load session1",
			want:  []string{"/load", "session1"},
		},
		{
			name:  "command with multiple arguments",
			input: "/search file.txt pattern",
			want:  []string{"/search", "file.txt", "pattern"},
		},
		{
			name:  "double quoted argument",
			input: `/search "hello world"`,
			want:  []string{"/search", "hello world"},
		},
		{
			name:  "single quoted argument",
			input: `/search 'hello world'`,
			want:  []string{"/search", "hello world"},
		},
		{
			name:  "mixed quotes",
			input: `/cmd "arg1" 'arg2'`,
			want:  []string{"/cmd", "arg1", "arg2"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string(nil),
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  []string(nil),
		},
		{
			name:  "multiple spaces between args",
			input: "/cmd   arg1    arg2",
			want:  []string{"/cmd", "arg1", "arg2"},
		},
		{
			name:  "quoted string with spaces",
			input: `/echo "this is a long message"`,
			want:  []string{"/echo", "this is a long message"},
		},
		{
			name:  "nested quotes not supported",
			input: `/cmd "outer 'inner' outer"`,
			want:  []string{"/cmd", "outer 'inner' outer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCommand(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitCommand(%q) = %v (len=%d), want %v (len=%d)",
					tt.input, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitCommand(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetChangeTypeJP(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want string
	}{
		{
			name: "write_file",
			tool: "write_file",
			want: "📝 作成",
		},
		{
			name: "str_replace",
			tool: "str_replace",
			want: "✏️  編集",
		},
		{
			name: "append_file",
			tool: "append_file",
			want: "✏️  編集",
		},
		{
			name: "prepend_file",
			tool: "prepend_file",
			want: "✏️  編集",
		},
		{
			name: "insert_after",
			tool: "insert_after",
			want: "✏️  編集",
		},
		{
			name: "insert_before",
			tool: "insert_before",
			want: "✏️  編集",
		},
		{
			name: "delete_file",
			tool: "delete_file",
			want: "🗑️  削除",
		},
		{
			name: "move_file",
			tool: "move_file",
			want: "📦 移動",
		},
		{
			name: "copy_file",
			tool: "copy_file",
			want: "📋 コピー",
		},
		{
			name: "delete_lines",
			tool: "delete_lines",
			want: "✂️  行削除",
		},
		{
			name: "unknown tool",
			tool: "unknown_tool",
			want: "🔧 変更",
		},
		{
			name: "empty string",
			tool: "",
			want: "🔧 変更",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getChangeTypeJP(tt.tool)
			if got != tt.want {
				t.Errorf("getChangeTypeJP(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}
