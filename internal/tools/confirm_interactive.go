package tools

import (
	"bufio"
	"os"
	"strings"
)

// ConfirmResult は確認結果
type ConfirmResult struct {
	Action  string // "yes", "no", "comment"
	Comment string // "comment" の場合のコメント内容
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
		comment := readMultiLineComment(reader)
		return ConfirmResult{Action: "comment", Comment: comment}
	}

	// その他 → デフォルトはキャンセル
	yellow.Println("Invalid input. Assuming 'no'.")
	return ConfirmResult{Action: "no"}
}

// readMultiLineComment は複数行コメントを読み取る
// 空行2回で入力終了
func readMultiLineComment(reader *bufio.Reader) string {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("💬 Enter your comment (press Enter twice to finish):")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var lines []string
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
			lines = append(lines, line)
		}
	}

	// 末尾の空行を削除
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	comment := strings.Join(lines, "\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return comment
}
