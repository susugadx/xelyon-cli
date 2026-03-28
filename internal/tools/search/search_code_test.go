package search

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func setupSearchTestMocks(t *testing.T) {
	t.Helper()
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
}

func TestSearchCode_BasicMatch(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(file1, []byte("package main\n\nfunc hello() {\n\tfmt.Println(\"hello world\")\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "hello", Path: dir, FilePattern: "*.go", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches but got 'No matches found'")
	}
	if !strings.Contains(result, "hello.go") {
		t.Error("Expected file name in result")
	}
	if !strings.Contains(result, "match") {
		t.Error("Expected 'match' in result")
	}
	if !strings.Contains(result, ">") {
		t.Error("Expected '>' match marker in result")
	}
	if !strings.Contains(result, lineRangeHint) {
		t.Error("Expected line-range hint in normal search result")
	}
}

func TestSearchCode_NoMatch(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "test.go")
	if err := os.WriteFile(file1, []byte("package main\n\nfunc foo() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "nonexistent_pattern_xyz", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if !strings.Contains(result, "No matches found") {
		t.Errorf("Expected 'No matches found' in result, got: %s", result)
	}
	if strings.Contains(result, lineRangeHint) {
		t.Error("No-match result should not include line-range hint")
	}
}

func TestSearchCode_EmptyPattern(t *testing.T) {
	setupSearchTestMocks(t)

	result := ExecuteSearchCode(SearchOptions{Pattern: "", Path: ".", FilePattern: "", FileType: "", IsRegex: true, Multiline: false})

	if !strings.Contains(result, "Error") {
		t.Errorf("Expected error for empty pattern, got: %s", result)
	}
}

func TestSearchCode_FilePattern(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	goFile := filepath.Join(dir, "test.go")
	jsFile := filepath.Join(dir, "test.js")
	if err := os.WriteFile(goFile, []byte("func target() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsFile, []byte("function target() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "target", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	if !strings.Contains(result, "test.go") {
		t.Error("Expected test.go in result")
	}
	if strings.Contains(result, "test.js") {
		t.Error("Should not contain test.js when file_pattern=*.go")
	}
}

func TestSearchCode_ContextLines(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "ctx.go")
	content := "line1\nline2\nline3\ntarget_match\nline5\nline6\nline7\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "target_match", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	if !strings.Contains(result, "target_match") {
		t.Error("Expected 'target_match' in result")
	}
}

func TestSearchCode_TokenBudget(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	for i := 0; i < 2; i++ {
		var content strings.Builder
		for j := 0; j < 50; j++ {
			content.WriteString("var target_budget_check = \"" + strings.Repeat("a", 240) + "\"\n")
		}
		fname := filepath.Join(dir, "budget"+string(rune('a'+i))+".go")
		if err := os.WriteFile(fname, []byte(content.String()), 0644); err != nil {
			t.Fatal(err)
		}
	}

	longLine := "var long_target_budget_check = \"" + strings.Repeat("a", 1000) + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "long.go"), []byte(longLine), 0644); err != nil {
		t.Fatal(err)
	}

	japaneseLine := "// target_budget_check " + strings.Repeat("あ", 100) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "japanese.go"), []byte(japaneseLine), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "target_budget_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 3, TokenBudget: 500, IsRegex: true, Multiline: false})
	resultWithLargeBudget := ExecuteSearchCode(SearchOptions{Pattern: "target_budget_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 3, TokenBudget: 60000, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	if !strings.Contains(result, "truncated") {
		t.Error("Expected truncation message from the internal safety valve")
	}
	if !strings.Contains(resultWithLargeBudget, "truncated") {
		t.Error("Expected truncation message even when a larger external token_budget is provided")
	}

	longResult := ExecuteSearchCode(SearchOptions{Pattern: "long_target_budget_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if !strings.Contains(longResult, strings.Repeat("a", 1000)) {
		t.Error("Expected long line to be preserved without line truncation")
	}

	jpResult := ExecuteSearchCode(SearchOptions{Pattern: "target_budget_check あ", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if !strings.Contains(jpResult, "target_budget_check") {
		t.Error("Expected japanese comment to be searchable and displayed")
	}
}

func TestSearchCode_MergeOverlappingContext(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "overlap.go")
	content := "line1\nmatch_one\nline3\nline4\nmatch_two\nline6\nline7\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "match_", Path: dir, FilePattern: "", FileType: "", CtxLines: 3, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	if !strings.Contains(result, "match_one") {
		t.Error("Expected 'match_one' in result")
	}
	if !strings.Contains(result, "match_two") {
		t.Error("Expected 'match_two' in result")
	}
	lines := strings.Split(result, "\n")
	lineNums := make(map[string]int)
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.Contains(l, "│") {
			parts := strings.SplitN(l, "│", 2)
			numPart := parts[0]
			for _, tag := range []string{"[def]", "[import]", "[call]", "[assign]", "[ref]", "[comment]", "[string]", ">"} {
				numPart = strings.ReplaceAll(numPart, tag, "")
			}
			numPart = strings.TrimSpace(numPart)
			lineNums[numPart]++
		}
	}
	for num, count := range lineNums {
		if count > 1 {
			t.Errorf("Line number %s appears %d times (should be merged)", num, count)
		}
	}
}

