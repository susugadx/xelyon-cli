package agent

import (
	"fmt"
	"strings"
	"testing"
)

// ── read_file: targeted read ──

func TestCompactReadFile_TargetedSymbolRead(t *testing.T) {
	// symbol mode read は圧縮しない（str_replace に必要な exact content を保持）
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("%d: func code line %d", i, i))
	}
	input := strings.Join(lines, "\n")

	args := map[string]string{"path": "foo.go", "symbol": "ProcessData"}
	got := compactReadFile(args, input)
	if got != input {
		t.Error("symbol mode read should not be compacted")
	}
}

func TestCompactReadFile_TargetedStartLine(t *testing.T) {
	// start_line 指定 read は圧縮しない
	var lines []string
	for i := 50; i <= 300; i++ {
		lines = append(lines, fmt.Sprintf("%d: line %d", i, i))
	}
	input := strings.Join(lines, "\n")

	args := map[string]string{"path": "foo.go", "start_line": "50"}
	got := compactReadFile(args, input)
	if got != input {
		t.Error("start_line read should not be compacted")
	}
}

func TestCompactReadFile_TargetedEndLine(t *testing.T) {
	// end_line 指定 read は圧縮しない
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("%d: line %d", i, i))
	}
	input := strings.Join(lines, "\n")

	args := map[string]string{"path": "foo.go", "end_line": "200"}
	got := compactReadFile(args, input)
	if got != input {
		t.Error("end_line read should not be compacted")
	}
}

func TestCompactReadFile_TargetedRange(t *testing.T) {
	// start_line + end_line 指定 read は圧縮しない
	var lines []string
	for i := 100; i <= 350; i++ {
		lines = append(lines, fmt.Sprintf("%d: old_str exact match %d", i, i))
	}
	input := strings.Join(lines, "\n")

	args := map[string]string{"path": "foo.go", "start_line": "100", "end_line": "350"}
	got := compactReadFile(args, input)
	if got != input {
		t.Error("range read should not be compacted (needed for str_replace)")
	}
}

// ── read_file: batch read (paths) ──

func TestCompactReadFile_BatchPathsWithRange(t *testing.T) {
	// paths に range 指定がある場合は targeted → 圧縮しない
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("📄 File: main.go:10-50\n%d: exact content %d", i, i))
	}
	input := strings.Join(lines, "\n")

	args := map[string]string{"paths": `["main.go:10-50","util.go"]`}
	got := compactReadFile(args, input)
	if got != input {
		t.Error("batch read with range should not be compacted")
	}
}

func TestCompactReadFile_BatchPathsSingleLineRange(t *testing.T) {
	// "path:123" (single line) も targeted
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("%d: line %d", i, i))
	}
	input := strings.Join(lines, "\n")

	args := map[string]string{"paths": `["config.go:42"]`}
	got := compactReadFile(args, input)
	if got != input {
		t.Error("batch read with single-line range should not be compacted")
	}
}

func TestCompactReadFile_BatchPathsBroadOnly(t *testing.T) {
	// paths に range がない場合は broad → 通常の圧縮対象
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("%d: broad content %d", i, i))
	}
	input := strings.Join(lines, "\n")

	args := map[string]string{"paths": `["main.go","util.go","config.go"]`}
	got := compactReadFile(args, input)
	if !strings.Contains(got, "lines omitted") {
		t.Error("batch broad read should be compacted")
	}
}

// ── read_file: broad read ──

func TestCompactReadFile_Short(t *testing.T) {
	short := strings.Repeat("1: line\n", 50)
	args := map[string]string{"path": "foo.go"}
	got := compactReadFile(args, short)
	if got != short {
		t.Error("short read_file should return as-is")
	}
}

func TestCompactReadFile_Outline(t *testing.T) {
	outline := strings.Repeat("1: line\n", 200) + "\n--- Signatures ---\n  L50 funcFoo\n"
	args := map[string]string{"path": "foo.go"}
	got := compactReadFile(args, outline)
	if got != outline {
		t.Error("outline format should return as-is")
	}
}

