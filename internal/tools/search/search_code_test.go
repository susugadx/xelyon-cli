package search

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func setupSearchTestMocks(t *testing.T) {
	t.Helper()
}

// --- 統合テスト ---

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

	result := ExecuteSearchCode(SearchOptions{Pattern: "nonexistent_pattern_xyz", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if !strings.Contains(result, "No matches found") {
		t.Errorf("Expected 'No matches found' in result, got: %s", result)
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

	// context_lines=0: コンテキスト行なし
	result := ExecuteSearchCode(SearchOptions{Pattern: "target_match", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

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

	// 1行が1000文字を超えるファイルを追加（行長制限テスト）
	longLine := "var long_target_budget_check = \"" + strings.Repeat("a", 1000) + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "long.go"), []byte(longLine), 0644); err != nil {
		t.Fatal(err)
	}

	// 日本語コメントを含むファイルを追加（マルチバイト文字推定テスト）
	japaneseLine := "// target_budget_check " + strings.Repeat("あ", 100) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "japanese.go"), []byte(japaneseLine), 0644); err != nil {
		t.Fatal(err)
	}

	// context_lines=3 + 小バジェット → 確実に打ち切り
	result := ExecuteSearchCode(SearchOptions{Pattern: "target_budget_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 3, TokenBudget: 500, IsRegex: true, Multiline: false})

	if strings.Contains(result, "No matches found") {
		t.Error("Expected matches")
	}
	if !strings.Contains(result, "truncated") {
		t.Error("Expected truncation message with small token budget")
	}

	// longLine が切り詰められていることを確認
	longResult := ExecuteSearchCode(SearchOptions{Pattern: "long_target_budget_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if !strings.Contains(longResult, "...") {
		t.Error("Expected long line to be truncated with ...")
	}
	// 行自体の長さが maxLineLength (500) 付近に切り詰められているので、1000文字の 'a' はフルでは含まれない
	if strings.Contains(longResult, strings.Repeat("a", 1000)) {
		t.Error("Expected long line to not contain the full 1000 character string")
	}

	// 日本語行の検索が正常に行われ、バジェット内に収まるか
	jpResult := ExecuteSearchCode(SearchOptions{Pattern: "target_budget_check あ", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if !strings.Contains(jpResult, "target_budget_check") {
		t.Error("Expected japanese comment to be searchable and displayed")
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

	result := ExecuteSearchCode(SearchOptions{Pattern: "match_", Path: dir, FilePattern: "", FileType: "", CtxLines: 3, TokenBudget: 3000, IsRegex: true, Multiline: false})

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
	result1 := ExecuteSearchCode(SearchOptions{Pattern: "cached_target", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if strings.Contains(result1, "No matches found") {
		t.Fatal("Expected matches on first search")
	}

	// キャッシュに保存されたか確認
	if cache.setCalls == 0 {
		t.Error("Expected SetSearch to be called after first search")
	}

	// 2回目の検索 — キャッシュヒット
	getCalls := cache.getCalls
	result2 := ExecuteSearchCode(SearchOptions{Pattern: "cached_target", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

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
	result0 := ExecuteSearchCode(SearchOptions{Pattern: "target_param_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if strings.Contains(result0, "No matches found") {
		t.Fatal("Expected matches with context_lines=0")
	}

	// context_lines=3 で検索 — キャッシュキーが異なるため別結果が返るべき
	result3 := ExecuteSearchCode(SearchOptions{Pattern: "target_param_check", Path: dir, FilePattern: "", FileType: "", CtxLines: 3, TokenBudget: 3000, IsRegex: true, Multiline: false})
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

func (c *testSearchCache) GetFile(path string) (string, bool)          { return "", false }
func (c *testSearchCache) SetFile(path, content string)                {}
func (c *testSearchCache) GetDir(path string) (string, bool)           { return "", false }
func (c *testSearchCache) SetDir(path, result string)                  {}
func (c *testSearchCache) InvalidateFile(path string)                  {}
func (c *testSearchCache) InvalidateDir(path string)                   {}
func (c *testSearchCache) Clear()                                      {}
func (c *testSearchCache) ClearSearchCache()                           {}
func (c *testSearchCache) InvalidateSearchCacheForFile(absPath string) {}

func (c *testSearchCache) GetSearch(pattern, path string) (string, bool) {
	c.getCalls++
	key := pattern + "|" + path
	if v, ok := c.data[key]; ok {
		return v, true
	}
	return "", false
}

func (c *testSearchCache) SetSearch(pattern, path, result string, affectedFiles []string) {
	c.setCalls++
	key := pattern + "|" + path
	c.data[key] = result
}

// --- adaptiveContextTrim テスト ---

func TestAdaptiveContextTrim_FewMatches(t *testing.T) {
	// 5マッチ以下 → コンテキスト行そのまま維持
	matches := make([]Match, 0)
	for i := 1; i <= 15; i++ {
		isMatch := i%3 == 0 // 行3,6,9,12,15 → 5マッチ
		matches = append(matches, Match{
			LineNum: i,
			Line:    fmt.Sprintf("line%d", i),
			IsMatch: isMatch,
			Type:    MatchTypeUsage,
		})
	}
	results := []SearchResult{{FilePath: "few.go", Matches: matches, MatchCount: 5}}
	trimmed := adaptiveContextTrim(results)

	if len(trimmed[0].Matches) != len(matches) {
		t.Errorf("Expected all %d matches preserved, got %d", len(matches), len(trimmed[0].Matches))
	}
}

func TestAdaptiveContextTrim_ManyMatches(t *testing.T) {
	// 10マッチ → コンテキスト行は直前後1行のみ
	var matches []Match
	for i := 1; i <= 30; i++ {
		isMatch := i%3 == 0 // 行3,6,9,...,30 → 10マッチ
		matches = append(matches, Match{
			LineNum: i,
			Line:    fmt.Sprintf("line%d", i),
			IsMatch: isMatch,
			Type:    MatchTypeUsage,
		})
	}
	results := []SearchResult{{FilePath: "many.go", Matches: matches, MatchCount: 10}}
	trimmed := adaptiveContextTrim(results)

	// マッチ行は全部残る
	matchCount := 0
	ctxCount := 0
	for _, m := range trimmed[0].Matches {
		if m.IsMatch {
			matchCount++
		} else {
			ctxCount++
		}
	}
	if matchCount != 10 {
		t.Errorf("Expected 10 match lines, got %d", matchCount)
	}
	// コンテキスト行は削減されるはず（元は20行 → マッチ隣接のみ残る）
	if ctxCount >= 20 {
		t.Errorf("Expected context lines to be reduced, but got %d (same as original)", ctxCount)
	}

	// 各コンテキスト行はマッチ行の直前後1行以内か確認
	for j, m := range trimmed[0].Matches {
		if m.IsMatch {
			continue
		}
		hasNearMatch := false
		for k := max(0, j-1); k <= min(len(trimmed[0].Matches)-1, j+1); k++ {
			if trimmed[0].Matches[k].IsMatch {
				hasNearMatch = true
				break
			}
		}
		if !hasNearMatch {
			t.Errorf("Context line %d at index %d is not adjacent to any match line", m.LineNum, j)
		}
	}
}

func TestAdaptiveContextTrim_TooManyMatches(t *testing.T) {
	// 20マッチ → コンテキスト行なし
	var matches []Match
	for i := 1; i <= 60; i++ {
		isMatch := i%3 == 0 // 行3,6,9,...,60 → 20マッチ
		matches = append(matches, Match{
			LineNum: i,
			Line:    fmt.Sprintf("line%d", i),
			IsMatch: isMatch,
			Type:    MatchTypeUsage,
		})
	}
	results := []SearchResult{{FilePath: "toomany.go", Matches: matches, MatchCount: 20}}
	trimmed := adaptiveContextTrim(results)

	// マッチ行のみ残る
	for _, m := range trimmed[0].Matches {
		if !m.IsMatch {
			t.Errorf("Expected only match lines, but found context line %d", m.LineNum)
		}
	}
	if len(trimmed[0].Matches) != 20 {
		t.Errorf("Expected 20 match lines, got %d", len(trimmed[0].Matches))
	}
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

	result := ExecuteSearchCode(SearchOptions{Pattern: "func_a,func_b", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

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

	result := ExecuteSearchCode(SearchOptions{Pattern: "existing_func,nonexistent_xyz_pattern", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	// マッチしたパターンの結果が含まれる
	if !strings.Contains(result, "existing_func") {
		t.Error("Expected existing_func match in result")
	}
	// マッチしないパターンは "No matches found"
	if !strings.Contains(result, "No matches found") {
		t.Error("Expected 'No matches found' for unmatched pattern")
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
		// Modifiers
		{"async def process():", MatchTypeDefinition},
		{"export function doSomething() {", MatchTypeDefinition},
		{"export default function doSomething() {", MatchTypeDefinition},
		{"async export const getUser = () => {", MatchTypeDefinition},
		// Return types
		{"static void main()", MatchTypeDefinition},
		{"public int getValue()", MatchTypeDefinition},
		{"private string ToString()", MatchTypeDefinition},
		// Control flow usage
		{"if something() {", MatchTypeUsage},
		{"for i := range items {", MatchTypeUsage},
		{"return getValue()", MatchTypeUsage},
		{"switch getType() {", MatchTypeUsage},
		{"while hasNext() {", MatchTypeUsage},
		// Anonymous func usage
		{"go func() {", MatchTypeUsage},
		{"defer func() {", MatchTypeUsage},
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

	result := ExecuteSearchCode(SearchOptions{Pattern: "target", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

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

	result := ExecuteSearchCode(SearchOptions{Pattern: "target_var", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

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
	result := ExecuteSearchCode(SearchOptions{Pattern: "func(", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if result == "No matches found" {
		t.Error("Invalid regex should NOT return 'No matches found' — should return error message")
	}
	if !strings.Contains(result, "Error") {
		t.Errorf("Expected error message for invalid regex, got: %s", result)
	}
}

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "500 characters or less",
			line:     strings.Repeat("a", 500),
			expected: strings.Repeat("a", 500),
		},
		{
			name:     "501 characters",
			line:     strings.Repeat("a", 501),
			expected: strings.Repeat("a", 500) + "...",
		},
		{
			name:     "empty line",
			line:     "",
			expected: "",
		},
		{
			name:     "multibyte characters (501 runes)",
			line:     strings.Repeat("あ", 501),
			expected: strings.Repeat("あ", 500) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateLine(tt.line)
			if result != tt.expected {
				t.Errorf("truncateLine() len=%d, expected len=%d", len([]rune(result)), len([]rune(tt.expected)))
			}
		})
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

	// regex としては '+' が特殊文字だが、固定文字列検索ならマッチする
	result := ExecuteSearchCode(SearchOptions{Pattern: "a+b", Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: false, Multiline: false})
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
	if !strings.Contains(result, "Warning: ripgrep (rg) not found; using grep fallback mode.") {
		t.Fatalf("expected grep fallback warning, got: %s", result)
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

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "English text",
			line:     "hello world", // runes: 11/2=5, bytes: 11/8=1, +3 = 9
			expected: 9,
		},
		{
			name:     "Japanese text",
			line:     "これはテストです", // runes: 8/2=4, bytes: 24/8=3, +3 = 10
			expected: 10,
		},
		{
			name:     "Mixed text",
			line:     "hello これはテスト world", // runes: 18/2=9, bytes: 30/8=3, +3 = 15
			expected: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateTokens(tt.line)
			if result != tt.expected {
				t.Errorf("estimateTokens(%q) = %d, expected %d", tt.line, result, tt.expected)
			}
		})
	}
}

// --- collapseBlockMatches テスト ---

func TestCollapseBlockMatches_ThreeMatchesSameBlock(t *testing.T) {
	block := &BlockInfo{Name: "func doStuff", StartLine: 10}
	results := []SearchResult{{
		FilePath:   "a.go",
		MatchCount: 3,
		Matches: []Match{
			{LineNum: 11, Line: "ctx line before", IsMatch: false},
			{LineNum: 12, Line: "match1", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 13, Line: "ctx between 1-2", IsMatch: false},
			{LineNum: 14, Line: "match2", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 15, Line: "ctx between 2-3", IsMatch: false},
			{LineNum: 16, Line: "match3", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 17, Line: "ctx line after", IsMatch: false},
		},
	}}

	collapseBlockMatches(results)

	matches := results[0].Matches
	// Expected: ctx_before, match1, marker, match3, ctx_after
	if len(matches) != 5 {
		t.Fatalf("expected 5 entries, got %d: %+v", len(matches), matches)
	}
	if matches[0].LineNum != 11 {
		t.Errorf("expected pre-context at line 11, got %d", matches[0].LineNum)
	}
	if matches[1].LineNum != 12 || !matches[1].IsMatch {
		t.Errorf("expected first match at line 12")
	}
	if matches[2].LineNum != -1 {
		t.Errorf("expected collapse marker (LineNum=-1), got %d", matches[2].LineNum)
	}
	if !strings.Contains(matches[2].Line, "+1 more match") {
		t.Errorf("expected collapse marker text, got %q", matches[2].Line)
	}
	if matches[3].LineNum != 16 || !matches[3].IsMatch {
		t.Errorf("expected last match at line 16")
	}
	if matches[4].LineNum != 17 {
		t.Errorf("expected post-context at line 17, got %d", matches[4].LineNum)
	}
}

func TestCollapseBlockMatches_TwoMatchesNoCollapse(t *testing.T) {
	block := &BlockInfo{Name: "func foo", StartLine: 5}
	results := []SearchResult{{
		FilePath:   "b.go",
		MatchCount: 2,
		Matches: []Match{
			{LineNum: 6, Line: "match1", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 7, Line: "match2", IsMatch: true, Type: MatchTypeUsage, Block: block},
		},
	}}

	collapseBlockMatches(results)

	if len(results[0].Matches) != 2 {
		t.Fatalf("expected 2 entries (no collapse), got %d", len(results[0].Matches))
	}
}

func TestCollapseBlockMatches_NilBlockNoCollapse(t *testing.T) {
	results := []SearchResult{{
		FilePath:   "c.go",
		MatchCount: 3,
		Matches: []Match{
			{LineNum: 1, Line: "match1", IsMatch: true, Type: MatchTypeUsage, Block: nil},
			{LineNum: 2, Line: "match2", IsMatch: true, Type: MatchTypeUsage, Block: nil},
			{LineNum: 3, Line: "match3", IsMatch: true, Type: MatchTypeUsage, Block: nil},
		},
	}}

	collapseBlockMatches(results)

	if len(results[0].Matches) != 3 {
		t.Fatalf("expected 3 entries (nil block, no collapse), got %d", len(results[0].Matches))
	}
}

func TestCollapseBlockMatches_DifferentBlocks(t *testing.T) {
	blockA := &BlockInfo{Name: "func a", StartLine: 10}
	blockB := &BlockInfo{Name: "func b", StartLine: 30}
	results := []SearchResult{{
		FilePath:   "d.go",
		MatchCount: 4,
		Matches: []Match{
			{LineNum: 12, Line: "a1", IsMatch: true, Type: MatchTypeUsage, Block: blockA},
			{LineNum: 14, Line: "a2", IsMatch: true, Type: MatchTypeUsage, Block: blockA},
			{LineNum: 32, Line: "b1", IsMatch: true, Type: MatchTypeUsage, Block: blockB},
			{LineNum: 34, Line: "b2", IsMatch: true, Type: MatchTypeUsage, Block: blockB},
		},
	}}

	collapseBlockMatches(results)

	// Neither block has 3+ matches, so no collapse
	if len(results[0].Matches) != 4 {
		t.Fatalf("expected 4 entries (different blocks <3), got %d", len(results[0].Matches))
	}
}

func TestCollapseBlockMatches_FiveMatches(t *testing.T) {
	block := &BlockInfo{Name: "func big", StartLine: 1}
	results := []SearchResult{{
		FilePath:   "e.go",
		MatchCount: 5,
		Matches: []Match{
			{LineNum: 2, Line: "m1", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 4, Line: "m2", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 6, Line: "m3", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 8, Line: "m4", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 10, Line: "m5", IsMatch: true, Type: MatchTypeUsage, Block: block},
		},
	}}

	collapseBlockMatches(results)

	matches := results[0].Matches
	// Expected: m1, marker(+3), m5
	if len(matches) != 3 {
		t.Fatalf("expected 3 entries, got %d: %+v", len(matches), matches)
	}
	if matches[0].LineNum != 2 {
		t.Errorf("expected first match at line 2")
	}
	if matches[1].LineNum != -1 || !strings.Contains(matches[1].Line, "+3 more match") {
		t.Errorf("expected collapse marker with +3, got %q", matches[1].Line)
	}
	if matches[2].LineNum != 10 {
		t.Errorf("expected last match at line 10")
	}
}

func TestCollapseBlockMatches_FormatterRendering(t *testing.T) {
	block := &BlockInfo{Name: "func render", StartLine: 5}
	results := []SearchResult{{
		FilePath:   "f.go",
		MatchCount: 3,
		Matches: []Match{
			{LineNum: 6, Line: "first", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 7, Line: "middle", IsMatch: true, Type: MatchTypeUsage, Block: block},
			{LineNum: 8, Line: "last", IsMatch: true, Type: MatchTypeUsage, Block: block},
		},
	}}

	collapseBlockMatches(results)
	out := formatSearchResults(results, false, 3000)

	if !strings.Contains(out, "+1 more match") {
		t.Errorf("expected collapse marker in formatted output, got:\n%s", out)
	}
	// 中間マッチ行は含まれない
	if strings.Contains(out, "middle") {
		t.Errorf("expected middle match to be collapsed, got:\n%s", out)
	}
}

// --- manifest mode テスト ---

func TestFormatManifestResults_WithBlocks(t *testing.T) {
	results := []SearchResult{
		{
			FilePath:   "internal/agent/agent.go",
			MatchCount: 5,
			Matches: []Match{
				{LineNum: 10, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func NewAgent", StartLine: 5}},
				{LineNum: 20, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func Cleanup", StartLine: 15}},
				{LineNum: 30, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func handleRequest", StartLine: 25}},
				{LineNum: 40, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func fourthBlock", StartLine: 35}},
				{LineNum: 50, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: nil},
			},
		},
		{
			FilePath:   "internal/tools/execute.go",
			MatchCount: 2,
			Matches: []Match{
				{LineNum: 5, Line: "y", IsMatch: true, Type: MatchTypeUsage, Block: nil},
				{LineNum: 15, Line: "y", IsMatch: true, Type: MatchTypeUsage, Block: nil},
			},
		},
	}

	out := formatManifestResults(results)

	if !strings.Contains(out, "Found 7 matches in 2 files") {
		t.Errorf("expected header, got:\n%s", out)
	}
	if !strings.Contains(out, "agent.go") {
		t.Errorf("expected agent.go in output")
	}
	// ブロック名は最大3つ
	if !strings.Contains(out, "func NewAgent") || !strings.Contains(out, "func Cleanup") || !strings.Contains(out, "func handleRequest") {
		t.Errorf("expected first 3 block names, got:\n%s", out)
	}
	if strings.Contains(out, "fourthBlock") {
		t.Errorf("should not include 4th block name, got:\n%s", out)
	}
	// ブロックなしファイル
	if !strings.Contains(out, "execute.go") || !strings.Contains(out, "2 matches") {
		t.Errorf("expected execute.go with 2 matches, got:\n%s", out)
	}
	// コードスニペットは含まれない
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "│") {
			t.Errorf("manifest mode should not contain code snippets, got line: %s", line)
		}
	}
}

func TestFormatManifestResults_NoBlocks(t *testing.T) {
	results := []SearchResult{{
		FilePath:   "main.go",
		MatchCount: 3,
		Matches: []Match{
			{LineNum: 1, Line: "a", IsMatch: true, Type: MatchTypeUsage},
			{LineNum: 2, Line: "b", IsMatch: true, Type: MatchTypeUsage},
			{LineNum: 3, Line: "c", IsMatch: true, Type: MatchTypeUsage},
		},
	}}

	out := formatManifestResults(results)
	if !strings.Contains(out, "3 matches") {
		t.Errorf("expected 3 matches, got:\n%s", out)
	}
	// ブロック名の括弧がないことを確認
	if strings.Contains(out, "(") {
		t.Errorf("should not have block names in parens, got:\n%s", out)
	}
}

func TestFormatManifestMultiResults(t *testing.T) {
	collected := []patternResult{
		{
			Pattern: "handleSSE",
			Results: []SearchResult{{
				FilePath:   "stream.go",
				MatchCount: 3,
				Matches: []Match{
					{LineNum: 10, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func handleSSE", StartLine: 5}},
					{LineNum: 20, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func handleSSE", StartLine: 5}},
					{LineNum: 30, Line: "x", IsMatch: true, Type: MatchTypeUsage, Block: &BlockInfo{Name: "func parseEvent", StartLine: 25}},
				},
			}},
		},
		{
			Pattern: "badPattern",
			Error:   "regex syntax error",
		},
	}

	out := formatManifestMultiResults(collected)
	if !strings.Contains(out, "handleSSE") {
		t.Errorf("expected pattern name")
	}
	if !strings.Contains(out, "stream.go") {
		t.Errorf("expected file path")
	}
	if !strings.Contains(out, "⚠️ Error: regex syntax error") {
		t.Errorf("expected error for bad pattern, got:\n%s", out)
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
	// manifest mode ではコードスニペットが含まれない
	if strings.Contains(result, "Println") {
		t.Errorf("manifest mode should not include code snippets, got:\n%s", result)
	}
	if strings.Contains(result, "│") {
		t.Errorf("manifest mode should not contain │ separator, got:\n%s", result)
	}
}

func TestFormatMultiResults_WithPatternError(t *testing.T) {
	collected := []patternResult{
		{Pattern: "ok", Results: []SearchResult{{FilePath: "a.go", MatchCount: 1, Matches: []Match{{LineNum: 1, Line: "ok", IsMatch: true, Type: MatchTypeUsage}}}}},
		{Pattern: "bad", Error: "regex error"},
	}
	out := formatMultiResults(collected, 3000)
	if !strings.Contains(out, "⚠️ Error: regex error") {
		t.Fatalf("expected pattern error to be shown, got: %s", out)
	}
}