func TestParseRipgrepJSON(t *testing.T) {
	input := `{"type":"begin","data":{"path":{"text":"src/main.go"}}}
{"type":"context","data":{"path":{"text":"src/main.go"},"line_number":4,"lines":{"text":"func setup() {\n"}}}
{"type":"match","data":{"path":{"text":"src/main.go"},"line_number":5,"lines":{"text":"\ttarget := \"hello\"\n"},"submatches":[{"match":{"text":"target"},"start":1,"end":7}]}}
{"type":"context","data":{"path":{"text":"src/main.go"},"line_number":6,"lines":{"text":"}\n"}}}
{"type":"end","data":{"path":{"text":"src/main.go"},"stats":{"elapsed":{"secs":0}}}}
{"type":"begin","data":{"path":{"text":"src/util.go"}}}
{"type":"match","data":{"path":{"text":"src/util.go"},"line_number":10,"lines":{"text":"var target = 42\n"},"submatches":[{"match":{"text":"target"},"start":4,"end":10}]}}
{"type":"end","data":{"path":{"text":"src/util.go"},"stats":{"elapsed":{"secs":0}}}}
{"type":"summary","data":{"elapsed_total":{"secs":0},"stats":{"searches":1}}}`

	results := parseRipgrepJSON(input, 200)

	if len(results) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(results))
	}

	r1 := results[0]
	if r1.FilePath != "src/main.go" {
		t.Errorf("Expected file path 'src/main.go', got '%s'", r1.FilePath)
	}
	if r1.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r1.MatchCount)
	}
	if len(r1.Matches) != 3 {
		t.Errorf("Expected 3 entries (context+match+context), got %d", len(r1.Matches))
	}
	found := false
	for _, m := range r1.Matches {
		if m.IsMatch && m.LineNum == 5 {
			found = true
			if !strings.Contains(m.Line, "target") {
				t.Errorf("Match line should contain 'target', got: %s", m.Line)
			}
		}
	}
	if !found {
		t.Error("Expected match at line 5")
	}

	r2 := results[1]
	if r2.FilePath != "src/util.go" {
		t.Errorf("Expected file path 'src/util.go', got '%s'", r2.FilePath)
	}
	if r2.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r2.MatchCount)
	}
}

func TestParseGrepOutput(t *testing.T) {
	input := `src/main.go-4-func setup() {
src/main.go:5:	target := "hello"
src/main.go-6-}
--
src/util.go:10:var target = 42`

	results := parseGrepOutput(input, 200)

	if len(results) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(results))
	}

	r1 := results[0]
	if r1.FilePath != "src/main.go" {
		t.Errorf("Expected file path 'src/main.go', got '%s'", r1.FilePath)
	}
	if r1.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r1.MatchCount)
	}
	if len(r1.Matches) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(r1.Matches))
	}

	matchFound := false
	for _, m := range r1.Matches {
		if m.LineNum == 5 && m.IsMatch {
			matchFound = true
		}
		if m.LineNum == 4 && m.IsMatch {
			t.Error("Line 4 should be context (IsMatch=false)")
		}
	}
	if !matchFound {
		t.Error("Expected match at line 5 with IsMatch=true")
	}

	r2 := results[1]
	if r2.FilePath != "src/util.go" {
		t.Errorf("Expected file path 'src/util.go', got '%s'", r2.FilePath)
	}
	if r2.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r2.MatchCount)
	}
}

