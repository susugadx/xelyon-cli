package tools

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

// ConfirmInteractive は拡張確認プロンプト (y/n/c)
// c (comment) を選択すると複数行コメントを入力できる
// 外部パッケージからも使用可能なようにエクスポート
// NOTE: テスト時は setupTestConfirmInteractive() でモックされる
var ConfirmInteractive = func(message string) ConfirmResult {
	reader := bufio.NewReader(os.Stdin)

	for {
		yellow.Printf("%s [y/n/c]: ", message)

		response, err := reader.ReadString('\n')
		if err != nil {
			// EOF時はnoを返して終了
			return ConfirmResult{Action: "no"}
		}
		response = strings.ToLower(strings.TrimSpace(response))

		// 空入力は無視してリトライ（誤Enter防止）
		if response == "" {
			continue
		}

		// y/yes/はい → 実行
		if response == "y" || response == "yes" || response == "ｙ" || response == "はい" {
			return ConfirmResult{Action: "yes"}
		}

		// n/no/いいえ → キャンセル
		if response == "n" || response == "no" || response == "ｎ" || response == "いいえ" {
			return ConfirmResult{Action: "no"}
		}

		// c/comment → コメント入力モード
		if response == "c" || response == "comment" || response == "コメント" {
			comment, image := readMultiLineComment(reader)
			return ConfirmResult{Action: "comment", Comment: comment, Image: image}
		}

		// その他 → リトライ
		yellow.Println("Invalid input. Please enter y/n/c.")
	}
}

// readMultiLineComment は複数行コメントを読み取る
// 空行2回で入力終了
// image:プレフィックスで画像を指定可能
// /paste（または /p）で Paste Mode を起動して長文を挿入可能
func readMultiLineComment(reader *bufio.Reader) (string, *ImageData) {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("💬 Enter your comment (press Enter twice to finish):")
	cyan.Println("   Tip: Use 'image:/path/to/file.png' to attach an image")
	cyan.Println("   Tip: Use '/paste' (or /p) to enter Paste Mode and insert long text")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

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
		yellow.Print("> ")

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
					red.Printf("⚠️  Paste Mode error: %v\n", err)
					continue
				}
				if cancelled {
					yellow.Println("❌ Cancelled - input discarded")
					continue
				}

				content = strings.ReplaceAll(content, "\r\n", "\n")
				content = strings.TrimRight(content, "\n")
				if content == "" {
					yellow.Println("⚠️ No content captured")
					continue
				}

				pastedLines := strings.Split(content, "\n")
				for _, pl := range pastedLines {
					lines = append(lines, pl)
					totalBytes += len(pl) + 1
				}
				emptyLineCount = 0

				if len(lines) >= maxLines {
					yellow.Printf("⚠️ Max lines (%d) reached\n", maxLines)
					goto done
				}
				if totalBytes >= maxBytes {
					yellow.Printf("⚠️ Max size (%d bytes) reached\n", maxBytes)
					goto done
				}

				green.Printf("✅ Inserted %d lines from Paste Mode into comment\n", len(pastedLines))
				cyan.Println("💬 Back to comment input (finish with empty line x2)")
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
					red.Printf("⚠️  Failed to load image: %v\n", err)
					lines = append(lines, line)
					totalBytes += len(line) + 1
				} else {
					imageData = img
					green.Printf("🖼️  Image loaded: %s (%s)\n", img.Path, FormatSize(img.Size))
				}
			} else {
				lines = append(lines, line)
				totalBytes += len(line) + 1
			}

			if len(lines) >= maxLines {
				yellow.Printf("⚠️ Max lines (%d) reached\n", maxLines)
				goto done
			}
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

	comment := strings.Join(lines, "\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return comment, imageData
}
