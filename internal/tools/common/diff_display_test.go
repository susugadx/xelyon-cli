package common

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestShowDiffWithOutputAndConfig(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)
	cfg := config.DefaultConfig()

	oldStr := "line 1\nline 2\nline 3"
	newStr := "line 1\nline 2 modified\nline 3"

	ShowDiffWithOutputAndConfig(out, cfg, oldStr, newStr, "test.go")

	result := buf.String()
	if !strings.Contains(result, "test.go") {
		t.Errorf("expected filename in output, got: %q", result)
	}
	// diff 出力に何らかの内容が含まれることを確認
	if len(result) == 0 {
		t.Error("expected non-empty diff output")
	}
}

func TestShowImprovedDiffWithOutputAndConfig(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)
	cfg := config.DefaultConfig()

	oldStr := "hello\nworld"
	newStr := "hello\ngo world"

	ShowImprovedDiffWithOutputAndConfig(out, cfg, oldStr, newStr)

	result := buf.String()
	if len(result) == 0 {
		t.Error("expected non-empty diff output")
	}
}

func TestShowPreviewWithOutputAndConfig(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)
	cfg := config.DefaultConfig()

	content := "alpha\nbeta\ngamma"
	ShowPreviewWithOutputAndConfig(out, cfg, content)

	result := buf.String()
	if !strings.Contains(result, "1: alpha") {
		t.Errorf("expected line-numbered output for first line, got: %q", result)
	}
	if !strings.Contains(result, "2: beta") {
		t.Errorf("expected line-numbered output for second line, got: %q", result)
	}
	if !strings.Contains(result, "3: gamma") {
		t.Errorf("expected line-numbered output for third line, got: %q", result)
	}
}

func TestShowPreview_Truncation(t *testing.T) {
	color.NoColor = true

	cfg := config.DefaultConfig()
	cfg.Diff.MaxTotalLines = 3

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	ShowPreviewWithOutputAndConfig(out, cfg, strings.Join(lines, "\n"))

	result := buf.String()
	// MaxTotalLines=3 なので 4行目以降は省略される
	if !strings.Contains(result, "... (2 more lines)") {
		t.Errorf("expected truncation notice '... (2 more lines)', got: %q", result)
	}
	// 切り詰め前の行は表示されるはず
	if !strings.Contains(result, "1: line1") {
		t.Errorf("expected first line in output, got: %q", result)
	}
}

func TestShowDiff_EmptyStrings(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)
	cfg := config.DefaultConfig()

	// 空文字列同士: パニックしないことを確認
	ShowDiffWithOutputAndConfig(out, cfg, "", "", "empty.go")

	result := buf.String()
	if !strings.Contains(result, "empty.go") {
		t.Errorf("expected filename in output even for empty strings, got: %q", result)
	}

	// old が空、new が非空
	buf.Reset()
	ShowDiffWithOutputAndConfig(out, cfg, "", "new content", "new.go")
	result = buf.String()
	if !strings.Contains(result, "new.go") {
		t.Errorf("expected filename in output, got: %q", result)
	}

	// old が非空、new が空
	buf.Reset()
	ShowDiffWithOutputAndConfig(out, cfg, "old content", "", "deleted.go")
	result = buf.String()
	if !strings.Contains(result, "deleted.go") {
		t.Errorf("expected filename in output, got: %q", result)
	}
}