func TestSearchCode_CacheHitAfterSearch(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "cached.go")
	if err := os.WriteFile(file1, []byte("func cached_target() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cache := &testSearchCache{data: make(map[string]string)}

	result1 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if strings.Contains(result1, "No matches found") {
		t.Fatal("Expected matches on first search")
	}

	if cache.setCalls == 0 {
		t.Error("Expected SetSearch to be called after first search")
	}

	getCalls := cache.getCalls
	result2 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if cache.getCalls <= getCalls {
		t.Error("Expected GetSearch to be called on second search")
	}
	if result2 != result1 {
		t.Error("Expected same result from cache")
	}
}

func TestSearchCode_CacheDifferentParams(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "params.go")
	content := "line1\nline2\ntarget_param_check\nline4\nline5\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cache := &testSearchCache{data: make(map[string]string)}

	resultDefault := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "target_param_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if strings.Contains(resultDefault, "No matches found") {
		t.Fatal("Expected matches with default output mode")
	}

	resultManifest := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "target_param_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 3, TokenBudget: 3000, IsRegex: true, Multiline: false, OutputMode: "manifest"})
	if strings.Contains(resultManifest, "No matches found") {
		t.Fatal("Expected matches with manifest output mode")
	}

	if resultDefault == resultManifest {
		t.Error("Results with different output_mode should differ (cache key should include output_mode)")
	}
}

type testSearchCache struct {
	mu          sync.Mutex
	data        map[string]string
	affected    map[string][]string
	dataKeys    map[string]string
	getCalls    int
	setCalls    int
	lastGetPath string
	lastSetPath string
}

func (c *testSearchCache) GetFile(path string) (string, bool) { return "", false }
func (c *testSearchCache) SetFile(path, content string)       {}
func (c *testSearchCache) GetDir(path string) (string, bool)  { return "", false }
func (c *testSearchCache) SetDir(path, result string)         {}
func (c *testSearchCache) InvalidateFile(path string)         {}
func (c *testSearchCache) InvalidateDir(path string)          {}
func (c *testSearchCache) Clear()                             {}
func (c *testSearchCache) ClearSearchCache()                  { tools.NotifySearchCacheCleared() }
func (c *testSearchCache) InvalidateSearchCacheForFile(absPath string) {
	c.mu.Lock()
	deletedKeys := make([]string, 0)
	deleted := false
	for key, files := range c.affected {
		for _, fp := range files {
			if fp == absPath {
				if dataKey, ok := c.dataKeys[key]; ok {
					delete(c.data, dataKey)
					delete(c.dataKeys, key)
				}
				delete(c.affected, key)
				deleted = true
				deletedKeys = append(deletedKeys, key)
				break
			}
		}
	}
	c.mu.Unlock()
	if deleted {
		tools.NotifySearchCacheInvalidatedKeys(deletedKeys)
	}
}

func (c *testSearchCache) GetSearch(pattern, path string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getCalls++
	c.lastGetPath = path
	key := pattern + "|" + path
	if v, ok := c.data[key]; ok {
		return v, true
	}
	return "", false
}

func (c *testSearchCache) SetSearch(pattern, path, result string, affectedFiles []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setCalls++
	c.lastSetPath = path
	key := pattern + "|" + path
	c.data[key] = result
	if c.affected == nil {
		c.affected = make(map[string][]string)
	}
	if c.dataKeys == nil {
		c.dataKeys = make(map[string]string)
	}
	searchKey := singlePatternBundleCacheKey(pattern, path)
	c.affected[searchKey] = append([]string(nil), affectedFiles...)
	c.dataKeys[searchKey] = key
}

