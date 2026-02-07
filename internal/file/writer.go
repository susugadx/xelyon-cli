package file

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// stripBracketedPaste removes bracketed paste escape sequences from input.
func stripBracketedPaste(input string) string {
	input = strings.ReplaceAll(input, "\x1b[200~", "")
	input = strings.ReplaceAll(input, "\x1b[201~", "")
	input = strings.ReplaceAll(input, "^[[200~", "")
	input = strings.ReplaceAll(input, "^[[201~", "")
	return input
}

func WriteFile(path string, content string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// ディレクトリがなければ作成
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(absPath, []byte(content), 0644)
}

func ConfirmApply(filename string, newContent string) bool {
	fmt.Println("\n" + strings.Repeat("─", 50))
	fmt.Printf("📝 ファイルを更新: %s\n", filename)
	fmt.Println(strings.Repeat("─", 50))

	// 新しい内容をプレビュー（最初の20行）
	lines := strings.Split(newContent, "\n")
	preview := lines
	if len(lines) > 20 {
		preview = lines[:20]
		fmt.Println(strings.Join(preview, "\n"))
		fmt.Printf("\n... 他 %d 行 ...\n", len(lines)-20)
	} else {
		fmt.Println(strings.Join(preview, "\n"))
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Print("適用しますか？ (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = stripBracketedPaste(input)
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}

func ExtractCodeBlock(response string) string {
	// ```で囲まれたコードブロックを抽出
	lines := strings.Split(response, "\n")
	var codeLines []string
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// コードブロック終了
				break
			} else {
				// コードブロック開始
				inCodeBlock = true
				continue
			}
		}
		if inCodeBlock {
			codeLines = append(codeLines, line)
		}
	}

	if len(codeLines) > 0 {
		return strings.Join(codeLines, "\n")
	}
	return ""
}
