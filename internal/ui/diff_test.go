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
	buf.ReadFrom(r)
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
