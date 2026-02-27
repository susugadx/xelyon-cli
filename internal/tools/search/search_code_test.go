package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/embedding"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
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
	if tools.GlobalReadTracker.IsReadLine(absFile1, 1) {
		t.Error("Line should not be marked as read before search")
	}

	result := ExecuteSearchCode("tracked_func", dir, "", "0", "3000")

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}

	// 検索後: ファイル全体は未読だが、マッチ行範囲は既読
	if tools.GlobalReadTracker.IsRead(absFile1) {
		t.Error("File should NOT be fully marked as read after search_code (only line ranges)")
	}
	// マッチ行（tracked_func は1行目）は既読
	if !tools.GlobalReadTracker.IsReadLine(absFile1, 1) {
		t.Error("Match line should be marked as read after search_code")
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
			numPart := parts[0]
			// タグとマッチマーカーを除去して行番号のみ抽出
			for _, tag := range []string{"[def]", "[import]", "[assign]", "[use]", "[comment]", ">"} {
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

	results := parseRipgrepJSON(input, 200)

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

	results := parseGrepOutput(input, 200)

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

func TestSearchCode_CacheDifferentParams(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "params.go")
	// コンテキスト行の有無で結果が変わるようにマッチ前後に行を配置
	content := "line1\nline2\ntarget_param_check\nline4\nline5\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cache := &testSearchCache{data: make(map[string]string)}
	origCache := tools.GlobalToolCache
	tools.GlobalToolCache = cache
	t.Cleanup(func() {
		tools.GlobalToolCache = origCache
	})

	// context_lines=0 で検索
	result0 := ExecuteSearchCode("target_param_check", dir, "", "0", "3000")
	if strings.Contains(result0, "No matches found") {
		t.Fatal("Expected matches with context_lines=0")
	}

	// context_lines=3 で検索 — キャッシュキーが異なるため別結果が返るべき
	result3 := ExecuteSearchCode("target_param_check", dir, "", "3", "3000")
	if strings.Contains(result3, "No matches found") {
		t.Fatal("Expected matches with context_lines=3")
	}

	// context_lines=3 の結果にはコンテキスト行（line1, line2 等）が含まれるはず
	// context_lines=0 の結果には含まれない
	if result0 == result3 {
		t.Error("Results with different context_lines should differ (cache key should include context_lines)")
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

// --- splitPatterns テスト ---

func TestSplitPatterns(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"  a , b ", []string{"a", "b"}},
		{"a,,b,", []string{"a", "b"}},
		{"a,b,c,d,e,f", []string{"a", "b", "c", "d", "e"}}, // 上限5
		{"single", []string{"single"}},
		{"", nil},
		{`a\,b,c`, []string{"a,b", "c"}},          // エスケープカンマ
		{`hello\,world`, []string{"hello,world"}}, // 単一パターン内のカンマ
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

// --- 複数パターン検索テスト ---

func TestExecuteSearchCode_MultiplePatterns(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "multi.go")
	content := "func func_a() {}\nvar x = 1\nfunc func_b() {}\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode("func_a,func_b", dir, "*.go", "0", "3000")

	// 複数パターンヘッダー
	if !strings.Contains(result, "patterns") {
		t.Errorf("Expected multi-pattern header, got:\n%s", result)
	}
	// Pattern 1/2 と Pattern 2/2
	if !strings.Contains(result, "Pattern 1/2") {
		t.Errorf("Expected 'Pattern 1/2' in result, got:\n%s", result)
	}
	if !strings.Contains(result, "Pattern 2/2") {
		t.Errorf("Expected 'Pattern 2/2' in result, got:\n%s", result)
	}
	// 各パターンのマッチ
	if !strings.Contains(result, "func_a") {
		t.Error("Expected func_a match in result")
	}
	if !strings.Contains(result, "func_b") {
		t.Error("Expected func_b match in result")
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

	result := ExecuteSearchCode("existing_func,nonexistent_xyz_pattern", dir, "*.go", "0", "3000")

	// マッチしたパターンの結果が含まれる
	if !strings.Contains(result, "existing_func") {
		t.Error("Expected existing_func match in result")
	}
	// マッチしないパターンは "No matches found"
	if !strings.Contains(result, "No matches found") {
		t.Error("Expected 'No matches found' for unmatched pattern")
	}
}

func TestExecuteSearchCode_MultiplePatterns_ReadTracker(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "tracker_multi.go")
	content := "func track_a() {}\nvar x = 1\nfunc track_b() {}\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	absFile1, _ := filepath.Abs(file1)

	// 検索前は未読
	if tools.GlobalReadTracker.IsReadLine(absFile1, 1) {
		t.Error("Line 1 should not be marked as read before search")
	}

	result := ExecuteSearchCode("track_a,track_b", dir, "*.go", "0", "3000")

	if !strings.Contains(result, "Pattern 1/2") {
		t.Fatalf("Expected multi-pattern result, got:\n%s", result)
	}

	// track_a は1行目、track_b は3行目 — 両方が ReadTracker に登録される
	if !tools.GlobalReadTracker.IsReadLine(absFile1, 1) {
		t.Error("Line 1 (track_a) should be marked as read after multi-pattern search")
	}
	if !tools.GlobalReadTracker.IsReadLine(absFile1, 3) {
		t.Error("Line 3 (track_b) should be marked as read after multi-pattern search")
	}
}

// --- マッチ種別分類テスト ---

func TestClassifyMatch(t *testing.T) {
	tests := []struct {
		line     string
		expected MatchType
	}{
		// Definition (top-level)
		{"func hello() {}", MatchTypeDefinition},
		{"type Config struct {", MatchTypeDefinition},
		{"class MyClass:", MatchTypeDefinition},
		{"def process(data):", MatchTypeDefinition},
		// Definition (indented — Python methods, nested classes, etc.)
		{"    def method(self):", MatchTypeDefinition},
		{"    class InnerClass:", MatchTypeDefinition},
		{"        func nested() {", MatchTypeDefinition},
		{"    var localVar = 1", MatchTypeDefinition},
		// Import
		{"import fmt", MatchTypeImport},
		{"from os import path", MatchTypeImport},
		{`const { useState } = require('react')`, MatchTypeImport},
		// Assignment
		{"x := 42", MatchTypeAssignment},
		{"self.value = data", MatchTypeAssignment},
		// Usage
		{"if x == 1 {", MatchTypeUsage},
		{"fmt.Println(target)", MatchTypeUsage},
		{`fmt.Println("hello=world")`, MatchTypeUsage},
		{`url := "https://example.com?key=value"`, MatchTypeAssignment},
		// Usage (struct tag with = inside backtick — NOT assignment)
		{"\tName string `json:\"name\"`", MatchTypeUsage},
		{"\tValue int `yaml:\"val=default\"`", MatchTypeUsage},
		// Comment
		{"// comment", MatchTypeComment},
		{"# python comment", MatchTypeComment},
	}
	for _, tt := range tests {
		got := classifyMatch(tt.line)
		if got != tt.expected {
			t.Errorf("classifyMatch(%q) = %d, want %d", tt.line, got, tt.expected)
		}
	}
}

func TestSearchCode_ResultRanking(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "rank.go")
	// 使用 → 定義 の順で配置（ソート後は定義が先に来るべき）
	content := "fmt.Println(target)\nfunc target() {}\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode("target", dir, "*.go", "0", "3000")

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}

	defIdx := strings.Index(result, "[def]")
	useIdx := strings.Index(result, "[use]")
	if defIdx < 0 {
		t.Error("Expected [def] tag in result")
	}
	if useIdx < 0 {
		t.Error("Expected [use] tag in result")
	}
	if defIdx >= 0 && useIdx >= 0 && defIdx >= useIdx {
		t.Errorf("Expected [def] before [use], but [def] at %d, [use] at %d", defIdx, useIdx)
	}
}

