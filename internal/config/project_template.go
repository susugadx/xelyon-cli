package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrProjectConfigExists は xelyon.yaml が既に存在する場合のエラー。
var ErrProjectConfigExists = errors.New("xelyon.yaml already exists")

const projectYAMLTemplate = `# %s - Project Configuration
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
# 明示完了時の final checks（省略時は config.yaml の final_checks を使用）
# final_checks:
#   commands:
#     - "go fmt ./... && go build ./... && go test ./..."
#   timeout: 600
`

// CreateProjectConfigTemplate は xelyon.yaml のテンプレートを作成する。
func CreateProjectConfigTemplate(path string, overwrite bool) error {
	if path == "" {
		path = "xelyon.yaml"
	}
	if _, err := os.Stat(path); err == nil && !overwrite {
		return ErrProjectConfigExists
	}

	projectName := projectNameForTemplate(path)
	content := fmt.Sprintf(projectYAMLTemplate, projectName, projectName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write xelyon.yaml: %w", err)
	}
	return nil
}

func projectNameForTemplate(path string) string {
	if path == "" || path == "xelyon.yaml" {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Base(cwd)
		}
		return "project"
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Base(cwd)
		}
		return "project"
	}
	return filepath.Base(dir)
}