func TestCompactReadFile_LargeFile(t *testing.T) {
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("%d: line content %d", i, i))
	}
	lines = append(lines, "(200 lines total. Use start_line/end_line to read specific sections)")
	input := strings.Join(lines, "\n")

	args := map[string]string{"path": "foo.go"}
	got := compactReadFile(args, input)

	if !strings.Contains(got, "1: line content 1") {
		t.Error("should contain first line")
	}
	if !strings.Contains(got, fmt.Sprintf("%d: line content %d", compactReadFileHeadLines, compactReadFileHeadLines)) {
		t.Errorf("should contain line %d", compactReadFileHeadLines)
	}
	if !strings.Contains(got, "lines omitted") {
		t.Error("should contain omission notice")
	}
	if !strings.Contains(got, "200: line content 200") {
		t.Error("should contain last content line")
	}
	if !strings.Contains(got, "200 lines total") {
		t.Error("should preserve guide message")
	}
	if len(got) >= len(input) {
		t.Errorf("compacted should be shorter: got %d >= original %d", len(got), len(input))
	}
}

func TestCompactReadFile_NoGuide(t *testing.T) {
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("%d: code line %d", i, i))
	}
	input := strings.Join(lines, "\n")

	args := map[string]string{"path": "foo.go"}
	got := compactReadFile(args, input)
	if !strings.Contains(got, "lines omitted") {
		t.Error("should compact even without guide message")
	}
}

// ── read_file: guidance preservation ──

func TestCompactReadFile_GuidancePreserved(t *testing.T) {
	// large broad read + [GUIDANCE] → compaction 後も guidance が残る
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("%d: code line %d", i, i))
	}
	body := strings.Join(lines, "\n")
	guidance := "\n\n[GUIDANCE] You have read this file 3 times in this session. " +
		"Consider reading a larger range or using symbol mode instead of multiple micro-reads."
	input := body + guidance

	args := map[string]string{"path": "foo.go"}
	got := compactReadFile(args, input)

	// body は圧縮されている
	if !strings.Contains(got, "lines omitted") {
		t.Error("body should be compacted")
	}

	// guidance は保持されている
	if !strings.Contains(got, "[GUIDANCE]") {
		t.Error("guidance should be preserved after compaction")
	}
	if !strings.Contains(got, "read this file 3 times") {
		t.Error("guidance content should be preserved")
	}
}

func TestCompactReadFile_GuidancePreservedShort(t *testing.T) {
	// short read + [GUIDANCE] → そのまま（body 圧縮不要だが guidance は維持）
	body := "1: line one\n2: line two\n"
	guidance := "\n\n[GUIDANCE] You have read this file 4 times."
	input := body + guidance

	args := map[string]string{"path": "foo.go"}
	got := compactReadFile(args, input)

	if got != input {
		t.Error("short read with guidance should return as-is")
	}
}

func TestCompactReadFile_GuidanceOnTargetedRead(t *testing.T) {
	// targeted read + [GUIDANCE] → 圧縮しないのでそのまま保持
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("%d: code %d", i, i))
	}
	body := strings.Join(lines, "\n")
	guidance := "\n\n[GUIDANCE] You have read this file 5 times."
	input := body + guidance

	args := map[string]string{"path": "foo.go", "symbol": "MyFunc"}
	got := compactReadFile(args, input)
	if got != input {
		t.Error("targeted read with guidance should return as-is")
	}
}

// ── search_code ──

func TestCompactSearchCode_Short(t *testing.T) {
	short := "Found 2 match(es) in 1 file(s)\n\n📄 foo.go (2 match(es))\n  [def]  >  10 │ func Foo()\n  [call] >  20 │ Foo()\n"
	got := compactSearchCode(short)
	if got != short {
		t.Error("short search_code should return as-is")
	}
}

