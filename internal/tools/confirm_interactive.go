package tools

import (
	"bufio"
	"os"
	"strings"
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
var ConfirmInteractive = func(message string) ConfirmResult {
	yellow.Printf("%s [y/n/c]: ", message)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return ConfirmResult{Action: "no"}
	}
	response = strings.ToLower(strings.TrimSpace(response))

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

	// その他 → デフォルトはキャンセル
	yellow.Println("Invalid input. Assuming 'no'.")
	return ConfirmResult{Action: "no"}
}

// readMultiLineComment は複数行コメントを読み取る
// 空行2回で入力終了
// image:プレフィックスで画像を指定可能
func readMultiLineComment(reader *bufio.Reader) (string, *ImageData) {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("💬 Enter your comment (press Enter twice to finish):")
	cyan.Println("   Tip: Use 'image:/path/to/file.png' to attach an image")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var lines []string
	var imageData *ImageData
	emptyLineCount := 0

	for {
		yellow.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimRight(line, "\r\n")

		// 空行チェック
		if strings.TrimSpace(line) == "" {
			emptyLineCount++
			if emptyLineCount >= 2 {
				// 空行2回で終了
				break
			}
			// 最初の空行は保持
			lines = append(lines, line)
		} else {
			emptyLineCount = 0

			// image: プレフィックスを検出
			if strings.HasPrefix(strings.TrimSpace(line), "image:") {
				imagePath := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "image:"))

				// 画像を読み込み
				img, err := LoadImage(imagePath)
				if err != nil {
					red.Printf("⚠️  Failed to load image: %v\n", err)
					// エラーでも続行（テキストとして残す）
					lines = append(lines, line)
				} else {
					imageData = img
					green.Printf("🖼️  Image loaded: %s (%s)\n", img.Path, FormatSize(img.Size))
					// image:行はコメントには含めない
				}
			} else {
				lines = append(lines, line)
			}
		}
	}

	// 末尾の空行を削除
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	comment := strings.Join(lines, "\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return comment, imageData
}
