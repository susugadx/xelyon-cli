package ui

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"

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

// captureOutput はShowColoredDiffの出力をキャプチャするヘルパー
func captureOutput(f func()) string {
	// 標準出力とcolorの出力先をリダイレクト
	oldStdout := os.Stdout
	oldColorOutput := color.Output
	r, w, _ := os.Pipe()
	os.Stdout = w
	color.Output = w

	f()

	w.Close()
	os.Stdout = oldStdout
	color.Output = oldColorOutput

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return stripANSI(buf.String())
}

func TestShowColoredDiff_NoChanges(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nline2\nline3"

	output := captureOutput(func() {
		ShowColoredDiff(old, new, nil)
	})

	// サマリー行があること（net: 0）
	if !strings.Contains(output, "net: 0") {
		t.Errorf("Expected summary to contain 'net: 0', got: %s", output)
	}
}

func TestShowColoredDiff_WithChanges(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nmodified\nline3"

	output := captureOutput(func() {
		ShowColoredDiff(old, new, nil)
	})

	// 差分表示があること
	if !strings.Contains(output, "Diff") {
		t.Errorf("Expected diff header, got: %s", output)
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

	output := captureOutput(func() {
		ShowColoredDiff(old, new, opts)
	})

	// +1の表示があること（net: +1）
	if !strings.Contains(output, "net: +1") {
		t.Errorf("Expected 'net: +1' in summary for net addition, got: %s", output)
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

	output := captureOutput(func() {
		ShowColoredDiff(old, new, opts)
	})

	// -1の表示があること（net: -1）
	if !strings.Contains(output, "net: -1") {
		t.Errorf("Expected 'net: -1' in summary for net deletion, got: %s", output)
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

	output := captureOutput(func() {
		ShowColoredDiff(old, new, opts)
	})

	// Before/After表示があること
	if !strings.Contains(output, "Before") || !strings.Contains(output, "After") {
		t.Errorf("Expected Before/After headers in side-by-side mode, got: %s", output)
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

	output := captureOutput(func() {
		ShowUnifiedDiff(diffOutput)
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
	if !strings.Contains(output, "Diff / 差分表示") {
		t.Fatalf("expected injected output to contain diff header, got %q", output)
	}
	if !strings.Contains(output, "net: 0") {
		t.Fatalf("expected injected output to contain summary, got %q", output)
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

	output := captureOutput(func() {
		ShowColoredDiff(old, new, opts)
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

	output := captureOutput(func() {
		ShowColoredDiff(old, new, opts)
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

	output := captureOutput(func() {
		ShowColoredDiff(old, new, opts)
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

	output := captureOutput(func() {
		ShowColoredDiff(old, new, nil)
	})

	// Should show net negative
	if !strings.Contains(output, "net:") {
		t.Errorf("Expected net summary, got: %s", output)
	}
}

func TestShowColoredDiff_LargeAddition(t *testing.T) {
	// Test case where new is much longer than old
	old := "line1"
	new := "line1\nline2\nline3\nline4\nline5\nline6\nline7"

	output := captureOutput(func() {
		ShowColoredDiff(old, new, nil)
	})

	// Should show net positive
	if !strings.Contains(output, "net: +") {
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

	output := captureOutput(func() {
		ShowUnifiedDiff(diffOutput)
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

	output := captureOutput(func() {
		ShowColoredDiff(old, new, opts)
	})

	// Should show "..." for skipped sections
	if !strings.Contains(output, "...") {
		t.Errorf("Expected '...' for skipped context, got: %s", output)
	}
}