func TestCompactSearchCode_Large(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("Found 50 match(es) in 5 file(s)\n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&sb, "\n📄 pkg/file%d.go (10 match(es))\n", i)
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&sb, "  [ref]  > %4d │ some code referencing Foo\n", i*100+j+1)
		}
	}
	sb.WriteString("\nTip: Use str_replace line-range mode for editing\n")
	input := sb.String()

	got := compactSearchCode(input)

	// Step1: search_code compaction disabled — 入力がそのまま返る
	if got != input {
		t.Errorf("compactSearchCode should return input unchanged, got len=%d want len=%d", len(got), len(input))
	}
}

func TestCompactSearchCode_ManifestMode(t *testing.T) {
	// manifest 形式の結果でファイル一覧が保持される
	var sb strings.Builder
	sb.WriteString("Found 100 matches in 10 files:\n")
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&sb, "  pkg/file%d.go                              — %d matches (funcA, funcB)\n", i, 10)
	}
	// pad to exceed threshold
	for i := 0; i < 50; i++ {
		sb.WriteString("\n")
	}
	input := sb.String()

	got := compactSearchCode(input)

	if !strings.Contains(got, "Found 100 matches in 10 files") {
		t.Error("should preserve summary line")
	}
	// manifest lines preserved
	for i := 0; i < 10; i++ {
		expected := fmt.Sprintf("pkg/file%d.go", i)
		if !strings.Contains(got, expected) {
			t.Errorf("should preserve manifest line for %s", expected)
		}
	}
	if !strings.Contains(got, "— 10 matches") {
		t.Error("should preserve match count in manifest lines")
	}
	if !strings.Contains(got, "funcA, funcB") {
		t.Error("should preserve block names in manifest lines")
	}
}

func TestCompactSearchCode_SinglePatternWarning(t *testing.T) {
	// single-pattern で warning が保持される
	var sb strings.Builder
	sb.WriteString("Warning: ripgrep (rg) not found; using grep fallback mode.\n")
	sb.WriteString("Found 30 match(es) in 3 file(s)\n")
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&sb, "\n📄 pkg/file%d.go (10 match(es))\n", i)
		for j := 0; j < 20; j++ {
			fmt.Fprintf(&sb, "  [ref]  > %4d │ some matching code\n", j+1)
		}
	}
	input := sb.String()

	got := compactSearchCode(input)

	if !strings.Contains(got, "Warning: ripgrep") {
		t.Error("should preserve warning in single-pattern compaction")
	}
	if !strings.Contains(got, "Found 30 match(es)") {
		t.Error("should preserve summary")
	}
}

// ── search_code: multi-pattern grouping ──

func TestCompactSearchCode_MultiPatternGrouping(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("Found 80 match(es) across 2/2 patterns\n")
	sb.WriteString("\n━━ Pattern 1/2: \"foo\" ━━\n")
	sb.WriteString("\n📄 a.go (20 match(es))\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "  [ref]  > %4d │ foo reference\n", i+1)
	}
	sb.WriteString("\n📄 c.go (20 match(es))\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "  [ref]  > %4d │ foo in c\n", i+1)
	}
	sb.WriteString("\n━━ Pattern 2/2: \"bar\" ━━\n")
	sb.WriteString("\n📄 b.go (20 match(es))\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "  [ref]  > %4d │ bar reference\n", i+1)
	}
	sb.WriteString("\n[Results truncated to fit token budget (3000)]\n")
	input := sb.String()

	got := compactSearchCode(input)

	// pattern headers are preserved
	if !strings.Contains(got, "━━ Pattern 1/2: \"foo\" ━━") {
		t.Error("should keep pattern 1 header")
	}
	if !strings.Contains(got, "━━ Pattern 2/2: \"bar\" ━━") {
		t.Error("should keep pattern 2 header")
	}

	// file headers are preserved
	if !strings.Contains(got, "📄 a.go") {
		t.Error("should keep file a.go under pattern 1")
	}
	if !strings.Contains(got, "📄 c.go") {
		t.Error("should keep file c.go under pattern 1")
	}
	if !strings.Contains(got, "📄 b.go") {
		t.Error("should keep file b.go under pattern 2")
	}

	// Step1: compaction disabled — 入力がそのまま返る
	if got != input {
		t.Errorf("compactSearchCode should return input unchanged, got len=%d want len=%d", len(got), len(input))
	}
}

