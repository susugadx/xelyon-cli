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

func TestHasEffectivePatternList(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{pattern: "", want: false},
		{pattern: " , ", want: false},
		{pattern: `\,`, want: true},
		{pattern: "A, B", want: true},
	}

	for _, tt := range tests {
		if got := HasEffectivePatternList(tt.pattern); got != tt.want {
			t.Fatalf("HasEffectivePatternList(%q) = %v, want %v", tt.pattern, got, tt.want)
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

func TestShouldAppendImpactTestProbe(t *testing.T) {
	tests := []struct {
		name        string
		baseOutput  string
		basePattern []string
		want        bool
	}{
		{
			name:        "already has test coverage",
			baseOutput:  "foo_test.go:1:def TestFoo():",
			basePattern: []string{"Foo"},
			want:        false,
		},
		{
			name:        "expansion cap reached",
			baseOutput:  "foo.go:1:func Foo()",
			basePattern: []string{"a", "b", "c", "d", "e"},
			want:        false,
		},
		{
			name:        "eligible for probe append",
			baseOutput:  "foo.go:1:func Foo()",
			basePattern: []string{"Foo", "FooImpl"},
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldAppendImpactTestProbe(tt.baseOutput, tt.basePattern); got != tt.want {
				t.Fatalf("shouldAppendImpactTestProbe() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendImpactTestProbePattern(t *testing.T) {
	tests := []struct {
		name         string
		basePatterns []string
		pattern      string
		want         []string
	}{
		{
			name:         "appends missing conservative test probe",
			basePatterns: []string{"helper", "helperImpl"},
			pattern:      "helper",
			want:         []string{"helper", "helperImpl", "TestHelper"},
		},
		{
			name:         "skips existing test probe",
			basePatterns: []string{"helper", "TestHelper"},
			pattern:      "helper",
			want:         []string{"helper", "TestHelper"},
		},
		{
			name:         "empty pattern keeps base list",
			basePatterns: []string{"helper"},
			pattern:      "",
			want:         []string{"helper"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendImpactTestProbePattern(tt.basePatterns, tt.pattern)
			if len(got) != len(tt.want) {
				t.Fatalf("appendImpactTestProbePattern() len = %d, want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("appendImpactTestProbePattern()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoadCachedMultiPatternSearch(t *testing.T) {
	opts := SearchOptions{Pattern: "Run,Build", Path: t.TempDir()}
	contexts := newSinglePatternExecutionContexts([]string{"Run", "Build"}, opts)

	if got, ok := loadCachedMultiPatternSearch(nil, contexts, opts); ok || got != "" {
		t.Fatalf("expected nil cache miss, got hit=%v output=%q", ok, got)
	}

	cache := &testSearchCache{data: make(map[string]string)}
	cache.SetSearch(buildMultiCacheKeyFromContexts(contexts), buildMultiSearchCacheKeyFromContexts(opts, contexts), "cached-multi", nil)

	got, ok := loadCachedMultiPatternSearch(cache, contexts, opts)
	if !ok {
		t.Fatal("expected cached multi-pattern hit")
	}
	if got != "cached-multi" {
		t.Fatalf("unexpected cached multi output: %q", got)
	}
}

func TestExecuteMultipleSearchPatterns_UsesCacheHit(t *testing.T) {
	opts := SearchOptions{Pattern: "Run,Build", Path: t.TempDir()}
	contexts := newSinglePatternExecutionContexts([]string{"Run", "Build"}, opts)

	cache := &testSearchCache{data: make(map[string]string)}
	cache.SetSearch(buildMultiCacheKeyFromContexts(contexts), buildMultiSearchCacheKeyFromContexts(opts, contexts), "cached-dispatch", nil)

	if got := executeMultipleSearchPatterns(cache, contexts, opts); got != "cached-dispatch" {
		t.Fatalf("expected cached dispatch output, got %q", got)
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

func TestShouldExpandImpactSearchPatterns(t *testing.T) {
	tests := []struct {
		name     string
		opts     SearchOptions
		patterns []string
		want     bool
	}{
		{
			name:     "impact single pattern",
			opts:     SearchOptions{Intent: "impact"},
			patterns: []string{"Run"},
			want:     true,
		},
		{
			name:     "impact multi pattern",
			opts:     SearchOptions{Intent: "impact"},
			patterns: []string{"Run", "Build"},
			want:     false,
		},
		{
			name:     "non impact intent",
			opts:     SearchOptions{Intent: "search"},
			patterns: []string{"Run"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldExpandImpactSearchPatterns(tt.opts, tt.patterns); got != tt.want {
				t.Fatalf("shouldExpandImpactSearchPatterns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSearchPatternDispatch_PreparesContextsForMultiPattern(t *testing.T) {
	opts := SearchOptions{Pattern: "Run,Build", Path: t.TempDir()}
	dispatch := newSearchPatternDispatch([]string{"Run", "Build"}, opts)

	if len(dispatch.patterns) != 2 {
		t.Fatalf("patterns len = %d, want %d", len(dispatch.patterns), 2)
	}
	if len(dispatch.contexts) != 2 {
		t.Fatalf("contexts len = %d, want %d", len(dispatch.contexts), 2)
	}
	if dispatch.contexts[0].Pattern != "Run" || dispatch.contexts[1].Pattern != "Build" {
		t.Fatalf("unexpected contexts: %+v", dispatch.contexts)
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
