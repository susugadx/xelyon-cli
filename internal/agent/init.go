package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// xelyonMDTemplate はXELYON.mdのテンプレート
const xelyonMDTemplate = `# %s

> AI 用コンテキスト。ドキュメントではありません。
> AI が許可なくこのファイルを肥大化させることを禁止します。

## 概要


## 開発ルール

`

// handleInitCommand は/initコマンドを処理（XELYON.md生成）
func handleInitCommand(agent *Agent) bool {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("📝 XELYON.md Template Generator")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// XELYON.mdが既に存在するか確認
	if _, err := os.Stat("XELYON.md"); err == nil {
		yellow.Println("⚠️  XELYON.md already exists")
		fmt.Print("Overwrite? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			red.Printf("Failed to read input: %v\n", err)
			return true
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			yellow.Println("Cancelled")
			return true
		}
	}

	// プロジェクト名を取得（ディレクトリ名）
	cwd, err := os.Getwd()
	if err != nil {
		red.Printf("Failed to get current directory: %v\n", err)
		return true
	}
	projectName := filepath.Base(cwd)

	// シンプルなテンプレートを生成
	content := generateXELYONMDTemplate(projectName)

	if err := os.WriteFile("XELYON.md", []byte(content), 0644); err != nil {
		red.Printf("Failed to write XELYON.md: %v\n", err)
		return true
	}

	green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Println("✅ XELYON.md template created!")
	green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	yellow.Println("Next steps:")
	yellow.Println("  1. Edit XELYON.md to add your project overview and rules")
	yellow.Println("  2. Keep it minimal - RepoMap handles code structure")
	yellow.Println("  3. XELYON.md will be automatically included in AI context")

	return true
}

// generateXELYONMDTemplate はシンプルなテンプレートを生成
func generateXELYONMDTemplate(projectName string) string {
	return fmt.Sprintf(xelyonMDTemplate, projectName)
}

// fileExists はファイルの存在を確認（他のコマンドでも使用）
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