func TestSearchCode_CacheKeyUsesInternalTokenBudget(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "cached.go")
	if err := os.WriteFile(file1, []byte("func cached_target() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cache := &testSearchCache{data: make(map[string]string)}

	result1 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, CtxLines: 0, TokenBudget: 500, IsRegex: true})
	if strings.Contains(result1, "No matches found") {
		t.Fatal("Expected matches on first search")
	}
	if !strings.Contains(cache.lastSetPath, "|3|15000|") {
		t.Fatalf("expected cache key to use internal defaults for context_lines=3 and token_budget=15000, got: %s", cache.lastSetPath)
	}

	result2 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, CtxLines: 0, TokenBudget: 99999, IsRegex: true})
	if result2 != result1 {
		t.Fatal("Expected second result to be served from the same cache key")
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected one cache write with normalized token budget, got %d", cache.setCalls)
	}
	if !strings.Contains(cache.lastGetPath, "|3|15000|") {
		t.Fatalf("expected cache lookup key to use internal defaults for context_lines=3 and token_budget=15000, got: %s", cache.lastGetPath)
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

func TestExecuteSearchCode_MultiplePatterns_PartialMatch(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "partial.go")
	content := "func existing_func() {}\nvar x = 1\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "existing_func,nonexistent_xyz_pattern", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if !strings.Contains(result, "existing_func") {
		t.Error("Expected existing_func match in result")
	}
	if !strings.Contains(result, "No matches found") {
		t.Error("Expected 'No matches found' for unmatched pattern")
	}
}

func TestSearchCode_ResultRanking(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "rank.go")
	content := "fmt.Println(target)\nfunc target() {}\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "target", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}

	defIdx := strings.Index(result, "[def]")
	refIdx := strings.Index(result, "[ref]")
	if defIdx < 0 {
		t.Error("Expected [def] tag in result")
	}
	if refIdx < 0 {
		t.Error("Expected [ref] tag in result")
	}
	if defIdx >= 0 && refIdx >= 0 && defIdx >= refIdx {
		t.Errorf("Expected [def] before [ref], but [def] at %d, [ref] at %d", defIdx, refIdx)
	}
}

func TestSearchCode_BlockDetection(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "blocks.go")
	content := `package main

func setup() {
	target_var := 1
	fmt.Println(target_var)
}

func process() {
	x := 2
}

func cleanup() {
	target_var := 3
}
`
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "target_var", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	if !strings.Contains(result, "── in") {
		t.Errorf("Expected block annotation '── in' in result, got:\n%s", result)
	}

	hasSetup := strings.Contains(result, "func setup")
	hasCleanup := strings.Contains(result, "func cleanup")
	if !hasSetup && !hasCleanup {
		t.Errorf("Expected block annotation for func setup or func cleanup, got:\n%s", result)
	}
}

func TestSearchCode_InvalidRegex(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "test.go")
	if err := os.WriteFile(file1, []byte("func hello() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "func(", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if result == "No matches found" {
		t.Error("Invalid regex should NOT return 'No matches found' — should return error message")
	}
	if !strings.Contains(result, "Error") {
		t.Errorf("Expected error message for invalid regex, got: %s", result)
	}
}

func TestSearchCode_FileTypePreferred(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	goFile := filepath.Join(dir, "typed.go")
	jsFile := filepath.Join(dir, "typed.js")
	if err := os.WriteFile(goFile, []byte("func typedTarget() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsFile, []byte("function typedTarget() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "typedTarget", Path: dir, FilePattern: "*.js", FileType: "go", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if !strings.Contains(result, "typed.go") {
		t.Fatalf("expected go file in result, got: %s", result)
	}
	if strings.Contains(result, "typed.js") {
		t.Fatalf("file_type should take precedence over file_pattern, got: %s", result)
	}
}

func TestSearchCode_FixedStrings(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "fixed.go")
	if err := os.WriteFile(file1, []byte("var name = \"a+b\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "a+b", Mode: string(SearchModeLiteral), Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: false, Multiline: false})
	if strings.Contains(result, "No matches found") || !strings.Contains(result, "a+b") {
		t.Fatalf("expected literal match with is_regex=false, got: %s", result)
	}
}

func TestSearchCode_IncludeHidden(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=hidden_value\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "hidden_value",
		Path:        dir,
		IsRegex:     true,
		CtxLines:    -1,
		TokenBudget: -1,
	})
	if strings.Contains(result, ".env") {
		t.Fatalf("hidden files should be excluded by default, got: %s", result)
	}

	result = ExecuteSearchCode(SearchOptions{
		Pattern:       "hidden_value",
		Path:          dir,
		IsRegex:       true,
		IncludeHidden: true,
		CtxLines:      -1,
		TokenBudget:   -1,
	})
	if !strings.Contains(result, ".env") {
		t.Fatalf("hidden files should be included with IncludeHidden, got: %s", result)
	}
}

func TestSearchCode_GrepFallback_DoesNotExcludeRootDot(t *testing.T) {
	setupSearchTestMocks(t)

	if runtime.GOOS == "windows" {
		t.Skip("grep fallback regression test is linux/mac specific")
	}

	grepPath, err := exec.LookPath("grep")
	if err != nil {
		t.Skip("grep not available")
	}

	binDir := t.TempDir()
	if err := os.Symlink(grepPath, filepath.Join(binDir, "grep")); err != nil {
		t.Skipf("failed to prepare isolated grep PATH: %v", err)
	}

	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldwd)
	})

	t.Setenv("PATH", binDir)

	file1 := filepath.Join(dir, "search_target.go")
	if err := os.WriteFile(file1, []byte("package main\n\nfunc maybeAutoCompress() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "maybeAutoCompress",
		Path:        ".",
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     true,
		Multiline:   false,
	})

	if strings.Contains(result, "No matches found") {
		t.Fatalf("expected grep fallback to find match from root dot, got: %s", result)
	}
	if strings.Contains(result, "Warning: ripgrep (rg) not found; using grep fallback mode.") {
		t.Fatalf("unexpected per-call grep fallback warning, got: %s", result)
	}
	if !strings.Contains(result, "search_target.go") {
		t.Fatalf("expected file name in result, got: %s", result)
	}
}

