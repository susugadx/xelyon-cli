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

func TestBuildCrossPatternExecutions_TruncatesByShorterInput(t *testing.T) {
	patterns := []string{"pat1", "pat2", "pat3"}
	outputs := []string{"out1", "out2"}

	collected := buildCrossPatternExecutions(patterns, outputs)
	if len(collected) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(collected))
	}
	if collected[0].Index != 0 || collected[1].Index != 1 {
		t.Fatalf("unexpected indices: %+v", collected)
	}
	if collected[0].Pattern != "pat1" || collected[0].Output != "out1" {
		t.Fatalf("unexpected first execution: %+v", collected[0])
	}
	if collected[1].Pattern != "pat2" || collected[1].Output != "out2" {
		t.Fatalf("unexpected second execution: %+v", collected[1])
	}
}

func TestBuildCrossPatternIndexData_CollectsSectionsAndHotspot(t *testing.T) {
	collected := buildCrossPatternExecutions(
		[]string{"pat1", "pat2"},
		[]string{
			"📄 src/handler.go (1 match(es))\n  line1\n\n📄 src/handler_test.go (1 match(es))\n  line2\n",
			"📄 src/handler.go (1 match(es))\n  line3\n\n📄 config.yaml (1 match(es))\n  line4\n",
		},
	)

	data := buildCrossPatternIndexData(collected, SearchOptions{})
	if data.isEmpty() {
		t.Fatal("expected non-empty index data")
	}
	if !data.hasHotspot {
		t.Fatal("expected hotspot to be detected")
	}
	if len(data.sections.implKeys) != 1 {
		t.Fatalf("expected 1 impl key, got %d", len(data.sections.implKeys))
	}
	if len(data.sections.testKeys) != 1 {
		t.Fatalf("expected 1 test key, got %d", len(data.sections.testKeys))
	}
	if len(data.sections.configKeys) != 1 {
		t.Fatalf("expected 1 config key, got %d", len(data.sections.configKeys))
	}
	if !data.shouldRender() {
		t.Fatal("expected index to be renderable")
	}
}

