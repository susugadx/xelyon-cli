package search

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

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
