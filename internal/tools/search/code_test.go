package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func setupSearchTestMocks(t *testing.T) {
	t.Helper()
	tools.GlobalReadTracker.Reset()
}

// --- 統合テスト ---

func TestSearchCode_BasicMatch(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(file1, []byte("package main\n\nfunc hello() {\n\tfmt.Println(\"hello world\")\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode("hello", dir, "*.go", "0", "3000")

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches but got 'No matches found'")
	}
	if !strings.Contains(result, "hello.go") {
		t.Error("Expected file name in result")
	}
	if !strings.Contains(result, "match") {
		t.Error("Expected 'match' in result")
	}
	// マッチ行マーカー > が含まれること
	if !strings.Contains(result, ">") {
		t.Error("Expected '>' match marker in result")
	}
}

func TestSearchCode_NoMatch(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "test.go")
	if err := os.WriteFile(file1, []byte("package main\n\nfunc foo() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode("nonexistent_pattern_xyz", dir, "", "0", "3000")

	if result != "No matches found" {
		t.Errorf("Expected 'No matches found', got: %s", result)
	}
}

func TestSearchCode_EmptyPattern(t *testing.T) {
	setupSearchTestMocks(t)

	result := ExecuteSearchCode("", ".", "", "", "")

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

	result := ExecuteSearchCode("target", dir, "*.go", "0", "3000")

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

	// context_lines=0: コンテキスト行なし
	result := ExecuteSearchCode("target_match", dir, "", "0", "3000")

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	// マッチ行（target_match）は含まれる
	if !strings.Contains(result, "target_match") {
		t.Error("Expected 'target_match' in result")
	}
}

func TestSearchCode_TokenBudget(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	// 複数ファイルに多くのマッチを生成して確実にバジェット超過させる
	for i := 0; i < 10; i++ {
		var content strings.Builder
		for j := 0; j < 100; j++ {
			content.WriteString("var target_budget_check = \"this is a long value for token estimation\"\n")
		}
		fname := filepath.Join(dir, "budget"+strings.Repeat("x", i)+".go")
		if err := os.WriteFile(fname, []byte(content.String()), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// context_lines=3 + 小バジェット → 確実に打ち切り
	result := ExecuteSearchCode("target_budget_check", dir, "", "3", "500")

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	if !strings.Contains(result, "truncated") {
		t.Error("Expected truncation message with small token budget")
	}
}

func TestSearchCode_ReadTrackerIntegration(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "tracked.go")
	if err := os.WriteFile(file1, []byte("func tracked_func() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 検索前は未読
	absFile1, _ := filepath.Abs(file1)
	if tools.GlobalReadTracker.IsRead(absFile1) {
		t.Error("File should not be marked as read before search")
	}

	result := ExecuteSearchCode("tracked_func", dir, "", "0", "3000")

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}

	// 検索後は既読
	if !tools.GlobalReadTracker.IsRead(absFile1) {
		t.Error("File should be marked as read after search_code finds matches")
	}
}

func TestSearchCode_MergeOverlappingContext(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "overlap.go")
	// 近接する2つのマッチ（行2と行5）: context_lines=3 なら重複するコンテキスト行あり
	content := "line1\nmatch_one\nline3\nline4\nmatch_two\nline6\nline7\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode("match_", dir, "", "3", "3000")

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	// 両方のマッチが含まれること
	if !strings.Contains(result, "match_one") {
		t.Error("Expected 'match_one' in result")
	}
	if !strings.Contains(result, "match_two") {
		t.Error("Expected 'match_two' in result")
	}
	// 重複行番号が出力されていないことを確認
	lines := strings.Split(result, "\n")
	lineNums := make(map[string]int)
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.Contains(l, "│") {
			parts := strings.SplitN(l, "│", 2)
			numPart := strings.TrimSpace(strings.TrimPrefix(parts[0], ">"))
			lineNums[numPart]++
		}
	}
	for num, count := range lineNums {
		if count > 1 {
			t.Errorf("Line number %s appears %d times (should be merged)", num, count)
		}
	}
}

// --- パーサーユニットテスト ---

func TestParseRipgrepJSON(t *testing.T) {
	// ハードコードされた rg --json 出力
	input := `{"type":"begin","data":{"path":{"text":"src/main.go"}}}
{"type":"context","data":{"path":{"text":"src/main.go"},"line_number":4,"lines":{"text":"func setup() {\n"}}}
{"type":"match","data":{"path":{"text":"src/main.go"},"line_number":5,"lines":{"text":"\ttarget := \"hello\"\n"},"submatches":[{"match":{"text":"target"},"start":1,"end":7}]}}
{"type":"context","data":{"path":{"text":"src/main.go"},"line_number":6,"lines":{"text":"}\n"}}}
{"type":"end","data":{"path":{"text":"src/main.go"},"stats":{"elapsed":{"secs":0}}}}
{"type":"begin","data":{"path":{"text":"src/util.go"}}}
{"type":"match","data":{"path":{"text":"src/util.go"},"line_number":10,"lines":{"text":"var target = 42\n"},"submatches":[{"match":{"text":"target"},"start":4,"end":10}]}}
{"type":"end","data":{"path":{"text":"src/util.go"},"stats":{"elapsed":{"secs":0}}}}
{"type":"summary","data":{"elapsed_total":{"secs":0},"stats":{"searches":1}}}`

	results := parseRipgrepJSON(input)

	if len(results) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(results))
	}

	// File 1: src/main.go
	r1 := results[0]
	if r1.FilePath != "src/main.go" {
		t.Errorf("Expected file path 'src/main.go', got '%s'", r1.FilePath)
	}
	if r1.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r1.MatchCount)
	}
	if len(r1.Matches) != 3 { // 1 context + 1 match + 1 context
		t.Errorf("Expected 3 entries (context+match+context), got %d", len(r1.Matches))
	}
	// マッチ行の検証
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

	// File 2: src/util.go
	r2 := results[1]
	if r2.FilePath != "src/util.go" {
		t.Errorf("Expected file path 'src/util.go', got '%s'", r2.FilePath)
	}
	if r2.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r2.MatchCount)
	}
}

