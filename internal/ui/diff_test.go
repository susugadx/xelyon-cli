package ui

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/fatih/color"
)

func init() {
	// テスト時はカラー出力を無効化
	color.NoColor = true
}

// stripANSI はANSIエスケープシーケンスを除去
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

// captureOutput は差分描画の出力をキャプチャするヘルパー
func captureOutput(f func(io.Writer)) string {
	oldColorOutput := color.Output
	var buf bytes.Buffer
	color.Output = &buf

	f(&buf)

	color.Output = oldColorOutput
	return stripANSI(buf.String())
}

func TestShowColoredDiff_NoChanges(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nline2\nline3"

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, nil)
	})

	if !strings.Contains(output, "(net 0)") {
		t.Errorf("Expected summary to contain '(net 0)', got: %s", output)
	}
}

func TestShowColoredDiff_WithChanges(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nmodified\nline3"

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, nil)
	})

	// stats line should contain removed/added counts
	if !strings.Contains(output, "-1") || !strings.Contains(output, "+1") {
		t.Errorf("Expected stats line with counts, got: %s", output)
	}
}

func TestShowColoredDiff_Addition(t *testing.T) {
	old := "line1\nline2"
	new := "line1\nline2\nline3"

	opts := &DiffOptions{
		ContextLines:  3,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: 50,
	}

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, opts)
	})

	if !strings.Contains(output, "(net +1)") {
		t.Errorf("Expected '(net +1)' in summary for net addition, got: %s", output)
	}
}

func TestShowColoredDiff_Deletion(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nline3"

	opts := &DiffOptions{
		ContextLines:  3,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: 50,
	}

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, opts)
	})

	if !strings.Contains(output, "(net -1)") {
		t.Errorf("Expected '(net -1)' in summary for net deletion, got: %s", output)
	}
}

func TestShowColoredDiff_SideBySide(t *testing.T) {
	old := "line1\nline2"
	new := "line1\nmodified"

	opts := &DiffOptions{
		ContextLines:  3,
		ShowLineNums:  true,
		InlineMode:    false, // Side by side mode
		MaxTotalLines: 50,
	}

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, opts)
	})

	// Before/After表示があること
	if !strings.Contains(output, "Before") || !strings.Contains(output, "After") {
		t.Errorf("Expected Before/After headers in side-by-side mode, got: %s", output)
	}
}

func TestShowColoredDiff_SideBySide_SeparatesBorderAndContentStyling(t *testing.T) {
	var buf bytes.Buffer
	p := diffPrinter{
		out: &buf,
		pal: fileOpPalette{
			AddLine: func(w io.Writer, s string) { _, _ = io.WriteString(w, s) },
			DelLine: func(w io.Writer, s string) { _, _ = io.WriteString(w, s) },
			Hunk:    func(w io.Writer, s string) { _, _ = io.WriteString(w, s) },
			Accent:  func(w io.Writer, s string) { _, _ = io.WriteString(w, s) },
			Muted:   func(w io.Writer, s string) { _, _ = io.WriteString(w, "[M]"+s+"[/M]") },
			Border:  func(w io.Writer, s string) { _, _ = io.WriteString(w, "[B]"+s+"[/B]") },
			Context: func(w io.Writer, s string) { _, _ = io.WriteString(w, "[C]"+s+"[/C]") },
		},
	}

	opts := &DiffOptions{ShowLineNums: true, InlineMode: false, MaxTotalLines: 50}
	showSideBySideDiffToWriter(p, []string{"line1"}, []string{"line2"}, opts)
	output := buf.String()

	if !strings.Contains(output, "[B]│ [/B][C]L1    line1") {
		t.Fatalf("expected border and content styling to be separated, got: %q", output)
	}
	if !strings.Contains(output, "[/C][B] │\n[/B]") {
		t.Fatalf("expected trailing border to be styled separately, got: %q", output)
	}
}

func TestShowUnifiedDiff(t *testing.T) {
	diffOutput := `--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-old line
+new line
 line3`

	output := captureOutput(func(w io.Writer) {
		ShowUnifiedDiffToWriter(w, diffOutput)
	})

	// 各行が正しく処理されていること
	if !strings.Contains(output, "---") {
		t.Errorf("Expected --- header, got: %s", output)
	}
	if !strings.Contains(output, "+++") {
		t.Errorf("Expected +++ header, got: %s", output)
	}
	if !strings.Contains(output, "@@") {
		t.Errorf("Expected @@ hunk header, got: %s", output)
	}
}

