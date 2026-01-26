package common

import (
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ConfirmResult は確認結果
type ConfirmResult struct {
	Action  string     // "yes", "no", "comment"
	Comment string     // "comment" の場合のコメント内容
	Image   *ImageData // 画像データ（オプション）
}

// IsInteractiveModeEnabled は対話的確認モードが有効かチェック
// デフォルトは有効。
//
// 無効化したい場合は以下のいずれかを設定:
//   - XELYON_INTERACTIVE_CONFIRM=0
//   - XELYON_INTERACTIVE_CONFIRM=false
func IsInteractiveModeEnabled() bool {
	v := os.Getenv("XELYON_INTERACTIVE_CONFIRM")
	switch v {
	case "0", "false", "FALSE":
		return false
	default:
		return true
	}
}

// ConfirmInteractive は拡張確認プロンプト (矢印キー選択UI)
// Yes/No/Comment を矢印キーで選択、Enterで確定
// y/n/c のショートカットキーも引き続きサポート
// 外部パッケージからも使用可能なようにエクスポート
// NOTE: テスト時は環境変数 XELYON_INTERACTIVE_CONFIRM=0 で無効化される
var ConfirmInteractive = func(message string) ConfirmResult {
	// 矢印キー選択UIを使用
	result, err := ui.ConfirmSelector(message)
	if err != nil {
		// キャンセルまたはエラー時はnoを返す
		return ConfirmResult{Action: "no"}
	}

	// comment選択時はコメント入力モードへ
	if result == "comment" {
		reader := bufio.NewReader(os.Stdin)
		comment, image := ReadMultiLineComment(reader)
		return ConfirmResult{Action: "comment", Comment: comment, Image: image}
	}

	return ConfirmResult{Action: result}
}

// ReadMultiLineComment は複数行コメントを読み取る
// 空行2回で入力終了
// image:プレフィックスで画像を指定可能
// /paste（または /p）で Paste Mode を起動して長文を挿入可能
func ReadMultiLineComment(reader *bufio.Reader) (string, *ImageData) {
	Cyan.Println("--------------------------------------------")
	Cyan.Println("Enter your comment (press Enter twice to finish):")
	Cyan.Println("   Tip: Use 'image:/path/to/file.png' to attach an image")
	Cyan.Println("   Tip: Use '/paste' (or /p) to enter Paste Mode and insert long text")
	Cyan.Println("--------------------------------------------")

	cfg := config.GetGlobalConfig()
	maxLines := cfg.Paste.MaxLines
	maxBytes := cfg.Paste.MaxBytes
	timeout := time.Duration(cfg.Paste.TimeoutSeconds) * time.Second

	var lines []string
	var imageData *ImageData
	emptyLineCount := 0
	totalBytes := 0

	type readResult struct {
		line string
		err  error
	}
	inputChan := make(chan readResult, 1)

	for {
		Yellow.Print("> ")

		go func() {
			line, err := reader.ReadString('\n')
			inputChan <- readResult{line: line, err: err}
		}()

		select {
		case result := <-inputChan:
			if result.err != nil {
				goto done
			}

			line := strings.TrimRight(result.line, "\r\n")
			trimmed := strings.TrimSpace(line)

			// /paste: enter paste mode and insert captured content
			if trimmed == "/p" || trimmed == "/paste" {
				pm := ui.NewPasteMode(cfg.Paste)
				content, cancelled, err := pm.Capture(os.Stdin, os.Stdout)
				if err != nil {
					Red.Printf("Paste Mode error: %v\n", err)
					continue
				}
				if cancelled {
					Yellow.Println("Cancelled - input discarded")
					continue
				}

				content = strings.ReplaceAll(content, "\r\n", "\n")
				content = strings.TrimRight(content, "\n")
				if content == "" {
					Yellow.Println("No content captured")
					continue
				}

				pastedLines := strings.Split(content, "\n")
				for _, pl := range pastedLines {
					lines = append(lines, pl)
					totalBytes += len(pl) + 1
				}
				emptyLineCount = 0

				if len(lines) >= maxLines {
					Yellow.Printf("Max lines (%d) reached\n", maxLines)
					goto done
				}
				if totalBytes >= maxBytes {
					Yellow.Printf("Max size (%d bytes) reached\n", maxBytes)
					goto done
				}

				Green.Printf("Inserted %d lines from Paste Mode into comment\n", len(pastedLines))
				Cyan.Println("Back to comment input (finish with empty line x2)")
				continue
			}

			// 空行チェック（Enter 2回で終了）
			if trimmed == "" {
				emptyLineCount++
				if emptyLineCount >= 2 {
					goto done
				}
				lines = append(lines, line)
				totalBytes += len(line) + 1
				continue
			}

			emptyLineCount = 0

			// image: プレフィックスを検出
			if strings.HasPrefix(trimmed, "image:") {
				imagePath := strings.TrimSpace(strings.TrimPrefix(trimmed, "image:"))

				img, err := LoadImage(imagePath)
				if err != nil {
					Red.Printf("Failed to load image: %v\n", err)
					lines = append(lines, line)
					totalBytes += len(line) + 1
				} else {
					imageData = img
					Green.Printf("Image loaded: %s (%s)\n", img.Path, FormatSize(img.Size))
				}
			} else {
				lines = append(lines, line)
				totalBytes += len(line) + 1
			}

			if len(lines) >= maxLines {
				Yellow.Printf("Max lines (%d) reached\n", maxLines)
				goto done
			}
			if totalBytes >= maxBytes {
				Yellow.Printf("Max size (%d bytes) reached\n", maxBytes)
				goto done
			}

		case <-time.After(timeout):
			Yellow.Printf("Timeout - no input for %d seconds\n", int(timeout.Seconds()))
			goto done
		}
	}

done:
	// 末尾の空行を削除
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	comment := strings.Join(lines, "\n")
	Cyan.Println("--------------------------------------------")
	return comment, imageData
}