// --- ヘルパー関数ユニットテスト ---

func TestBuildMatchBlocks(t *testing.T) {
	matches := []Match{
		{LineNum: 1, Line: "ctx1", IsMatch: false, Type: MatchTypeUsage},
		{LineNum: 2, Line: "match1", IsMatch: true, Type: MatchTypeDefinition},
		{LineNum: 3, Line: "ctx2", IsMatch: false, Type: MatchTypeUsage},
		{LineNum: 4, Line: "ctx3", IsMatch: false, Type: MatchTypeUsage},
		{LineNum: 5, Line: "match2", IsMatch: true, Type: MatchTypeUsage},
		{LineNum: 6, Line: "ctx4", IsMatch: false, Type: MatchTypeUsage},
	}

	blocks := buildMatchBlocks(matches)

	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(blocks))
	}

	// Block 1: [ctx1, match1] typ=Definition
	if blocks[0].typ != MatchTypeDefinition {
		t.Errorf("Block 0 typ = %d, want %d", blocks[0].typ, MatchTypeDefinition)
	}
	if len(blocks[0].matches) != 2 {
		t.Errorf("Block 0 has %d matches, want 2", len(blocks[0].matches))
	}

	// Block 2: [ctx2, ctx3, match2, ctx4] typ=Usage
	if blocks[1].typ != MatchTypeUsage {
		t.Errorf("Block 1 typ = %d, want %d", blocks[1].typ, MatchTypeUsage)
	}
	if len(blocks[1].matches) != 4 {
		t.Errorf("Block 1 has %d matches, want 4", len(blocks[1].matches))
	}
}