func TestParseGrepOutput(t *testing.T) {
	// ハードコードされた grep -rn -C 1 出力
	input := `src/main.go-4-func setup() {
src/main.go:5:	target := "hello"
src/main.go-6-}
--
src/util.go:10:var target = 42`

	results := parseGrepOutput(input)

	if len(results) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(results))
	}

	// File 1: src/main.go
	r1 := results[0]
	if r1.FilePath != "src/main.go" {
		t.Errorf("Expected file path 'src/main.go', got '%s'", r1.FilePath)
	}
	if r1.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", r1.MatchCount)
	}
	if len(r1.Matches) != 3 { // context + match + context
		t.Errorf("Expected 3 entries, got %d", len(r1.Matches))
	}

	// マッチ行の IsMatch 検証
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

	// File 2: src/util.go
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

	// テスト用の簡易キャッシュ実装
	cache := &testSearchCache{data: make(map[string]string)}
	origCache := tools.GlobalToolCache
	tools.GlobalToolCache = cache
	t.Cleanup(func() {
		tools.GlobalToolCache = origCache
	})

	// 1回目の検索
	result1 := ExecuteSearchCode("cached_target", dir, "", "0", "3000")
	if strings.Contains(result1, "No matches found") {
		t.Fatal("Expected matches on first search")
	}

	// キャッシュに保存されたか確認
	if cache.setCalls == 0 {
		t.Error("Expected SetSearch to be called after first search")
	}

	// 2回目の検索 — キャッシュヒット
	getCalls := cache.getCalls
	result2 := ExecuteSearchCode("cached_target", dir, "", "0", "3000")

	if cache.getCalls <= getCalls {
		t.Error("Expected GetSearch to be called on second search")
	}
	if result2 != result1 {
		t.Error("Expected same result from cache")
	}
}

// testSearchCache はテスト用のキャッシュ実装
type testSearchCache struct {
	data     map[string]string
	getCalls int
	setCalls int
}

func (c *testSearchCache) GetFile(path string) (string, bool) { return "", false }
func (c *testSearchCache) SetFile(path, content string)       {}
func (c *testSearchCache) GetDir(path string) (string, bool)  { return "", false }
func (c *testSearchCache) SetDir(path, result string)         {}
func (c *testSearchCache) InvalidateFile(path string)         {}
func (c *testSearchCache) InvalidateDir(path string)          {}
func (c *testSearchCache) Clear()                             {}
func (c *testSearchCache) ClearSearchCache()                  {}

func (c *testSearchCache) GetSearch(pattern, path string) (string, bool) {
	c.getCalls++
	key := pattern + "|" + path
	if v, ok := c.data[key]; ok {
		return v, true
	}
	return "", false
}

func (c *testSearchCache) SetSearch(pattern, path, result string) {
	c.setCalls++
	key := pattern + "|" + path
	c.data[key] = result
}
