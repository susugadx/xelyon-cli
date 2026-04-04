package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCode_ImpactIntentCacheRemainsValid(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "impact_cached.py")
	if err := os.WriteFile(file1, []byte("def NewAgent():\n    return 1\n\ndef NewAgentImpl():\n    return NewAgent()\n\ndef TestNewAgent():\n    return NewAgentImpl()\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: "NewAgent", Intent: "impact", Path: dir, FileType: "py", FilePattern: "*.py", IsRegex: true}

	result1 := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result1, "Pattern 1/") {
		t.Fatalf("expected multi-pattern result on first impact search, got:\n%s", result1)
	}
	if cache.setCalls == 0 {
		t.Fatal("expected cache write for impact search")
	}

	getCalls := cache.getCalls
	result2 := ExecuteSearchCodeWithCache(cache, opts)
	if cache.getCalls <= getCalls {
		t.Fatal("expected cache lookup on second impact search")
	}
	if result2 != result1 {
		t.Fatal("expected same cached result for repeated impact search")
	}
}

func TestSplitPatterns(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"  a , b ", []string{"a", "b"}},
		{"a,,b,", []string{"a", "b"}},
		{"a,b,c,d,e,f", []string{"a", "b", "c", "d", "e", "f"}},
		{"a,b,c,d,e,f,g,h,i,j,k", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
		{"single", []string{"single"}},
		{"", nil},
		{`a\,b,c`, []string{"a,b", "c"}},
		{`hello\,world`, []string{"hello,world"}},
	}
	for _, tt := range tests {
		got := splitPatterns(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("splitPatterns(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.expected, len(tt.expected))
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("splitPatterns(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestExpandImpactPatterns(t *testing.T) {
	got := expandImpactPatterns("NewAgent", SearchOptions{Intent: "impact"})
	want := []string{"NewAgent", "NewAgentImpl"}

	if len(got) != len(want) {
		t.Fatalf("expandImpactPatterns() len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expandImpactPatterns()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExpandImpactPatterns_DoesNotIncludeDuplicateProneTestNameVariants(t *testing.T) {
	got := expandImpactPatterns("Foo", SearchOptions{Intent: "impact"})
	for _, disallowed := range []string{"TestFoo", "FooTest", "Foo_test"} {
		for _, candidate := range got {
			if candidate == disallowed {
				t.Fatalf("expandImpactPatterns() should not include %q: %v", disallowed, got)
			}
		}
	}
}

func TestImpactTestProbePattern_UsesCommonTestSymbolFormConservatively(t *testing.T) {
	if got := impactTestProbePattern("helper"); got != "TestHelper" {
		t.Fatalf("impactTestProbePattern() = %q, want %q", got, "TestHelper")
	}
	if got := impactTestProbePattern("NewAgent"); got != "TestNewAgent" {
		t.Fatalf("impactTestProbePattern() = %q, want %q", got, "TestNewAgent")
	}
}

func TestEffectiveSearchPatterns_EmptyIntentPreservesSinglePattern(t *testing.T) {
	got := effectiveSearchPatterns(SearchOptions{Pattern: "NewAgent"})
	if len(got) != 1 || got[0] != "NewAgent" {
		t.Fatalf("effectiveSearchPatterns() = %v, want [NewAgent]", got)
	}
}

func TestEffectiveSearchPatterns_AlreadyMultiPatternNotExpandedTwice(t *testing.T) {
	got := effectiveSearchPatterns(SearchOptions{Pattern: "NewAgent,Run", Intent: "impact"})
	want := []string{"NewAgent", "Run"}
	if len(got) != len(want) {
		t.Fatalf("effectiveSearchPatterns() len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("effectiveSearchPatterns()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExecuteSearchCode_MultiplePatterns(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "multi.go")
	content := "func func_a() {}\nvar x = 1\nfunc func_b() {}\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "func_a,func_b", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if !strings.Contains(result, "Pattern") {
		t.Errorf("Expected multi-pattern header, got:\n%s", result)
	}
	if !strings.Contains(result, "Pattern 1/2") {
		t.Errorf("Expected 'Pattern 1/2' in result, got:\n%s", result)
	}
	if !strings.Contains(result, "Pattern 2/2") {
		t.Errorf("Expected 'Pattern 2/2' in result, got:\n%s", result)
	}
	if !strings.Contains(result, "func_a") {
		t.Error("Expected func_a match in result")
	}
	if !strings.Contains(result, "func_b") {
		t.Error("Expected func_b match in result")
	}
	if !strings.Contains(result, lineRangeHint) {
		t.Error("Expected line-range hint in multi-pattern result")
	}
}

func TestExecuteSearchCode_ImpactIntentExpandsSinglePattern(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "impact.py")
	content := "def NewAgent():\n    return 1\n\ndef NewAgentImpl():\n    return NewAgent()\n\ndef TestNewAgent():\n    return NewAgentImpl()\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Intent: "impact", Path: dir, FilePattern: "*.py", FileType: "py", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if !strings.Contains(result, "Pattern 1/") {
		t.Fatalf("expected multi-pattern output, got:\n%s", result)
	}
	if !strings.Contains(result, `"NewAgent"`) {
		t.Fatalf("expected base pattern header in output, got:\n%s", result)
	}
	if !strings.Contains(result, "TestNewAgent") {
		t.Fatalf("expected impact test probe in output, got:\n%s", result)
	}
}

func TestExecuteSearchCode_ImpactIntentFallsBackToCommonTestSymbolForm(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "impact_fallback.py")
	content := "def helper():\n    return 1\n\ndef helperImpl():\n    return helper()\n\ndef TestHelper():\n    return helperImpl()\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "helper", Intent: "impact", Path: dir, FilePattern: "*.py", FileType: "py", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if !strings.Contains(result, "TestHelper") {
		t.Fatalf("expected conservative fallback test probe to surface TestHelper, got:\n%s", result)
	}
}

func TestBuildSearchCacheKeyWithRoute_SeparatesIntent(t *testing.T) {
	opts := SearchOptions{Pattern: "Run", Path: ".", FileType: "go"}
	impactOpts := SearchOptions{Pattern: "Run", Intent: "impact", Path: ".", FileType: "go"}

	normalKey := buildSearchCacheKeyWithRoute(opts, "route")
	impactKey := buildSearchCacheKeyWithRoute(impactOpts, "route")

	if normalKey == impactKey {
		t.Fatalf("expected cache key to differ by intent, got %q", normalKey)
	}
}

func TestBuildMultiSearchCacheKey_SeparatesIntent(t *testing.T) {
	opts := SearchOptions{Pattern: "Run,Build", Path: ".", FileType: "go"}
	impactOpts := SearchOptions{Pattern: "Run,Build", Intent: "impact", Path: ".", FileType: "go"}
	patterns := []string{"Run", "Build"}

	normalKey := buildMultiSearchCacheKey(opts, patterns)
	impactKey := buildMultiSearchCacheKey(impactOpts, patterns)

	if normalKey == impactKey {
		t.Fatalf("expected multi cache key to differ by intent, got %q", normalKey)
	}
}
