package repomap

import "testing"

func TestPatternLangForPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "main.go", want: "go"},
		{path: "src/app.tsx", want: "js"},
		{path: "lib/tasks.py", want: "py"},
		{path: "src/lib.rs", want: "rs"},
		{path: "pkg/UserService.java", want: "java"},
		{path: "tool.sh", want: "sh"},
		{path: "README.md", want: ""},
	}

	for _, tt := range tests {
		if got := patternLangForPath(tt.path); got != tt.want {
			t.Fatalf("patternLangForPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