func TestShowColoredDiffWithRuntime_UsesInjectedWriter(t *testing.T) {
	runtime := NewRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)

	ShowColoredDiffWithRuntime(runtime, "before\nline2", "after\nline2", nil)

	output := stripANSI(out.String())
	if !strings.Contains(output, "(net 0)") {
		t.Fatalf("expected injected output to contain stats, got %q", output)
	}
}

func TestShowUnifiedDiffToWriter_UsesInjectedWriter(t *testing.T) {
	var buf bytes.Buffer

	ShowUnifiedDiffToWriter(&buf, "--- a/file.txt\n+++ b/file.txt\n@@ -1 +1 @@\n-old\n+new\n")

	output := stripANSI(buf.String())
	if !strings.Contains(output, "--- a/file.txt") {
		t.Fatalf("expected injected output to contain unified diff header, got %q", output)
	}
	if !strings.Contains(output, "+new") {
		t.Fatalf("expected injected output to contain added line, got %q", output)
	}
}

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long line", 10, "this is..."},
		{"日本語テスト", 5, "日本..."},
		{"", 10, ""},
	}

	for _, tc := range tests {
		result := truncateLine(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncateLine(%q, %d) = %q, want %q",
				tc.input, tc.maxLen, result, tc.expected)
		}
	}
}

func TestMaxMin(t *testing.T) {
	if max(3, 5) != 5 {
		t.Error("max(3, 5) should be 5")
	}
	if max(5, 3) != 5 {
		t.Error("max(5, 3) should be 5")
	}
	if min(3, 5) != 3 {
		t.Error("min(3, 5) should be 3")
	}
	if min(5, 3) != 3 {
		t.Error("min(5, 3) should be 3")
	}
}

func TestShowColoredDiff_MaxTotalLines(t *testing.T) {
	// Create content that exceeds MaxTotalLines
	var oldLines, newLines []string
	for i := 0; i < 100; i++ {
		oldLines = append(oldLines, "old line")
		newLines = append(newLines, "new line")
	}
	old := strings.Join(oldLines, "\n")
	new := strings.Join(newLines, "\n")

	opts := &DiffOptions{
		ContextLines:  1,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: 10, // Limit to 10 lines
	}

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, opts)
	})

	// Should contain truncation message
	if !strings.Contains(output, "more lines") {
		t.Errorf("Expected truncation message, got: %s", output)
	}
}

func TestShowColoredDiff_SideBySide_MaxTotalLines(t *testing.T) {
	// Create content that exceeds MaxTotalLines
	var oldLines, newLines []string
	for i := 0; i < 50; i++ {
		oldLines = append(oldLines, "old line content here")
		newLines = append(newLines, "new line content here")
	}
	old := strings.Join(oldLines, "\n")
	new := strings.Join(newLines, "\n")

	opts := &DiffOptions{
		ContextLines:  1,
		ShowLineNums:  true,
		InlineMode:    false, // Side by side mode
		MaxTotalLines: 10,    // Limit to 10 lines
	}

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, opts)
	})

	// Should contain truncation message
	if !strings.Contains(output, "omitted") {
		t.Errorf("Expected 'omitted' truncation message in side-by-side mode, got: %s", output)
	}
}

func TestShowColoredDiff_NoLineNumbers(t *testing.T) {
	old := "line1\nline2"
	new := "line1\nmodified"

	opts := &DiffOptions{
		ContextLines:  3,
		ShowLineNums:  false, // Disable line numbers
		InlineMode:    true,
		MaxTotalLines: 50,
	}

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, opts)
	})

	// Should not contain line numbers like "   1 "
	if strings.Contains(output, "   1 -") || strings.Contains(output, "   1 +") {
		t.Errorf("Expected no line numbers when ShowLineNums=false, got: %s", output)
	}
}

func TestShowColoredDiff_LargeDeletion(t *testing.T) {
	// Test case where old is much longer than new
	old := "line1\nline2\nline3\nline4\nline5\nline6\nline7"
	new := "line1"

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, nil)
	})

	// Should show net negative
	if !strings.Contains(output, "net") {
		t.Errorf("Expected net summary, got: %s", output)
	}
}

func TestShowColoredDiff_LargeAddition(t *testing.T) {
	// Test case where new is much longer than old
	old := "line1"
	new := "line1\nline2\nline3\nline4\nline5\nline6\nline7"

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, nil)
	})

	// Should show net positive
	if !strings.Contains(output, "net +") {
		t.Errorf("Expected positive net summary, got: %s", output)
	}
}