func TestSearchCode_TypeToGlobMapping(t *testing.T) {
	tests := []struct {
		fileType string
		wantGlob string
		wantOK   bool
	}{
		{fileType: "go", wantGlob: "*.go", wantOK: true},
		{fileType: "py", wantGlob: "*.py", wantOK: true},
		{fileType: "rust", wantGlob: "*.rs", wantOK: true},
		{fileType: "unknown", wantGlob: "", wantOK: false},
	}

	for _, tt := range tests {
		got, ok := fileTypeToGlob(tt.fileType)
		if ok != tt.wantOK || got != tt.wantGlob {
			t.Fatalf("fileTypeToGlob(%q) = (%q, %v), want (%q, %v)", tt.fileType, got, ok, tt.wantGlob, tt.wantOK)
		}
	}
}

func TestSearchCode_ManifestMode(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(file1, []byte("package main\n\nfunc hello() {\n\tfmt.Println(\"hello world\")\n}\n\nfunc goodbye() {\n\tfmt.Println(\"hello again\")\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "hello",
		Path:        dir,
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     true,
		OutputMode:  "manifest",
	})

	if !strings.Contains(result, "matches") {
		t.Errorf("expected manifest output with 'matches', got:\n%s", result)
	}
	if strings.Contains(result, "Println") {
		t.Errorf("manifest mode should not include code snippets, got:\n%s", result)
	}
	if strings.Contains(result, "│") {
		t.Errorf("manifest mode should not contain │ separator, got:\n%s", result)
	}
	if strings.Contains(result, lineRangeHint) {
		t.Errorf("manifest mode should not include line-range hint, got:\n%s", result)
	}
}

func TestExecuteSearchCodeWithConfig_ProjectIgnorePatterns(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - generated\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "generated"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "generated", "skip.go"), []byte("package generated\n\nfunc target() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.go"), []byte("package main\n\nfunc target() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	result := ExecuteSearchCodeWithConfig(config.DefaultConfig(), nil, SearchOptions{
		Pattern:     "target",
		Path:        dir,
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     true,
	})

	if strings.Contains(result, "generated/skip.go") {
		t.Fatalf("generated/skip.go should be ignored by xelyon.yaml ignore.patterns, got %q", result)
	}
	if !strings.Contains(result, "keep.go") {
		t.Fatalf("keep.go should be included, got %q", result)
	}
}

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
	// 3 unique files → 3 locator IDs
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
	// 1 unique file in 1 category, no hotspot → should be suppressed
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
	// 1 unique file but it's a hotspot → should be shown
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
