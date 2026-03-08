package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

// handlePasteCommand はペーストモードを開始し、複数行入力を受け付ける
// WSLなどBracketed Paste Modeが動作しない環境向け
func handlePasteCommand(agent *Agent, args []string) bool {
	cfg := agent.cfg()
	out := agent.output()

	pm := ui.NewPasteMode(cfg.Paste)

	// MultilineReader経由で読み取り（raw mode goroutine対応でペーストマーカー抑制）
	var content string
	var cancelled bool
	var err error
	runtimeUI := agent.ui()
	if reader := runtimeUI.PromptReader(); reader != nil {
		content, cancelled, err = pm.CaptureWithMultilineReader(reader, runtimeUI.Output())
	} else {
		content, cancelled, err = pm.Capture(runtimeUI.Input(), runtimeUI.Output())
	}
	if err != nil {
		red.Fprintf(out, "Read error: %v\n", err)
		return true
	}
	if cancelled {
		yellow.Fprintln(out, "❌ Cancelled - input discarded")
		return true
	}
	if content == "" {
		yellow.Fprintln(out, "⚠️ No content captured")
		return true
	}

	sizeKB := float64(len(content)) / 1024
	linesCount := 1
	if strings.Contains(content, "\n") {
		linesCount = len(strings.Split(content, "\n"))
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintf(out, "✅ Captured %d lines (%.1f KB)\n", linesCount, sizeKB)
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)

	// エージェントに入力を渡す（chat メソッドを呼び出す）
	agent.chat(content)
	return true
}
