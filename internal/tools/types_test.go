package tools

import (
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
