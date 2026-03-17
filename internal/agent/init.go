package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// xelyonYAMLTemplate は xelyon.yaml のテンプレート
const xelyonYAMLTemplate = `# %s - Project Configuration
# AI 用コンテキスト。ドキュメントではありません。
# AI が許可なくこのファイルを肥大化させることを禁止します。

# プロジェクトの概要・背景情報（AI に注入されるコンテキスト）
context: |
  # %s
  ここにプロジェクトの概要を記述してください。

# 必須ルール（AI は必ずこれに従う）
rules:
  - "コード変更後は必ず go fmt ./... && go build ./... を実行すること"
  - "テストが通ることを確認してからコミットすること"

# 条件付きルール/コンテキスト（対象パスが会話に出た時だけ注入）
# conditional:
#   - name: Go backend
#     paths:
#       - "cmd/**/*.go"
#       - "internal/**/*.go"
#     rules:
#       - "公開関数・型には日本語コメントを付けること"
#     context: |
#       context.Context を先頭引数に取り、table-driven test を優先します。
#
# Project Map / list_dir / search_code で共有する ignore パターン
# ignore:
#   patterns:
#     - "dist"
#     - "*.min.js"
#
# 完了時フック（省略時は config.yaml の hooks を使用）
# hooks:
#   on_completion:
#     - "go fmt ./... && go build ./... && go test ./..."
#   timeout: 60
#   max_retry: 3
`

// handleInitCommand は/initコマンドを処理（xelyon.yaml 生成）
func handleInitCommand(agent *Agent) bool {
	runtimeUI := agent.ui()
	promptIO := runtimeUI.PromptIO()
	out := runtimeUI.Output()

	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Fprintln(out, "📝 xelyon.yaml Template Generator")
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)

	// xelyon.yaml が既に存在するか確認
	if _, err := os.Stat("xelyon.yaml"); err == nil {
		yellow.Fprintln(out, "⚠️  xelyon.yaml already exists")
		_, _ = fmt.Fprint(out, "Overwrite? (y/n): ")
		input, err := promptIO.ReadSimpleLine()
		if err != nil {
			red.Fprintf(out, "Failed to read input: %v\n", err)
			return true
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			yellow.Fprintln(out, "Cancelled")
			return true
		}
	}

	// プロジェクト名を取得（ディレクトリ名）
	cwd, err := os.Getwd()
	if err != nil {
		red.Fprintf(out, "Failed to get current directory: %v\n", err)
		return true
	}
	projectName := filepath.Base(cwd)

	// xelyon.yaml テンプレートを生成
	content := fmt.Sprintf(xelyonYAMLTemplate, projectName, projectName)

	if err := os.WriteFile("xelyon.yaml", []byte(content), 0644); err != nil {
		red.Fprintf(out, "Failed to write xelyon.yaml: %v\n", err)
		return true
	}

	green.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Fprintln(out, "✅ xelyon.yaml template created!")
	green.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)
	yellow.Fprintln(out, "Next steps:")
	yellow.Fprintln(out, "  1. Edit xelyon.yaml to add your project context and rules")
	yellow.Fprintln(out, "  2. Optionally add conditional rules or shared ignore patterns")
	yellow.Fprintln(out, "  3. Optionally configure hooks.on_completion for verification")
	yellow.Fprintln(out, "  4. xelyon.yaml will be automatically loaded on next session")

	return true
}

// fileExists はファイルの存在を確認（他のコマンドでも使用）
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