func TestFindBlockForLine(t *testing.T) {
	ranges := []common.BlockRange{
		{Name: "func outer", StartLine: 1, EndLine: 20},
		{Name: "func inner", StartLine: 5, EndLine: 10},
	}

	// 最内ブロック優先
	b := findBlockForLine(ranges, 7)
	if b == nil || b.Name != "func inner" {
		t.Errorf("Expected innermost block 'func inner' for line 7, got: %v", b)
	}

	// outer のみ
	b = findBlockForLine(ranges, 15)
	if b == nil || b.Name != "func outer" {
		t.Errorf("Expected 'func outer' for line 15, got: %v", b)
	}

	// 範囲外
	b = findBlockForLine(ranges, 25)
	if b != nil {
		t.Errorf("Expected nil for line 25, got: %v", b)
	}

	// 空 ranges
	b = findBlockForLine(nil, 1)
	if b != nil {
		t.Errorf("Expected nil for empty ranges, got: %v", b)
	}

	// 境界値: ブロック開始行
	b = findBlockForLine(ranges, 5)
	if b == nil || b.Name != "func inner" {
		t.Errorf("Expected 'func inner' for start line 5, got: %v", b)
	}

	// 境界値: ブロック終了行
	b = findBlockForLine(ranges, 10)
	if b == nil || b.Name != "func inner" {
		t.Errorf("Expected 'func inner' for end line 10, got: %v", b)
	}
}

// --- ブロック認識テスト ---

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

	result := ExecuteSearchCode("target_var", dir, "*.go", "0", "3000")

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}

	// ブロック注釈が含まれること
	if !strings.Contains(result, "── in") {
		t.Errorf("Expected block annotation '── in' in result, got:\n%s", result)
	}

	// func setup または func cleanup のブロック注釈が含まれること
	hasSetup := strings.Contains(result, "func setup")
	hasCleanup := strings.Contains(result, "func cleanup")
	if !hasSetup && !hasCleanup {
		t.Errorf("Expected block annotation for func setup or func cleanup, got:\n%s", result)
	}
}