func TestCompactSearchCode_MultiPatternOldTestCompat(t *testing.T) {
	// backward compat: simpler multi-pattern test
	var sb strings.Builder
	sb.WriteString("Found 30 match(es) across 2/2 patterns\n")
	sb.WriteString("\n━━ Pattern 1/2: \"foo\" ━━\n")
	sb.WriteString("\n📄 a.go (15 match(es))\n")
	for i := 0; i < 15; i++ {
		fmt.Fprintf(&sb, "  [ref]  > %4d │ foo reference\n", i+1)
	}
	sb.WriteString("\n━━ Pattern 2/2: \"bar\" ━━\n")
	sb.WriteString("\n📄 b.go (15 match(es))\n")
	for i := 0; i < 15; i++ {
		fmt.Fprintf(&sb, "  [ref]  > %4d │ bar reference\n", i+1)
	}
	sb.WriteString("\n[Results truncated to fit token budget (3000)]\n")
	input := sb.String()

	got := compactSearchCode(input)
	if !strings.Contains(got, "━━ Pattern 1/2") {
		t.Error("should keep multi-pattern headers")
	}
	if !strings.Contains(got, "Results truncated") {
		t.Error("should keep truncation notice")
	}
}

func TestCompactSearchCode_MultiPatternDiagnostics(t *testing.T) {
	// multi-pattern で一部 pattern が error/no match の場合に diagnostics が保持される
	var sb strings.Builder
	sb.WriteString("Found 50 match(es) across 1/3 patterns\n\n")
	// pattern 1: normal results (enough lines to exceed threshold)
	sb.WriteString("━━ Pattern 1/3: \"foo\" ━━\n")
	sb.WriteString("\n📄 a.go (30 match(es))\n")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&sb, "  [ref]  > %4d │ foo reference line\n", i+1)
	}
	sb.WriteString("\n📄 d.go (20 match(es))\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "  [ref]  > %4d │ foo in d line\n", i+1)
	}
	// pattern 2: error
	sb.WriteString("\n━━ Pattern 2/3: \"bad[regex\" ━━\n")
	sb.WriteString("⚠️ Error: regex syntax error\n\n")
	// pattern 3: no matches
	sb.WriteString("━━ Pattern 3/3: \"nonexistent\" ━━\n")
	sb.WriteString("No matches found\n\n")
	sb.WriteString("[Results truncated to fit token budget (3000)]\n")
	input := sb.String()

	got := compactSearchCode(input)

	// Step1: compaction disabled — 入力がそのまま返る
	if got != input {
		t.Errorf("compactSearchCode should return input unchanged, got len=%d want len=%d", len(got), len(input))
	}
}

func TestCompactSearchCode_MultiPatternWarning(t *testing.T) {
	// multi-pattern で Warning が保持される
	var sb strings.Builder
	sb.WriteString("Found 20 match(es) across 1/2 patterns\n\n")
	sb.WriteString("━━ Pattern 1/2: \"foo\" ━━\n")
	sb.WriteString("Warning: ripgrep (rg) not found; using grep fallback mode.\n")
	sb.WriteString("\n📄 a.go (20 match(es))\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "  [ref]  > %4d │ foo ref\n", i+1)
	}
	sb.WriteString("\n━━ Pattern 2/2: \"bar\" ━━\n")
	sb.WriteString("\n📄 b.go (20 match(es))\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "  [ref]  > %4d │ bar ref\n", i+1)
	}
	input := sb.String()

	got := compactSearchCode(input)

	if !strings.Contains(got, "Warning: ripgrep") {
		t.Error("should preserve pattern-level warning")
	}

	// Warning should appear between pattern 1 header and pattern 2 header
	p1Idx := strings.Index(got, "━━ Pattern 1/2")
	p2Idx := strings.Index(got, "━━ Pattern 2/2")
	warnIdx := strings.Index(got, "Warning: ripgrep")
	if warnIdx < p1Idx || warnIdx > p2Idx {
		t.Error("warning should appear between pattern 1 and pattern 2")
	}
}

