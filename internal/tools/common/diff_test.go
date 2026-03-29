package common

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestShowPreview(t *testing.T) {
	color.NoColor = true
	oldColorOut := color.Output
	defer func() {
		color.Output = oldColorOut
	}()

	var out bytes.Buffer
	color.Output = &out

	content := "line 1\nline 2\nline 3"
	ShowPreviewWithOutputAndConfig(NewOutput(&out, &out), config.DefaultConfig(), content)

	if !strings.Contains(out.String(), "   1: line 1") || !strings.Contains(out.String(), "   3: line 3") {
		t.Errorf("ShowPreview output missing content, got: %q", out.String())
	}
}

func TestShowPreview_LongContent(t *testing.T) {
	color.NoColor = true
	cfg := config.DefaultConfig()
	cfg.Diff.MaxTotalLines = 20
	oldColorOut := color.Output
	defer func() {
		color.Output = oldColorOut
	}()

	var out bytes.Buffer
	color.Output = &out

	// 25行のコンテンツ（20行で切り詰められるはず）
	var lines []string
	for i := 1; i <= 25; i++ {
		lines = append(lines, "line")
	}
	ShowPreviewWithOutputAndConfig(NewOutput(&out, &out), cfg, strings.Join(lines, "\n"))

	// 切り詰められた際に表示される行の存在を確認
	if !strings.Contains(out.String(), "line") {
		t.Errorf("ShowPreview should show lines before truncation")
	}
	if !strings.Contains(out.String(), "... (5 more lines)") {
		t.Errorf("ShowPreview should show truncation notice, got: %q", out.String())
	}
	// セパレータ（薄い区切り線）が出力されること
	if !strings.Contains(out.String(), "──────────") {
		t.Errorf("ShowPreview separator missing, got: %q", out.String())
	}
}