func TestBuildCrossPatternIndexData_SingleFileWithoutHotspotIsSuppressed(t *testing.T) {
	collected := buildCrossPatternExecutions(
		[]string{"pat1"},
		[]string{"📄 src/handler.go (1 match(es))\n  line1\n"},
	)

	data := buildCrossPatternIndexData(collected, SearchOptions{})
	if data.isEmpty() {
		t.Fatal("expected non-empty index data")
	}
	if data.shouldRender() {
		t.Fatal("expected single-file non-hotspot index to be suppressed")
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

func TestAppendCrossPatternIndexLocator(t *testing.T) {
	entry := &crossPatternIndexEntry{
		ref: primaryFileRef{
			DisplayPath:  "src/handler.go",
			ResolvedPath: "/tmp/src/handler.go",
		},
		patternCount: 1,
	}

	base := formatCrossPatternIndexEntryLine(entry)
	if got := appendCrossPatternIndexLocator(base, entry, nil); got != base {
		t.Fatalf("expected line without registry to remain unchanged, got %q", got)
	}

	reg := locator.NewRegistry()
	withLoc := appendCrossPatternIndexLocator(base, entry, reg)
	if !strings.Contains(withLoc, "[L1]") {
		t.Fatalf("expected locator id to be appended, got %q", withLoc)
	}
	loc, ok := reg.Resolve("[L1]")
	if !ok {
		t.Fatal("expected [L1] to be registered")
	}
	if loc.FilePath != "src/handler.go" {
		t.Fatalf("unexpected locator file path: %+v", loc)
	}
}

func TestCrossPatternIndexSectionsCategoryCount(t *testing.T) {
	sections := crossPatternIndexSections{
		implKeys: []string{"impl"},
		testKeys: []string{"test"},
	}
	if got := sections.categoryCount(); got != 2 {
		t.Fatalf("expected category count 2, got %d", got)
	}

	if got := (crossPatternIndexSections{}).categoryCount(); got != 0 {
		t.Fatalf("expected empty category count 0, got %d", got)
	}
}

func TestCrossPatternIndexRenderPolicyShouldRender(t *testing.T) {
	policy := crossPatternIndexRenderPolicy{MinCategoryCount: 2, MinUniqueFiles: 3}

	if !policy.shouldRender([]string{"a"}, crossPatternIndexSections{}, true) {
		t.Fatal("expected hotspot to force render")
	}
	if !policy.shouldRender([]string{"a", "b"}, crossPatternIndexSections{implKeys: []string{"a"}, testKeys: []string{"b"}}, false) {
		t.Fatal("expected mixed categories to render")
	}
	if !policy.shouldRender([]string{"a", "b", "c"}, crossPatternIndexSections{implKeys: []string{"a", "b", "c"}}, false) {
		t.Fatal("expected enough unique files to render")
	}
	if policy.shouldRender([]string{"a"}, crossPatternIndexSections{implKeys: []string{"a"}}, false) {
		t.Fatal("expected single-file non-hotspot index to be suppressed")
	}
}

func TestCrossPatternIndexCollectorAddRef(t *testing.T) {
	collector := newCrossPatternIndexCollector()
	ref := primaryFileRef{DisplayPath: "src/handler.go", ResolvedPath: "/tmp/src/handler.go"}

	collector.addRef(ref)
	collector.addRef(ref)

	fileMap, order := collector.results()
	if len(order) != 1 {
		t.Fatalf("expected one ordered key, got %d", len(order))
	}
	entry, ok := fileMap[crossPatternIndexEntryKey(ref)]
	if !ok {
		t.Fatalf("expected entry for key %q", crossPatternIndexEntryKey(ref))
	}
	if entry.patternCount != 2 {
		t.Fatalf("expected pattern count 2, got %d", entry.patternCount)
	}
	if entry.category != "impl" {
		t.Fatalf("expected impl category, got %q", entry.category)
	}
}

func TestSummarizeCrossPatternIndex(t *testing.T) {
	fileMap := map[string]*crossPatternIndexEntry{
		"impl":   {patternCount: 1, category: "impl"},
		"test":   {patternCount: 2, category: "test"},
		"config": {patternCount: 1, category: "config"},
	}
	order := []string{"impl", "test", "config"}

	sections, hasHotspot := summarizeCrossPatternIndex(fileMap, order)
	if !hasHotspot {
		t.Fatal("expected hotspot to be detected")
	}
	if len(sections.implKeys) != 1 || sections.implKeys[0] != "impl" {
		t.Fatalf("unexpected impl keys: %+v", sections.implKeys)
	}
	if len(sections.testKeys) != 1 || sections.testKeys[0] != "test" {
		t.Fatalf("unexpected test keys: %+v", sections.testKeys)
	}
	if len(sections.configKeys) != 1 || sections.configKeys[0] != "config" {
		t.Fatalf("unexpected config keys: %+v", sections.configKeys)
	}
}

func TestBuildCrossPatternIndexGroupLines(t *testing.T) {
	fileMap := map[string]*crossPatternIndexEntry{
		"impl": {
			ref: primaryFileRef{DisplayPath: "src/handler.go", ResolvedPath: "/tmp/src/handler.go"},
		},
		"hotspot": {
			ref:          primaryFileRef{DisplayPath: "src/service.go", ResolvedPath: "/tmp/src/service.go"},
			patternCount: 2,
		},
	}

	lines := buildCrossPatternIndexGroupLines([]string{"impl", "hotspot"}, fileMap, nil)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d (%+v)", len(lines), lines)
	}
	if lines[0] != "  src/handler.go" {
		t.Fatalf("unexpected first line: %q", lines[0])
	}
	if lines[1] != "  src/service.go (★2 patterns)" {
		t.Fatalf("unexpected second line: %q", lines[1])
	}

	reg := locator.NewRegistry()
	lines = buildCrossPatternIndexGroupLines([]string{"impl"}, fileMap, reg)
	if len(lines) != 1 || !strings.Contains(lines[0], "[L1]") {
		t.Fatalf("expected locator id in line, got %+v", lines)
	}
}
