package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// handlePasteCommand はペーストモードを開始し、複数行入力を受け付ける
// WSLなどBracketed Paste Modeが動作しない環境向け
func handlePasteCommand(agent *Agent, args []string) bool {
	cfg := config.GetGlobalConfig()

	pm := ui.NewPasteMode(cfg.Paste)

	// 共有リーダーを使用（バッファ競合を回避）
	var content string
	var cancelled bool
	var err error
	if agent.mlReader != nil {
		content, cancelled, err = pm.CaptureWithReader(agent.mlReader.Reader(), os.Stdout)
	} else {
		content, cancelled, err = pm.Capture(os.Stdin, os.Stdout)
	}
	if err != nil {
		red.Printf("Read error: %v\n", err)
		return true
	}
	if cancelled {
		yellow.Println("❌ Cancelled - input discarded")
		return true
	}
	if content == "" {
		yellow.Println("⚠️ No content captured")
		return true
	}

	sizeKB := float64(len(content)) / 1024
	linesCount := 1
	if strings.Contains(content, "\n") {
		linesCount = len(strings.Split(content, "\n"))
	}

	fmt.Println()
	green.Printf("✅ Captured %d lines (%.1f KB)\n", linesCount, sizeKB)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// エージェントに入力を渡す（chat メソッドを呼び出す）
	agent.chat(content)
	return true
}