func TestShowUnifiedDiff_EmptyLines(t *testing.T) {
	diffOutput := `--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@

-old
+new

 context`

	output := captureOutput(func(w io.Writer) {
		ShowUnifiedDiffToWriter(w, diffOutput)
	})

	// Should process empty lines without error
	if !strings.Contains(output, "old") {
		t.Errorf("Expected content in output, got: %s", output)
	}
}

func TestShowInlineDiff_SkipContext(t *testing.T) {
	// Test the gap/skip logic with many unchanged lines between changes
	old := "change1\nunchanged1\nunchanged2\nunchanged3\nunchanged4\nunchanged5\nunchanged6\nunchanged7\nunchanged8\nchange2"
	new := "modified1\nunchanged1\nunchanged2\nunchanged3\nunchanged4\nunchanged5\nunchanged6\nunchanged7\nunchanged8\nmodified2"

	opts := &DiffOptions{
		ContextLines:  1, // Only show 1 line of context
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: 100,
	}

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, opts)
	})

	// Should show "..." for skipped sections
	if !strings.Contains(output, "...") {
		t.Errorf("Expected '...' for skipped context, got: %s", output)
	}
}

func TestShowPatchToWriter(t *testing.T) {
	patchOutput := `*** Begin Patch
*** Update File: test.go
@@ func main()
 -old line
 +new line
*** Add File: new.txt
+new file content
*** Delete File: old.txt
*** End Patch`

	output := captureOutput(func(w io.Writer) {
		ShowPatchToWriter(w, patchOutput)
	})

	// ***行が存在すること
	if !strings.Contains(output, "*** Begin Patch") {
		t.Errorf("Expected '*** Begin Patch' header, got: %s", output)
	}
	if !strings.Contains(output, "*** Update File: test.go") {
		t.Errorf("Expected '*** Update File: test.go', got: %s", output)
	}
	if !strings.Contains(output, "*** Add File: new.txt") {
		t.Errorf("Expected '*** Add File: new.txt', got: %s", output)
	}
	if !strings.Contains(output, "*** Delete File: old.txt") {
		t.Errorf("Expected '*** Delete File: old.txt', got: %s", output)
	}

	// @@行が存在すること
	if !strings.Contains(output, "@@ func main()") {
		t.Errorf("Expected '@@ func main()' hunk header, got: %s", output)
	}

	// -行が存在すること
	if !strings.Contains(output, "-old line") {
		t.Errorf("Expected '-old line' removed line, got: %s", output)
	}

	// +行が存在すること
	if !strings.Contains(output, "+new line") {
		t.Errorf("Expected '+new line' added line, got: %s", output)
	}
	if !strings.Contains(output, "+new file content") {
		t.Errorf("Expected '+new file content' added line, got: %s", output)
	}
}

// isAllDashes は文字列が全て '─' (U+2500) で構成されているかを返す。
func isAllDashes(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r != '─' {
			return false
		}
		i += size
	}
	return true
}

func TestShowColoredDiff_StructuralLayout(t *testing.T) {
	old := "aaa\nbbb\nccc\nddd\neee\nfff\nggg\nhhh\niii\njjj"
	new := "aaa\nBBB\nccc\nddd\neee\nfff\nggg\nHHH\niii\njjj"
	opts := &DiffOptions{ContextLines: 1, ShowLineNums: true, InlineMode: true, MaxTotalLines: 100}

	output := captureOutput(func(w io.Writer) {
		ShowColoredDiffToWriter(w, old, new, opts)
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")

	hasDivider := false
	hasStats := false
	hasDel := false
	hasAdd := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isAllDashes(trimmed) && len(trimmed) > 10 {
			hasDivider = true
		}
		if strings.Contains(line, "-2") && strings.Contains(line, "+2") {
			hasStats = true
		}
		if strings.Contains(line, "- bbb") || strings.Contains(line, "- hhh") {
			hasDel = true
		}
		if strings.Contains(line, "+ BBB") || strings.Contains(line, "+ HHH") {
			hasAdd = true
		}
	}

	if !hasDivider {
		t.Error("missing divider line")
	}
	if !hasStats {
		t.Error("missing stats line")
	}
	if !hasDel {
		t.Error("missing deletion lines")
	}
	if !hasAdd {
		t.Error("missing addition lines")
	}

	// old heavy header must NOT be present
	if strings.Contains(output, "Diff /") {
		t.Error("old heavy header still present")
	}
	if strings.Contains(output, "Summary") {
		t.Error("old Summary heading still present")
	}
}
