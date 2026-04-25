//go:build ignore

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

func main() {
	// docs/commands.md を読み込み
	inputPath := "docs/commands.md"
	content, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", inputPath, err)
		os.Exit(1)
	}

	// 既存のコマンドセクションを検出
	existingCommands := findExistingCommands(string(content))

	// 不足しているコマンドを検出
	var missingCommands []commandcatalog.CommandInfo
	for _, cmd := range commandcatalog.Commands {
		cmdName := cmd.Name
		if len(cmd.Aliases) > 0 {
			// エイリアス付きで検索
			found := false
			for existing := range existingCommands {
				if strings.Contains(existing, cmdName) {
					found = true
					break
				}
			}
			if !found {
				missingCommands = append(missingCommands, cmd)
			}
		} else {
			if !existingCommands[cmdName] {
				missingCommands = append(missingCommands, cmd)
			}
		}
	}

	if len(missingCommands) == 0 {
		fmt.Println("All commands are documented in docs/commands.md")
		return
	}

	// 骨格を生成
	var buf bytes.Buffer
	buf.WriteString("\n## 未ドキュメント化コマンド（自動追加）\n\n")
	buf.WriteString("<!-- TODO: 以下のコマンドに詳細な説明を追加してください -->\n\n")

	for _, cmd := range missingCommands {
		// セクションヘッダー
		header := cmd.Name
		if len(cmd.Aliases) > 0 {
			header += ", " + strings.Join(cmd.Aliases, ", ")
		}
		buf.WriteString(fmt.Sprintf("### `%s`\n\n", header))

		// 説明
		buf.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))

		// 使用例
		usage := cmd.Name
		if cmd.Args != "" {
			usage += " " + cmd.Args
		}
		buf.WriteString("```\n")
		buf.WriteString(fmt.Sprintf("> %s\n", usage))
		buf.WriteString("```\n\n")

		// サブコマンド
		if len(cmd.SubCommands) > 0 {
			buf.WriteString("**サブコマンド:**\n\n")
			for _, sub := range cmd.SubCommands {
				buf.WriteString(fmt.Sprintf("- `%s` - %s\n", sub.Name, sub.Description))
			}
			buf.WriteString("\n")
		}
	}

	// ファイルに追記
	newContent := string(content) + buf.String()
	if err := os.WriteFile(inputPath, []byte(newContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added %d missing command(s) to %s:\n", len(missingCommands), inputPath)
	for _, cmd := range missingCommands {
		fmt.Printf("  - %s\n", cmd.Name)
	}
}

// findExistingCommands は docs/commands.md から既存のコマンドセクションを検出
func findExistingCommands(content string) map[string]bool {
	existing := make(map[string]bool)

	// ### `/command` または ### `/command`, `/alias` 形式を検出
	// バッククォート付き: ### `/help`
	// バッククォートなし: ### /help
	re := regexp.MustCompile(`###\s+[\x60]?(/[a-z_]+)`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			existing[matches[1]] = true
		}
	}

	return existing
}
