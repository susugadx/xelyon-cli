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
