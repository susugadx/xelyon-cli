package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func TestExtractPrimaryFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "regular search results",
			input:    "Found 2 match(es) in 2 file(s)\n\n📄 src/handler.go (1 match(es))\n  line1\n\n📄 src/config.go (1 match(es))\n  line2\n",
			expected: []string{"src/handler.go", "src/config.go"},
		},
		{
			name:     "symbol definition header",
			input:    "── func HandleSSE (L10-L50) in internal/api/handler.go ──\nbody\n",
			expected: []string{"internal/api/handler.go"},
		},
		{
			name:     "symbol header with locator",
			input:    "── func Foo (L5) in pkg/foo.go @loc1 ──\nbody\n",
			expected: []string{"pkg/foo.go"},
		},
		{
			name:     "no matches",
			input:    "No matches found\n",
			expected: nil,
		},
		{
			name:     "mixed regular and symbol",
			input:    "📄 src/a.go (2 match(es))\n  line\n── type Config (L1-L10) in src/b.go ──\nbody\n",
			expected: []string{"src/a.go", "src/b.go"},
		},
		{
			name:     "deduplication",
			input:    "📄 src/a.go (1 match(es))\n  line\n📄 src/a.go (2 match(es))\n  line2\n",
			expected: []string{"src/a.go"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPrimaryFilePaths(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("extractPrimaryFilePaths() = %v (len %d), want %v (len %d)", got, len(got), tt.expected, len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestClassifyFilePath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"src/handler.go", "impl"},
		{"src/handler_test.go", "test"},
		{"src/handler.test.ts", "test"},
		{"src/handler.spec.js", "test"},
		{"test_helper.py", "test"},
		{"config.yaml", "config"},
		{"settings.yml", "config"},
		{"app.toml", "config"},
		{".env", "config"},
		{"src/main.go", "impl"},
	}
	for _, tt := range tests {
		got := classifyFilePath(tt.path)
		if got != tt.expected {
			t.Errorf("classifyFilePath(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestBuildCrossPatternIndex(t *testing.T) {
	patterns := []string{"funcA", "funcB"}
	outputs := []string{
		"📄 src/handler.go (1 match(es))\n  line1\n\n📄 src/handler_test.go (1 match(es))\n  line2\n",
		"📄 src/handler.go (1 match(es))\n  line3\n\n📄 config.yaml (1 match(es))\n  line4\n",
	}

	result := buildCrossPatternIndex(patterns, outputs, nil)

	if !strings.Contains(result, "File Index") {
		t.Error("Expected File Index header")
	}
	if !strings.Contains(result, "3 unique files") {
		t.Errorf("Expected 3 unique files, got:\n%s", result)
	}
	if !strings.Contains(result, "handler.go (★2 patterns)") {
		t.Errorf("Expected hotspot marker for handler.go, got:\n%s", result)
	}
	if !strings.Contains(result, "Impl:") {
		t.Error("Expected Impl category")
	}
	if !strings.Contains(result, "Test:") {
		t.Error("Expected Test category")
	}
	if !strings.Contains(result, "Config:") {
		t.Error("Expected Config category")
	}
}

func TestBuildCrossPatternIndex_WithLocator(t *testing.T) {
	reg := locator.NewRegistry()
	patterns := []string{"funcA", "funcB"}
	outputs := []string{
		"📄 src/handler.go (1 match(es))\n  line1\n\n📄 src/handler_test.go (1 match(es))\n  line2\n",
		"📄 src/handler.go (1 match(es))\n  line3\n\n📄 config.yaml (1 match(es))\n  line4\n",
	}

	result := buildCrossPatternIndex(patterns, outputs, reg)

	if !strings.Contains(result, "[L") {
		t.Errorf("Expected locator IDs in result, got:\n%s", result)
	}
	// 3 unique files -> 3 locator IDs
	loc1, ok := reg.Resolve("[L1]")
	if !ok {
		t.Fatal("Expected [L1] to be registered")
	}
	if loc1.FilePath != "src/handler.go" {
		t.Errorf("Expected [L1] to be src/handler.go, got %q", loc1.FilePath)
	}
}

func TestBuildCrossPatternIndex_Empty(t *testing.T) {
	result := buildCrossPatternIndex(
		[]string{"nonexistent"},
		[]string{"No matches found\n"},
		nil,
	)
	if result != "" {
		t.Errorf("Expected empty string for no matches, got: %q", result)
	}
}

func TestBuildCrossPatternIndex_Suppressed(t *testing.T) {
	// 1 unique file in 1 category, no hotspot -> should be suppressed
	result := buildCrossPatternIndex(
		[]string{"pat1", "pat2"},
		[]string{
			"📄 src/handler.go (1 match(es))\n  line1\n",
			"No matches found\n",
		},
		nil,
	)
	if result != "" {
		t.Errorf("Expected suppressed index for single file, got: %q", result)
	}
}

func TestBuildCrossPatternIndex_ShownForHotspot(t *testing.T) {
	// 1 unique file but it's a hotspot -> should be shown
	result := buildCrossPatternIndex(
		[]string{"pat1", "pat2"},
		[]string{
			"📄 src/handler.go (1 match(es))\n  line1\n",
			"📄 src/handler.go (2 match(es))\n  line2\n",
		},
		nil,
	)
	if !strings.Contains(result, "File Index") {
		t.Errorf("Expected File Index for hotspot, got: %q", result)
	}
	if !strings.Contains(result, "★2 patterns") {
		t.Errorf("Expected hotspot marker, got: %q", result)
	}
}
