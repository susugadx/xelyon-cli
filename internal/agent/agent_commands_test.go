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