// --- ファイルソート順テスト ---

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.go", false},
		{"main_test.go", true},
		{"src/handler.go", false},
		{"src/handler_test.go", true},
		{"app.test.js", true},
		{"app.test.ts", true},
		{"app.spec.js", true},
		{"app.spec.ts", true},
		{"test_helper.py", true},
		{"utils.py", false},
	}
	for _, tt := range tests {
		got := isTestFile(tt.path)
		if got != tt.expected {
			t.Errorf("isTestFile(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestSortResultsByPriority(t *testing.T) {
	results := []SearchResult{
		{FilePath: "handler_test.go", Matches: []Match{{IsMatch: true, Type: MatchTypeUsage}}},
		{FilePath: "handler.go", Matches: []Match{{IsMatch: true, Type: MatchTypeUsage}}},
	}
	sortResultsByPriority(results)

	if results[0].FilePath != "handler.go" {
		t.Errorf("Expected non-test file first, got %s", results[0].FilePath)
	}
	if results[1].FilePath != "handler_test.go" {
		t.Errorf("Expected test file second, got %s", results[1].FilePath)
	}
}

func TestSortResultsByPriority_DefinitionFirst(t *testing.T) {
	results := []SearchResult{
		{FilePath: "caller.go", Matches: []Match{{IsMatch: true, Type: MatchTypeUsage}}},
		{FilePath: "define.go", Matches: []Match{{IsMatch: true, Type: MatchTypeDefinition}}},
	}
	sortResultsByPriority(results)

	if results[0].FilePath != "define.go" {
		t.Errorf("Expected definition file first, got %s", results[0].FilePath)
	}
	if results[1].FilePath != "caller.go" {
		t.Errorf("Expected usage-only file second, got %s", results[1].FilePath)
	}
}

// --- 不正 regex テスト ---

func TestSearchCode_InvalidRegex(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "test.go")
	if err := os.WriteFile(file1, []byte("func hello() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 不正な regex パターン（閉じ括弧なし）
	result := ExecuteSearchCode("func(", dir, "", "0", "3000")

	if result == "No matches found" {
		t.Error("Invalid regex should NOT return 'No matches found' — should return error message")
	}
	if !strings.Contains(result, "Error") {
		t.Errorf("Expected error message for invalid regex, got: %s", result)
	}
}

// --- convertEmbeddingResults テスト ---

func TestConvertEmbeddingResults_SingleFile(t *testing.T) {
	embResults := []embedding.SearchResult{
		{
			ChunkMeta: embedding.ChunkMeta{
				FilePath:  "src/main.go",
				StartLine: 10,
				EndLine:   12,
				BlockName: "func main",
			},
			Score:   0.95,
			Content: "line10\nline11\nline12",
		},
	}

	results := convertEmbeddingResults(embResults)

	if len(results) != 1 {
		t.Fatalf("Expected 1 file result, got %d", len(results))
	}
	if results[0].FilePath != "src/main.go" {
		t.Errorf("Expected FilePath 'src/main.go', got '%s'", results[0].FilePath)
	}
	if results[0].MatchCount != 3 {
		t.Errorf("Expected MatchCount 3, got %d", results[0].MatchCount)
	}
	if len(results[0].Matches) != 3 {
		t.Fatalf("Expected 3 matches, got %d", len(results[0].Matches))
	}

	// 全マッチが MatchTypeSemantic であること
	for i, m := range results[0].Matches {
		if m.Type != MatchTypeSemantic {
			t.Errorf("Match[%d].Type = %d, want %d (MatchTypeSemantic)", i, m.Type, MatchTypeSemantic)
		}
		if !m.IsMatch {
			t.Errorf("Match[%d].IsMatch should be true", i)
		}
		expectedLineNum := 10 + i
		if m.LineNum != expectedLineNum {
			t.Errorf("Match[%d].LineNum = %d, want %d", i, m.LineNum, expectedLineNum)
		}
	}

	// BlockInfo が設定されていること
	for i, m := range results[0].Matches {
		if m.Block == nil {
			t.Errorf("Match[%d].Block should not be nil (BlockName='func main')", i)
		} else {
			if m.Block.Name != "func main" {
				t.Errorf("Match[%d].Block.Name = %q, want 'func main'", i, m.Block.Name)
			}
			if m.Block.StartLine != 10 {
				t.Errorf("Match[%d].Block.StartLine = %d, want 10", i, m.Block.StartLine)
			}
		}
	}
}

func TestConvertEmbeddingResults_MultipleFiles(t *testing.T) {
	embResults := []embedding.SearchResult{
		{
			ChunkMeta: embedding.ChunkMeta{
				FilePath:  "src/a.go",
				StartLine: 1,
				EndLine:   2,
			},
			Content: "line1\nline2",
		},
		{
			ChunkMeta: embedding.ChunkMeta{
				FilePath:  "src/b.go",
				StartLine: 5,
				EndLine:   5,
			},
			Content: "line5",
		},
		{
			ChunkMeta: embedding.ChunkMeta{
				FilePath:  "src/a.go",
				StartLine: 10,
				EndLine:   10,
			},
			Content: "line10",
		},
	}

	results := convertEmbeddingResults(embResults)

	if len(results) != 2 {
		t.Fatalf("Expected 2 file results, got %d", len(results))
	}

	// src/a.go は2つのチャンクがマージされる
	if results[0].FilePath != "src/a.go" {
		t.Errorf("Expected first file 'src/a.go', got '%s'", results[0].FilePath)
	}
	if results[0].MatchCount != 3 { // 2 lines + 1 line
		t.Errorf("Expected MatchCount 3 for src/a.go, got %d", results[0].MatchCount)
	}

	// src/b.go
	if results[1].FilePath != "src/b.go" {
		t.Errorf("Expected second file 'src/b.go', got '%s'", results[1].FilePath)
	}
	if results[1].MatchCount != 1 {
		t.Errorf("Expected MatchCount 1 for src/b.go, got %d", results[1].MatchCount)
	}
}

func TestConvertEmbeddingResults_Empty(t *testing.T) {
	results := convertEmbeddingResults(nil)
	if len(results) != 0 {
		t.Errorf("Expected 0 results for nil input, got %d", len(results))
	}

	results = convertEmbeddingResults([]embedding.SearchResult{})
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(results))
	}
}

func TestConvertEmbeddingResults_NoBlockName(t *testing.T) {
	embResults := []embedding.SearchResult{
		{
			ChunkMeta: embedding.ChunkMeta{
				FilePath:  "src/util.go",
				StartLine: 1,
				EndLine:   1,
				BlockName: "", // ブロック名なし
			},
			Content: "package util",
		},
	}

	results := convertEmbeddingResults(embResults)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Matches[0].Block != nil {
		t.Error("Expected Block to be nil when BlockName is empty")
	}
}

// --- mergeSearchResults テスト ---

func TestMergeSearchResults_RgOnly(t *testing.T) {
	rgResults := []SearchResult{
		{FilePath: "a.go", Matches: []Match{{LineNum: 1, IsMatch: true, Type: MatchTypeDefinition}}, MatchCount: 1},
	}

	merged := mergeSearchResults(rgResults, nil)

	if len(merged) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(merged))
	}
	if merged[0].FilePath != "a.go" {
		t.Errorf("Expected 'a.go', got '%s'", merged[0].FilePath)
	}
}

