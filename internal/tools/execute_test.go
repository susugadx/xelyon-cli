package tools

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestPreviewToolCall(t *testing.T) {
	tests := []struct {
		name string
		tc   *ToolCall
		want []string
	}{
		{
			name: "gather_context",
			tc:   &ToolCall{Tool: "gather_context", Args: map[string]string{"query": "Agent", "path": "internal/agent", "file_filter": "go"}},
			want: []string{"Query: Agent", "Path: internal/agent", "File filter: go"},
		},
		{
			name: "read_file",
			tc:   &ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["test.txt"]`}},
			want: []string{"File: test.txt"},
		},
		{
			name: "write_file",
			tc:   &ToolCall{Tool: "write_file", Args: map[string]string{"path": "test.txt", "content": "hello\nworld"}},
			want: []string{"File: test.txt (2 lines)"},
		},
		{
			name: "apply_patch",
			tc:   &ToolCall{Tool: "apply_patch", Args: map[string]string{"patch": "*** Begin Patch\n*** Add File: test.txt\n+hello\n*** End Patch"}},
			want: []string{"Patch: 4 lines"},
		},
		{
			name: "bash",
			tc:   &ToolCall{Tool: "bash", Args: map[string]string{"command": "ls -la"}},
			want: []string{"Command: ls -la"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color.NoColor = true
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			PreviewToolCall(tt.tc)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			for _, w := range tt.want {
				if !strings.Contains(output, w) {
					t.Errorf("PreviewToolCall() output missing %q, got: %q", w, output)
				}
			}
		})
	}
}

func TestIsBashReadOnly(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		// read-only コマンド
		{"ls -la", true},
		{"cat main.go", true},
		{"grep -r TODO .", true},
		{"git status", true},
		{"git log --oneline -5", true},
		{"git diff HEAD~1", true},
		{"go test ./...", true},
		{"pwd", true},
		{"echo hello", true},
		{"find . -name '*.go'", true},
		{"head -20 main.go", true},
		{"tree", true},
		{"env", true},

		// 空コマンド
		{"", true},
		{"  ", true},

		// unsafe コマンド（パイプ）
		{"cat main.go | grep TODO", false},
		{"ls -la | wc -l", false},

		// unsafe コマンド（リダイレクト）
		{"echo hello > file.txt", false},
		{"cat a >> b", false},

		// unsafe コマンド（連結）
		{"go fmt ./... && go build", false},
		{"ls; rm foo", false},

		// unsafe コマンド（不明なコマンド）
		{"make build", false},
		{"go build -o xelyon", false},
		{"rm -rf dist", false},
		{"npm install", false},
		{"go mod tidy", false},

		// プレフィックス誤マッチ防止（lsxyz は ls ではない）
		{"lsxyz", false},
		{"grepall foo", false},
	}
	for _, tt := range tests {
		got := IsReadOnlyBashCommand(tt.command)
		if got != tt.want {
			t.Errorf("IsReadOnlyBashCommand(%q) = %v, want %v", tt.command, got, tt.want)
		}
	}
}

func TestIsWriteToolConsistency(t *testing.T) {
	writeTools := []string{"apply_patch", "write_file", "str_replace", "delete_file"}
	for _, tool := range writeTools {
		if !IsWriteTool(tool) {
			t.Errorf("IsWriteTool(%q) = false, want true", tool)
		}
	}

	readTools := []string{"gather_context", "read_file", "list_dir", "web_search"}
	for _, tool := range readTools {
		if IsWriteTool(tool) {
			t.Errorf("IsWriteTool(%q) = true, want false", tool)
		}
	}
}
