package taskstate

import "testing"

func TestNormalizeRepoRelativePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
		ok   bool
	}{
		{name: "plain relative", path: "src/main.go", want: "src/main.go", ok: true},
		{name: "dot segments", path: "./src/../main.go", want: "main.go", ok: true},
		{name: "quoted", path: "`src/main.go`", want: "src/main.go", ok: true},
		{name: "parent escape", path: "../main.go"},
		{name: "absolute", path: "/tmp/main.go"},
		{name: "windows absolute", path: `C:\tmp\main.go`},
		{name: "url", path: "https://example.com/main.go"},
		{name: "glob", path: "*.go"},
		{name: "locator", path: "locator:abc"},
		{name: "locator id", path: "L12"},
		{name: "newline", path: "src/main.go\nother.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeRepoRelativePath(tt.path)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("NormalizeRepoRelativePath(%q) = %q, %v; want %q, %v", tt.path, got, ok, tt.want, tt.ok)
			}
		})
	}
}
