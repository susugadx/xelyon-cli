package pathmatch

import (
	"reflect"
	"testing"
)

func TestNormalizePatterns(t *testing.T) {
	patterns := NormalizePatterns([]string{" node_modules ", "./dist", "node_modules", "", "."})
	want := []string{"node_modules", "dist"}
	if !reflect.DeepEqual(patterns, want) {
		t.Fatalf("NormalizePatterns() = %v, want %v", patterns, want)
	}
}

func TestMatcherMatch(t *testing.T) {
	matcher := NewMatcher([]string{
		"node_modules",
		"internal/agent/**",
		"*.gen.go",
	})

	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{path: "src/node_modules/pkg/index.js", want: true},
		{path: "internal/agent/compress.go", want: true},
		{path: "internal/config/project.go", want: false},
		{path: "pkg/generated/foo.gen.go", want: true},
		{path: "pkg/generated/foo.go", want: false},
		{path: "internal/agent", isDir: true, want: true},
	}

	for _, tc := range cases {
		if got := matcher.Match(tc.path, tc.isDir); got != tc.want {
			t.Fatalf("Match(%q, %v) = %v, want %v", tc.path, tc.isDir, got, tc.want)
		}
	}
}

func TestBuildRGIgnoreGlobs(t *testing.T) {
	globs := BuildRGIgnoreGlobs([]string{"node_modules", "internal/generated/**", "*.gen.go"})
	want := []string{
		"!node_modules",
		"!node_modules/**",
		"!**/node_modules",
		"!**/node_modules/**",
		"!internal/generated/**",
		"!*.gen.go",
		"!**/*.gen.go",
	}

	if !reflect.DeepEqual(globs, want) {
		t.Fatalf("BuildRGIgnoreGlobs() = %v, want %v", globs, want)
	}
}
