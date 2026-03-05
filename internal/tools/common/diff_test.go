package common

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestShowPreview(t *testing.T) {
	color.NoColor = true
	config.SetGlobalConfig(config.DefaultConfig())
	old := os.Stdout
	oldColorOut := color.Output
	r, w, _ := os.Pipe()
	os.Stdout = w
	color.Output = w

	content := "line 1\nline 2\nline 3"
	ShowPreview(content)

	w.Close()
	os.Stdout = old
	color.Output = oldColorOut

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "   1: line 1") || !strings.Contains(output, "   3: line 3") {
		t.Errorf("ShowPreview output missing content, got: %q", output)
	}
}

func TestShowPreview_LongContent(t *testing.T) {
	color.NoColor = true
	cfg := config.DefaultConfig()
	cfg.Diff.MaxTotalLines = 20
	config.SetGlobalConfig(cfg)
	old := os.Stdout
	oldColorOut := color.Output
	r, w, _ := os.Pipe()
	os.Stdout = w
	color.Output = w

	// 25行のコンテンツ（20行で切り詰められるはず）
	var lines []string
	for i := 1; i <= 25; i++ {
		lines = append(lines, "line")
	}
	ShowPreview(strings.Join(lines, "\n"))

	w.Close()
	os.Stdout = old
	color.Output = oldColorOut

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// 切り詰められた際に表示される行の存在を確認
	if !strings.Contains(output, "line") {
		t.Errorf("ShowPreview should show lines before truncation")
	}
	if !strings.Contains(output, "... (5 more lines)") {
		t.Errorf("ShowPreview should show truncation notice, got: %q", output)
	}
	// カラー出力が抑制されていても、固定文字列部分は出力されるはず
	// 失敗する場合は Printf の挙動に依存するため、最低限セパレータを確認
	if !strings.Contains(output, "--------------------------------------------------") {
		t.Errorf("ShowPreview separator missing")
	}
}
