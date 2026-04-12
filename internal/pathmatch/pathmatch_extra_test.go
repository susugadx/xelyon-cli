package pathmatch

import (
	"reflect"
	"regexp"
	"testing"
)

func TestDefaultIgnorePatternsReturnsCopy(t *testing.T) {
	got := DefaultIgnorePatterns()
	if !reflect.DeepEqual(got, defaultIgnorePatterns) {
		t.Fatalf("DefaultIgnorePatterns() = %v, want %v", got, defaultIgnorePatterns)
	}

	got[0] = "mutated"
	again := DefaultIgnorePatterns()
	if again[0] != defaultIgnorePatterns[0] {
		t.Fatalf("DefaultIgnorePatterns() should return a copy, got %v", again)
	}
}

func TestNormalizeTarget(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect string
	}{
		{name: "empty", path: "   ", expect: ""},
		{name: "currentDir", path: "./", expect: ""},
		{name: "fileWithCleanup", path: "./internal/../internal/pathmatch.go", expect: "internal/pathmatch.go"},
		{name: "windowsPath", path: `.\pkg\demo`, isDir: true, expect: "pkg/demo/"},
		{name: "rootPrefixed", path: "/tmp/file.txt", expect: "tmp/file.txt"},
		{name: "directoryPreservesSlash", path: "internal/pathmatch", isDir: true, expect: "internal/pathmatch/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTarget(tt.path, tt.isDir); got != tt.expect {
				t.Fatalf("normalizeTarget(%q, %v) = %q, want %q", tt.path, tt.isDir, got, tt.expect)
			}
		})
	}
}

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		match   string
		noMatch string
	}{
		{name: "singleStar", pattern: "*.gen.go", match: "demo.gen.go", noMatch: "nested/demo.gen.go"},
		{name: "doubleStar", pattern: "**/generated/*.go", match: "internal/generated/file.go", noMatch: "internal/generated/file.txt"},
		{name: "questionMark", pattern: "file?.txt", match: "file1.txt", noMatch: "file12.txt"},
		{name: "characterClass", pattern: "file[0-9].txt", match: "file7.txt", noMatch: "filex.txt"},
		{name: "unterminatedClass", pattern: "file[abc", match: "file[abc", noMatch: "filex"},
		{name: "regexMetaEscaped", pattern: "config+.yaml", match: "config+.yaml", noMatch: "configx.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := regexp.MustCompile("^" + globToRegex(tt.pattern) + "$")
			if !re.MatchString(tt.match) {
				t.Fatalf("globToRegex(%q) should match %q", tt.pattern, tt.match)
			}
			if re.MatchString(tt.noMatch) {
				t.Fatalf("globToRegex(%q) should not match %q", tt.pattern, tt.noMatch)
			}
		})
	}
}

func TestCompilePatternMatchesSegmentAndPathPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		target  string
		want    bool
	}{
		{name: "segmentMatch", pattern: "node_modules", target: "src/node_modules/pkg", want: true},
		{name: "fullPathMatch", pattern: "internal/generated", target: "internal/generated/file.go", want: true},
		{name: "wildcardPathMatch", pattern: "internal/**/generated/*.go", target: "internal/foo/generated/file.go", want: true},
		{name: "wildcardMiss", pattern: "internal/**/generated/*.go", target: "internal/foo/generated/file.txt", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := compilePattern(tt.pattern)
			if got := re.MatchString(tt.target); got != tt.want {
				t.Fatalf("compilePattern(%q).MatchString(%q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
			}
		})
	}
}
