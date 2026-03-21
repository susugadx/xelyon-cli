package common

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestDisplayPatternNotFoundWithOutput(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	lines := []string{"first line", "second line", "third line", "fourth line", "fifth line"}
	DisplayPatternNotFoundWithOutput(out, "missing_pattern", lines, 3)

	result := buf.String()
	if !strings.Contains(result, "Pattern not found: missing_pattern") {
		t.Errorf("expected 'Pattern not found' message, got: %q", result)
	}
	// maxPreview=3 なので最初の3行が表示される
	if !strings.Contains(result, "1: first line") {
		t.Errorf("expected first line in preview, got: %q", result)
	}
	if !strings.Contains(result, "3: third line") {
		t.Errorf("expected third line in preview, got: %q", result)
	}
	// 残り行数の通知
	if !strings.Contains(result, "... (2 more lines)") {
		t.Errorf("expected truncation notice, got: %q", result)
	}
}

func TestDisplayMultipleMatchesWithOutput(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	lines := []string{"aaa", "bbb", "target", "ccc", "ddd", "eee", "target", "fff"}
	matchIndices := []int{2, 6}

	DisplayMultipleMatchesWithOutput(out, matchIndices, lines)

	result := buf.String()
	// エラーメッセージ
	if !strings.Contains(result, "matches 2 locations") {
		t.Errorf("expected match count message, got: %q", result)
	}
	// マッチ位置が "> " プレフィックスで表示される
	if !strings.Contains(result, ">    3: target") {
		t.Errorf("expected first match highlight at line 3, got: %q", result)
	}
	if !strings.Contains(result, ">    7: target") {
		t.Errorf("expected second match highlight at line 7, got: %q", result)
	}
	// コンテキスト行（マッチの前後）が表示される
	if !strings.Contains(result, "1: aaa") {
		t.Errorf("expected context line before first match, got: %q", result)
	}
}

func TestDisplayContextAroundWithOutput(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	lines := []string{"line0", "line1", "line2", "match_line", "line4", "line5", "line6"}
	DisplayContextAroundWithOutput(out, 3, lines, 2)

	result := buf.String()
	// マッチ行番号の表示
	if !strings.Contains(result, "Pattern found at line 4") {
		t.Errorf("expected 'Pattern found at line 4', got: %q", result)
	}
	// コンテキスト行（matchIdx=3, contextLines=2: 行1〜5を表示）
	if !strings.Contains(result, "2: line1") {
		t.Errorf("expected context line before match, got: %q", result)
	}
	if !strings.Contains(result, ">    4: match_line") {
		t.Errorf("expected highlighted match line, got: %q", result)
	}
	if !strings.Contains(result, "6: line5") {
		t.Errorf("expected context line after match, got: %q", result)
	}
}

func TestDisplayContentToInsertWithOutput(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	content := "inserted code\nmore code"
	DisplayContentToInsertWithOutput(out, content)

	result := buf.String()
	if !strings.Contains(result, "Content to insert") {
		t.Errorf("expected 'Content to insert' header, got: %q", result)
	}
	if !strings.Contains(result, "inserted code") {
		t.Errorf("expected inserted content in output, got: %q", result)
	}
	if !strings.Contains(result, "more code") {
		t.Errorf("expected all content lines in output, got: %q", result)
	}
}

func TestDisplayMultipleMatches_EmptyIndices(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	// 空のインデックスで呼び出してもパニックしない
	DisplayMultipleMatchesWithOutput(out, []int{}, []string{"line1", "line2"})

	result := buf.String()
	// matchIndices が空なので "matches 0 locations" が表示される
	if !strings.Contains(result, "matches 0 locations") {
		t.Errorf("expected 'matches 0 locations' for empty indices, got: %q", result)
	}
}