func TestMergeSearchResults_EmbOnly(t *testing.T) {
	embResults := []SearchResult{
		{FilePath: "b.go", Matches: []Match{{LineNum: 5, IsMatch: true, Type: MatchTypeSemantic}}, MatchCount: 1},
	}

	merged := mergeSearchResults(nil, embResults)

	if len(merged) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(merged))
	}
	if merged[0].FilePath != "b.go" {
		t.Errorf("Expected 'b.go', got '%s'", merged[0].FilePath)
	}
}

func TestMergeSearchResults_BothNil(t *testing.T) {
	merged := mergeSearchResults(nil, nil)
	if merged != nil {
		t.Errorf("Expected nil for both nil inputs, got %v", merged)
	}
}

func TestMergeSearchResults_DeduplicateSameFileLine(t *testing.T) {
	rgResults := []SearchResult{
		{
			FilePath: "a.go",
			Matches: []Match{
				{LineNum: 10, Line: "func foo()", IsMatch: true, Type: MatchTypeDefinition},
				{LineNum: 15, Line: "x := 1", IsMatch: true, Type: MatchTypeAssignment},
			},
			MatchCount: 2,
		},
	}
	embResults := []SearchResult{
		{
			FilePath: "a.go",
			Matches: []Match{
				{LineNum: 10, Line: "func foo()", IsMatch: true, Type: MatchTypeSemantic}, // 重複
				{LineNum: 20, Line: "y := 2", IsMatch: true, Type: MatchTypeSemantic},     // 新規
			},
			MatchCount: 2,
		},
	}

	merged := mergeSearchResults(rgResults, embResults)

	if len(merged) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(merged))
	}
	// 行10は重複除去、行15と20は残る → 3マッチ
	if merged[0].MatchCount != 3 {
		t.Errorf("Expected MatchCount 3 (dedup line 10), got %d", merged[0].MatchCount)
	}
	if len(merged[0].Matches) != 3 {
		t.Errorf("Expected 3 matches, got %d", len(merged[0].Matches))
	}
}

func TestMergeSearchResults_DifferentFiles(t *testing.T) {
	rgResults := []SearchResult{
		{FilePath: "a.go", Matches: []Match{{LineNum: 1, IsMatch: true}}, MatchCount: 1},
	}
	embResults := []SearchResult{
		{FilePath: "b.go", Matches: []Match{{LineNum: 5, IsMatch: true, Type: MatchTypeSemantic}}, MatchCount: 1},
	}

	merged := mergeSearchResults(rgResults, embResults)

	if len(merged) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(merged))
	}
	files := make(map[string]bool)
	for _, r := range merged {
		files[r.FilePath] = true
	}
	if !files["a.go"] || !files["b.go"] {
		t.Errorf("Expected both a.go and b.go, got %v", files)
	}
}

func TestMergeSearchResults_EmptySlices(t *testing.T) {
	// 空スライス（nil ではない）
	rgResults := []SearchResult{}
	embResults := []SearchResult{
		{FilePath: "c.go", Matches: []Match{{LineNum: 1, IsMatch: true, Type: MatchTypeSemantic}}, MatchCount: 1},
	}

	merged := mergeSearchResults(rgResults, embResults)

	if len(merged) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(merged))
	}
	if merged[0].FilePath != "c.go" {
		t.Errorf("Expected 'c.go', got '%s'", merged[0].FilePath)
	}
}
