package tools

import "testing"

func TestParseXMLParams(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name:    "with args wrapper",
			content: "<args>\n  <path>main.go</path>\n</args>",
			want:    map[string]string{"path": "main.go"},
		},
		{
			name:    "without args wrapper",
			content: "<path>main.go</path>\n<pattern>func main</pattern>",
			want:    map[string]string{"path": "main.go", "pattern": "func main"},
		},
		{
			name:    "single param",
			content: "<command>ls -la</command>",
			want:    map[string]string{"command": "ls -la"},
		},
		{
			name:    "empty content",
			content: "",
			want:    map[string]string{},
		},
		{
			name:    "JSON inside XML",
			content: `{"path": "main.go"}`,
			want:    map[string]string{"path": "main.go"},
		},
		{
			name:    "JSON with multiple keys",
			content: `{"pattern": "func main", "path": "."}`,
			want:    map[string]string{"pattern": "func main", "path": "."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseXMLParams(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("parseXMLParams() returned %d params, want %d: got=%v", len(got), len(tt.want), got)
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseXMLParams()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}