func TestCompactSearchCode_MultiPatternManifest(t *testing.T) {
	// multi-pattern manifest 形式で pattern ごとの manifest 行が保持される
	var sb strings.Builder
	sb.WriteString("Found 40 matches across 2 patterns:\n\n")
	sb.WriteString("━━ Pattern 1/2: \"foo\" ━━\n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&sb, "  pkg/foo%d.go                              — %d matches (funcA)\n", i, 4)
	}
	sb.WriteString("\n")
	sb.WriteString("━━ Pattern 2/2: \"bar\" ━━\n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&sb, "  pkg/bar%d.go                              — %d matches (funcB, funcC)\n", i, 4)
	}
	sb.WriteString("\n")
	// pad to exceed threshold
	for i := 0; i < 50; i++ {
		sb.WriteString("\n")
	}
	input := sb.String()

	got := compactSearchCode(input)

	// pattern headers
	if !strings.Contains(got, "━━ Pattern 1/2: \"foo\" ━━") {
		t.Error("should keep pattern 1 header")
	}
	if !strings.Contains(got, "━━ Pattern 2/2: \"bar\" ━━") {
		t.Error("should keep pattern 2 header")
	}

	// manifest lines under pattern 1
	for i := 0; i < 5; i++ {
		expected := fmt.Sprintf("pkg/foo%d.go", i)
		if !strings.Contains(got, expected) {
			t.Errorf("should preserve manifest line %s under pattern 1", expected)
		}
	}
	// manifest lines under pattern 2
	for i := 0; i < 5; i++ {
		expected := fmt.Sprintf("pkg/bar%d.go", i)
		if !strings.Contains(got, expected) {
			t.Errorf("should preserve manifest line %s under pattern 2", expected)
		}
	}

	// grouping: foo files before pattern 2, bar files after
	p2Idx := strings.Index(got, "━━ Pattern 2/2")
	foo0Idx := strings.Index(got, "pkg/foo0.go")
	bar0Idx := strings.Index(got, "pkg/bar0.go")
	if foo0Idx > p2Idx {
		t.Error("foo manifest lines should appear before pattern 2 header")
	}
	if bar0Idx < p2Idx {
		t.Error("bar manifest lines should appear after pattern 2 header")
	}

	// block names preserved
	if !strings.Contains(got, "funcA") {
		t.Error("should preserve block names in pattern 1 manifest lines")
	}
	if !strings.Contains(got, "funcB, funcC") {
		t.Error("should preserve block names in pattern 2 manifest lines")
	}
}

// ── isManifestLine ──

func TestIsManifestLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		// positive: formatManifestLine output
		{"  pkg/foo.go                              — 5 matches (funcA, funcB)", true},
		{"  pkg/bar.go                              — 3 matches", true},
		{"  very/long/path/to/file.go               — 1 matches", true},

		// negative: other line types
		{"Found 10 matches in 3 files:", false},
		{"📄 foo.go (5 match(es))", false},
		{"  [ref]  >   10 │ some code", false},
		{"━━ Pattern 1/2: \"foo\" ━━", false},
		{"No matches found", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isManifestLine(tt.line); got != tt.want {
				t.Errorf("isManifestLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

// ── pathsHasRange ──

func TestPathsHasRange(t *testing.T) {
	tests := []struct {
		name string
		json string
		want bool
	}{
		{"broad only", `["a.go","b.go"]`, false},
		{"with range", `["a.go:10-20"]`, true},
		{"single line", `["a.go:42"]`, true},
		{"mixed", `["a.go","b.go:1-5"]`, true},
		{"windows path no range", `["C:\\src\\foo.go"]`, false},
		{"invalid json", `not json`, false},
		{"empty array", `[]`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathsHasRange(tt.json); got != tt.want {
				t.Errorf("pathsHasRange(%s) = %v, want %v", tt.json, got, tt.want)
			}
		})
	}
}

// ── inspect_symbol ──

func TestCompactInspectSymbol_Short(t *testing.T) {
	short := "── func Foo (L10-L20) in foo.go ──\n10: func Foo() {\n11: }\n"
	got := compactInspectSymbol(short)
	if got != short {
		t.Error("short inspect_symbol should return as-is")
	}
}

func TestCompactInspectSymbol_Large(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("── func ProcessData (L10-L80) in processor.go ──\n")
	for i := 10; i <= 80; i++ {
		fmt.Fprintf(&sb, "%d: code line %d\n", i, i)
	}
	sb.WriteString("\nCallers: 5 examples (of 20 found)\n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&sb, "  - caller%d.go:%d in funcCaller%d\n", i, i*10+1, i)
	}
	sb.WriteString("  (+ more callers. Use mode=\"full\" or search_code)\n")
	sb.WriteString("\nReferences (8):\n")
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&sb, "  - ref%d.go:%d | some reference code\n", i, i*5+1)
	}
	sb.WriteString("\nRelated tests (3):\n")
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&sb, "  - test%d.go:%d | func TestProcessor%d\n", i, i*20+1, i)
	}
	sb.WriteString("\nNote: Some results were truncated upstream. For comprehensive search, use search_code.\n")
	input := sb.String()

	got := compactInspectSymbol(input)

	if !strings.Contains(got, "── func ProcessData (L10-L80) in processor.go ──") {
		t.Error("should contain header")
	}
	if !strings.Contains(got, "10: code line 10") {
		t.Error("should contain beginning of body")
	}
	if !strings.Contains(got, "more body lines omitted") {
		t.Error("should contain body omission marker")
	}
	if !strings.Contains(got, "Callers: 5 examples") {
		t.Error("should contain Callers header")
	}
	if !strings.Contains(got, "(+ more callers") {
		t.Error("should preserve escalation hint")
	}
	if !strings.Contains(got, "Note:") {
		t.Error("should preserve Note line")
	}
	if len(got) >= len(input) {
		t.Errorf("compacted should be shorter: got %d >= original %d", len(got), len(input))
	}
}

func TestCompactInspectSymbol_MultipleCandidates(t *testing.T) {
	input := `Multiple symbols matched "Foo":
  1. foo.go                                   type Foo (L10-L50)
  2. bar.go                                   func Foo (L200-L220)

Refine with path to disambiguate.`

	got := compactInspectSymbol(input)
	if got != input {
		t.Error("multiple candidates should return as-is")
	}
}

// ── list_dir ──

func TestCompactListDir_Short(t *testing.T) {
	short := "📂 /path/to/dir\nsummary: depth=1, dirs=2, files=3\ndirs: a/, b/\nfiles: c.go (100 bytes)\n"
	got := compactListDir(short)
	if got != short {
		t.Error("short list_dir should return as-is")
	}
}

