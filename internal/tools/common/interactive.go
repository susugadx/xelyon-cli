package common

import (
	"bufio"
	"os"
	"strings"

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
	return ConfirmInteractiveWithIO(ui.DefaultPromptIO(), message)
}

// ConfirmInteractiveWithIO は入出力先を指定した拡張確認プロンプトを実行する。
func ConfirmInteractiveWithIO(promptIO ui.PromptIO, message string) ConfirmResult {
	return ConfirmInteractiveRequestWithIO(promptIO, ui.PromptRequest{
		Kind:         ui.PromptKindConfirm,
		Message:      message,
		AllowComment: true,
	})
}

// ConfirmInteractiveRequestWithIO は PromptRequest を指定して拡張確認プロンプトを実行する。
func ConfirmInteractiveRequestWithIO(promptIO ui.PromptIO, req ui.PromptRequest) ConfirmResult {
	promptIO = ui.NormalizePromptIO(promptIO)
	req.Kind = ui.PromptKindConfirm
	if prompter := promptIO.Prompter(); prompter != nil {
		resp, err := prompter.Prompt(promptIO.PromptContext(), req)
		if err != nil || resp.Cancelled {
			return ConfirmResult{Action: "no"}
		}
		switch resp.Action {
		case ui.PromptActionYes:
			return ConfirmResult{Action: "yes"}
		case ui.PromptActionComment:
			comment, image := parseMultiLineCommentText(promptIO, resp.Text)
			return ConfirmResult{Action: "comment", Comment: comment, Image: image}
		default:
			return ConfirmResult{Action: "no"}
		}
	}

	// 矢印キー選択UIを使用
	result, err := ui.ConfirmSelectorRequestWithIO(promptIO, req)
	if err != nil {
		// キャンセルまたはエラー時はnoを返す
		return ConfirmResult{Action: "no"}
	}

	// comment選択時はコメント入力モードへ
	if result == "comment" {
		comment, image := ReadMultiLineCommentWithIO(promptIO)
		return ConfirmResult{Action: "comment", Comment: comment, Image: image}
	}
	return ConfirmResult{Action: result}
}

// ReadMultiLineComment は複数行コメントを読み取る
// 空行2回で入力終了
// image:プレフィックスで画像を指定可能
func ReadMultiLineComment(reader *bufio.Reader) (string, *ImageData) {
	return readMultiLineComment(ui.DefaultPromptIO(), reader)
}

// ReadMultiLineCommentWithIO は入出力先を指定して複数行コメントを読み取る。
func ReadMultiLineCommentWithIO(promptIO ui.PromptIO) (string, *ImageData) {
	promptIO = ui.NormalizePromptIO(promptIO)
	return readMultiLineComment(promptIO, nil)
}

func readMultiLineComment(promptIO ui.PromptIO, reader *bufio.Reader) (string, *ImageData) {
	out := NewOutput(promptIO.Out, promptIO.Err)
	out.Cyan.Println("--------------------------------------------")
	out.Cyan.Println("Enter your comment (press Enter twice to finish):")
	out.Cyan.Println("   Tip: Use 'image:/path/to/file.png' to attach an image")
	out.Cyan.Println("--------------------------------------------")

	cfg := config.DefaultConfig()
	maxLines := cfg.Paste.MaxLines
	maxBytes := cfg.Paste.MaxBytes

	var lines []string
	var imageData *ImageData
	emptyLineCount := 0
	totalBytes := 0
	if promptIO.Reader == nil && reader == nil {
		reader = promptIO.BufioReader()
	}

	for {
		out.Yellow.Print("> ")

		var line string
		var err error
		if promptIO.Reader != nil {
			line, err = promptIO.Reader.ReadSimpleLine()
		} else {
			line, err = reader.ReadString('\n')
			if err == nil {
				line = strings.TrimRight(line, "\r\n")
				line = ui.StripBracketedPaste(line)
			}
		}
		if err != nil {
			goto done
		}

		trimmed := strings.TrimSpace(line)

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
				out.Red.Printf("Failed to load image: %v\n", err)
				lines = append(lines, line)
				totalBytes += len(line) + 1
			} else {
				imageData = img
				out.Green.Printf("Image loaded: %s (%s)\n", img.Path, FormatSize(img.Size))
			}
		} else {
			lines = append(lines, line)
			totalBytes += len(line) + 1
		}

		if len(lines) >= maxLines {
			out.Yellow.Printf("Max lines (%d) reached\n", maxLines)
			goto done
		}
		if totalBytes >= maxBytes {
			out.Yellow.Printf("Max size (%d bytes) reached\n", maxBytes)
			goto done
		}
	}

done:
	// 末尾の空行を削除
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	comment := strings.Join(lines, "\n")
	out.Cyan.Println("--------------------------------------------")
	return comment, imageData
}

func parseMultiLineCommentText(promptIO ui.PromptIO, text string) (string, *ImageData) {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := NewOutput(promptIO.Out, promptIO.Err)
	cfg := config.DefaultConfig()
	maxLines := cfg.Paste.MaxLines
	maxBytes := cfg.Paste.MaxBytes

	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	inputLines := strings.Split(normalized, "\n")
	lines := make([]string, 0, len(inputLines))
	var imageData *ImageData
	totalBytes := 0

	for _, line := range inputLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "image:") {
			imagePath := strings.TrimSpace(strings.TrimPrefix(trimmed, "image:"))
			img, err := LoadImage(imagePath)
			if err != nil {
				out.Red.Printf("Failed to load image: %v\n", err)
				lines = append(lines, line)
				totalBytes += len(line) + 1
			} else {
				imageData = img
				out.Green.Printf("Image loaded: %s (%s)\n", img.Path, FormatSize(img.Size))
			}
		} else {
			lines = append(lines, line)
			totalBytes += len(line) + 1
		}

		if len(lines) >= maxLines || totalBytes >= maxBytes {
			break
		}
	}

	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n"), imageData
}
