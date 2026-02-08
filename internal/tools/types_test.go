package tools

import (
	"strings"
	"testing"
)

func TestToolCall_NormalizeArgs(t *testing.T) {
	tests := []struct {
		name    string
		rawArgs map[string]any
		want    map[string]string
	}{
		{
			name:    "string values",
			rawArgs: map[string]any{"path": "/home/user/file.txt", "content": "hello"},
			want:    map[string]string{"path": "/home/user/file.txt", "content": "hello"},
		},
		{
			name:    "integer as float64",
			rawArgs: map[string]any{"start_line": float64(10), "end_line": float64(20)},
			want:    map[string]string{"start_line": "10", "end_line": "20"},
		},
		{
			name:    "float value",
			rawArgs: map[string]any{"ratio": 3.14159},
			want:    map[string]string{"ratio": "3.14159"},
		},
		{
			name:    "boolean values",
			rawArgs: map[string]any{"recursive": true, "force": false},
			want:    map[string]string{"recursive": "true", "force": "false"},
		},
		{
			name:    "int64 value",
			rawArgs: map[string]any{"size": int64(1024)},
			want:    map[string]string{"size": "1024"},
		},
		{
			name:    "mixed types",
			rawArgs: map[string]any{"path": "/test", "line": float64(5), "force": true},
			want:    map[string]string{"path": "/test", "line": "5", "force": "true"},
		},
		{
			name:    "empty raw args",
			rawArgs: map[string]any{},
			want:    map[string]string{},
		},
		{
			name:    "nil value (default case)",
			rawArgs: map[string]any{"unknown": nil},
			want:    map[string]string{"unknown": "<nil>"},
		},
		{
			name:    "slice value (default case)",
			rawArgs: map[string]any{"items": []string{"a", "b"}},
			want:    map[string]string{"items": "[a b]"},
		},
		{
			name:    "large integer as float64",
			rawArgs: map[string]any{"big": float64(1000000)},
			want:    map[string]string{"big": "1000000"},
		},
		{
			name:    "path with junk suffix",
			rawArgs: map[string]any{"path": "internal/agent/agent.go lodge-187541604"},
			want:    map[string]string{"path": "internal/agent/agent.go"},
		},
		{
			name:    "path with trailing control chars",
			rawArgs: map[string]any{"path": "main.go\x00\x00"},
			want:    map[string]string{"path": "main.go"},
		},
		{
			name:    "normal path unchanged",
			rawArgs: map[string]any{"path": "internal/tools/types.go"},
			want:    map[string]string{"path": "internal/tools/types.go"},
		},
		{
			name:    "content not trimmed by path logic",
			rawArgs: map[string]any{"content": "hello world", "path": "test.go"},
			want:    map[string]string{"content": "hello world", "path": "test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &ToolCall{
				Tool:    "test_tool",
				RawArgs: tt.rawArgs,
			}
			tc.NormalizeArgs()

			if len(tc.Args) != len(tt.want) {
				t.Errorf("NormalizeArgs() got %d args, want %d", len(tc.Args), len(tt.want))
				return
			}

			for k, wantV := range tt.want {
				if gotV, ok := tc.Args[k]; !ok {
					t.Errorf("NormalizeArgs() missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("NormalizeArgs()[%q] = %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}

func TestSanitizeArgValue_PathWithJunkSuffix(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{"normal path", "path", "internal/agent/agent.go", "internal/agent/agent.go"},
		{"path with random ID", "path", "internal/agent/agent.go lodge-187541604", "internal/agent/agent.go"},
		{"path with hash suffix", "path", "main.go abc123def", "main.go"},
		{"path with multiple spaces", "path", "dir/file.py extra junk data", "dir/file.py"},
		{"directory path with suffix", "path", "src/components/ extra", "src/components/"},
		{"non-path key not trimmed", "content", "hello world foo", "hello world foo"},
		{"old_str not trimmed", "old_str", "func main() { lodge-123", "func main() { lodge-123"},
		{"path without extension", "path", "Makefile", "Makefile"},
		{"path with trailing null", "path", "file.go\x00", "file.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeArgValue(tt.key, tt.value)
			if got != tt.want {
				t.Errorf("sanitizeArgValue(%q, %q) = %q, want %q", tt.key, tt.value, got, tt.want)
			}
		})
	}
}

func TestLooksLikeFilePath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"main.go", true},
		{"internal/agent/agent.go", true},
		{"src/", true},
		{"Makefile", false},
		{"lodge-187541604", false},
		{"file.py", true},
		{"README.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeFilePath(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeFilePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanPathArg(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"internal/agent/agent.go", "internal/agent/agent.go"},
		{"internal/agent/agent.go lodge-187541604", "internal/agent/agent.go"},
		{"main.go abc123", "main.go"},
		{"Makefile", "Makefile"},
		// Makefile + space: Makefile doesn't have extension, so looksLikeFilePath returns false → no trim
		{"Makefile extra", "Makefile extra"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanPathArg(tt.input)
			if !strings.EqualFold(got, tt.want) {
				t.Errorf("cleanPathArg(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