func TestCompactListDir_Large(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("📂 /path/to/project\n")
	sb.WriteString("summary: depth=3, dirs=50, files=200\n")
	sb.WriteString("dirs: src/, test/, docs/, config/, scripts/, internal/, pkg/, cmd/\n")
	sb.WriteString("files: go.mod (1024 bytes), go.sum (8192 bytes), Makefile (512 bytes)\n")
	sb.WriteString("subtrees: 6 shown (+2 more)\n")
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&sb, "- subdir%d/ -> dirs=%d, files=%d\n", i, i%5, i%10)
		fmt.Fprintf(&sb, "  files: file%d.go (%d bytes)\n", i, i*100)
	}
	input := sb.String()

	got := compactListDir(input)

	if !strings.Contains(got, "📂 /path/to/project") {
		t.Error("should contain path header")
	}
	if !strings.Contains(got, "summary: depth=3") {
		t.Error("should contain summary line")
	}
	if !strings.Contains(got, "dirs: src/") {
		t.Error("should contain root dirs")
	}
	if !strings.Contains(got, "subtree lines omitted") {
		t.Error("should contain omission notice")
	}
	if len(got) >= len(input) {
		t.Errorf("compacted should be shorter: got %d >= original %d", len(got), len(input))
	}
}

// ── isTargetedRead ──

func TestIsTargetedRead(t *testing.T) {
	tests := []struct {
		name string
		args map[string]string
		want bool
	}{
		{"broad read", map[string]string{"path": "foo.go"}, false},
		{"symbol mode", map[string]string{"path": "foo.go", "symbol": "Foo"}, true},
		{"start_line", map[string]string{"path": "foo.go", "start_line": "10"}, true},
		{"end_line", map[string]string{"path": "foo.go", "end_line": "50"}, true},
		{"range", map[string]string{"path": "foo.go", "start_line": "10", "end_line": "50"}, true},
		{"empty args", map[string]string{}, false},

		// batch read (paths)
		{"paths broad only", map[string]string{"paths": `["a.go","b.go"]`}, false},
		{"paths with range", map[string]string{"paths": `["a.go:10-20","b.go"]`}, true},
		{"paths single line", map[string]string{"paths": `["a.go:42"]`}, true},
		{"paths all ranges", map[string]string{"paths": `["a.go:1-10","b.go:20-30"]`}, true},
		{"paths windows path", map[string]string{"paths": `["C:\\src\\foo.go"]`}, false},
		{"paths invalid json", map[string]string{"paths": `not json`}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTargetedRead(tt.args); got != tt.want {
				t.Errorf("isTargetedRead(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

// ── extractGuidance ──

func TestExtractGuidance(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantBody  string
		wantGuide string
	}{
		{
			name:      "no guidance",
			input:     "1: line\n2: line\n",
			wantBody:  "1: line\n2: line\n",
			wantGuide: "",
		},
		{
			name:      "with guidance",
			input:     "1: line\n2: line\n\n[GUIDANCE] You have read this file 3 times.",
			wantBody:  "1: line\n2: line",
			wantGuide: "\n\n[GUIDANCE] You have read this file 3 times.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, guide := extractGuidance(tt.input)
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
			if guide != tt.wantGuide {
				t.Errorf("guidance = %q, want %q", guide, tt.wantGuide)
			}
		})
	}
}

// ── compactToolResult dispatch 統合テスト ──

func TestCompactToolResult_DispatchesToBroadReadFile(t *testing.T) {
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, fmt.Sprintf("%d: content %d", i, i))
	}
	input := strings.Join(lines, "\n")
	args := map[string]string{"path": "foo.go"}

	got := compactReadFile(args, input)
	if !strings.Contains(got, "lines omitted") {
		t.Error("broad read_file should compact large results")
	}
}

func TestCompactToolResult_DispatchesToSearchCode(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("Found 60 match(es) in 6 file(s)\n")
	for i := 0; i < 6; i++ {
		fmt.Fprintf(&sb, "\n📄 file%d.go (10 match(es))\n", i)
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&sb, "  [ref]  > %4d │ code\n", j+1)
		}
	}
	input := sb.String()

	got := compactSearchCode(input)
	// Step1: search_code compaction disabled — 入力がそのまま返る
	if got != input {
		t.Errorf("compactSearchCode should return input unchanged, got len=%d want len=%d", len(got), len(input))
	}
}
