package ui

import (
	"strings"
	"testing"
)

func TestCollapseOutput_ShortOutput(t *testing.T) {
	output := "line1\nline2\nline3"
	result := CollapseOutput(output, 5)

	// 5行以下なので折りたたみなし
	if result != output {
		t.Errorf("Expected no collapse for short output, got: %s", result)
	}
}

func TestCollapseOutput_LongOutput(t *testing.T) {
	lines := make([]string, 10)
	for i := 0; i < 10; i++ {
		lines[i] = "line"
	}
	output := strings.Join(lines, "\n")

	result := CollapseOutput(output, 5)

	// 最初の5行 + 折りたたみ表示
	if !strings.Contains(result, "... +5 lines") {
		t.Errorf("Expected collapse indicator, got: %s", result)
	}

	// 最初の5行が含まれている
	resultLines := strings.Split(result, "\n")
	if len(resultLines) != 6 { // 5行 + 折りたたみ行
		t.Errorf("Expected 6 lines (5 visible + 1 indicator), got %d", len(resultLines))
	}
}

func TestCollapseOutput_ExactMaxLines(t *testing.T) {
	output := "line1\nline2\nline3\nline4\nline5"
	result := CollapseOutput(output, 5)

	// ちょうど5行なので折りたたみなし
	if result != output {
		t.Errorf("Expected no collapse for exact max lines, got: %s", result)
	}
}

func TestCollapseOutputWithPrefix(t *testing.T) {
	output := "line1\nline2\nline3\nline4\nline5\nline6\nline7"
	result := CollapseOutputWithPrefix(output, "  ", 3)

	// プレフィックス付きで3行 + 折りたたみ
	if !strings.HasPrefix(result, "  line1") {
		t.Errorf("Expected prefix on first line, got: %s", result)
	}

	if !strings.Contains(result, "... +4 lines") {
		t.Errorf("Expected collapse indicator, got: %s", result)
	}
}

func TestFormatToolOutput(t *testing.T) {
	output := "line1\nline2\nline3\nline4\nline5\nline6"
	result := FormatToolOutput(output, 3)

	// Claude Code 風の表示: ⎿ で始まる
	if !strings.HasPrefix(result, "⎿  line1") {
		t.Errorf("Expected Claude Code style prefix, got: %s", result)
	}

	// 折りたたみ表示
	if !strings.Contains(result, "... +3 lines") {
		t.Errorf("Expected collapse indicator, got: %s", result)
	}
}

func TestFormatToolOutput_EmptyOutput(t *testing.T) {
	result := FormatToolOutput("", 5)

	if result != "" {
		t.Errorf("Expected empty string for empty output, got: %s", result)
	}
}

func TestFormatToolOutput_SingleLine(t *testing.T) {
	output := "single line"
	result := FormatToolOutput(output, 5)

	expected := "⎿  single line"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestGetMaxVisibleLines_Default(t *testing.T) {
	// 環境変数とconfigがない場合のデフォルト
	lines := GetMaxVisibleLines()

	if lines <= 0 {
		t.Errorf("Expected positive default, got %d", lines)
	}
}
