package agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// handlePasteCommand はペーストモードを開始し、複数行入力を受け付ける
// WSLなどBracketed Paste Modeが動作しない環境向け
func handlePasteCommand(agent *Agent, args []string) bool {
	cfg := config.GetGlobalConfig()
	maxLines := cfg.Paste.MaxLines
	maxBytes := cfg.Paste.MaxBytes
	timeout := time.Duration(cfg.Paste.TimeoutSeconds) * time.Second

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("📝 Paste Mode / ペーストモード")
	cyan.Println("   End: empty line x2, 'END', /end, Ctrl+D")
	cyan.Println("   Cancel: /cancel, /c")
	cyan.Println("   終了: 空行2回, END, /end, Ctrl+D")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	var lines []string
	emptyCount := 0
	totalBytes := 0
	reader := bufio.NewReader(os.Stdin)

	// タイムアウト用チャネル
	type readResult struct {
		line string
		err  error
	}
	inputChan := make(chan readResult, 1)

	for {
		// 非同期で読み取り
		go func() {
			line, err := reader.ReadString('\n')
			inputChan <- readResult{line: line, err: err}
		}()

		select {
		case result := <-inputChan:
			// EOFまたはCtrl+D
			if result.err == io.EOF {
				yellow.Println("\n⚠️ EOF detected (Ctrl+D)")
				goto done
			}
			if result.err != nil {
				red.Printf("Read error: %v\n", result.err)
				return true
			}

			line := strings.TrimRight(result.line, "\r\n")

			// 終了条件チェック
			if line == "" {
				emptyCount++
				if emptyCount >= 2 {
					goto done
				}
				lines = append(lines, line)
				totalBytes += len(line) + 1 // +1 for newline
				continue
			}

			// 明示的終了コマンド
			if line == "END" || line == "/end" {
				goto done
			}

			// キャンセル
			if line == "/cancel" || line == "/c" {
				yellow.Println("❌ Cancelled - input discarded")
				return true
			}

			emptyCount = 0
			lines = append(lines, line)
			totalBytes += len(line) + 1

			// 行数上限チェック
			if len(lines) >= maxLines {
				yellow.Printf("⚠️ Max lines (%d) reached\n", maxLines)
				goto done
			}

			// バイト数上限チェック
			if totalBytes >= maxBytes {
				yellow.Printf("⚠️ Max size (%d bytes) reached\n", maxBytes)
				goto done
			}

		case <-time.After(timeout):
			yellow.Printf("⚠️ Timeout - no input for %d seconds\n", int(timeout.Seconds()))
			goto done
		}
	}

done:
	// 末尾の空行を削除
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		yellow.Println("⚠️ No content captured")
		return true
	}

	content := strings.Join(lines, "\n")
	sizeKB := float64(len(content)) / 1024

	fmt.Println()
	green.Printf("✅ Captured %d lines (%.1f KB)\n", len(lines), sizeKB)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// エージェントに入力を渡す（chat メソッドを呼び出す）
	agent.chat(content)
	return true
}
