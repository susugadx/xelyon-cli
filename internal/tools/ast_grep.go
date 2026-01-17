package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// executeAstGrep はast-grepで構造的コード検索を実行
// ast-grepはTree-sitterベースの構造的コード検索ツール
// パターン例:
//   - 'func $NAME($ARGS)' - Go関数にマッチ
//   - 'async function $NAME' - JS async関数にマッチ
//   - 'def $NAME($ARGS):' - Python関数にマッチ
func executeAstGrep(pattern, lang, path string) string {
	if pattern == "" {
		return "Error: pattern is required\n\n" + getAstGrepHelp()
	}

	// ast-grepバイナリ確認
	if _, err := exec.LookPath("ast-grep"); err != nil {
		return "Error: ast-grep is not installed.\n\nInstall with one of:\n  brew install ast-grep\n  cargo install ast-grep --locked\n  npm i @ast-grep/cli -g\n  pip install ast-grep-cli"
	}

	green.Printf("🔍 ast-grep: searching for pattern '%s'\n", pattern)

	// コマンド構築
	// ast-grep run --pattern '<pattern>' [--lang <lang>] [path]
	args := []string{"run", "--pattern", pattern}
	if lang != "" {
		args = append(args, "--lang", lang)
	}
	if path != "" {
		args = append(args, path)
	} else {
		args = append(args, ".")
	}

	cmd := exec.Command("ast-grep", args...)
	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		// ast-grepはマッチがない場合も終了コード1を返すことがある
		if result == "" {
			return fmt.Sprintf("No matches found for pattern '%s'", pattern)
		}
		// エラーメッセージがある場合はそのまま返す
		if strings.Contains(result, "error") || strings.Contains(result, "Error") {
			return fmt.Sprintf("ast-grep error: %s", result)
		}
	}

	// 結果が空の場合
	if strings.TrimSpace(result) == "" {
		return fmt.Sprintf("No matches found for pattern '%s'", pattern)
	}

	// 結果が長すぎる場合は切り詰め
	lines := strings.Split(result, "\n")
	if len(lines) > 100 {
		result = strings.Join(lines[:100], "\n") +
			fmt.Sprintf("\n... (%d more lines)", len(lines)-100)
	}

	return result
}

// getAstGrepHelp はast-grepのパターン例を返す
func getAstGrepHelp() string {
	return `ast-grep pattern examples:
  Go:     'func $NAME($ARGS)'              - function definition
  Go:     'if err != nil { return $ERR }'  - error handling
  JS/TS:  'async function $NAME($ARGS)'    - async function
  JS/TS:  'console.log($ARG)'              - console.log calls
  Python: 'def $NAME($ARGS):'              - function definition
  Rust:   'fn $NAME($ARGS) -> $RET'        - function with return type

Metavariables:
  $NAME  - matches a single AST node (identifier, expression, etc.)
  $$$    - matches zero or more AST nodes
  $_     - matches any single node (unnamed)

More info: https://ast-grep.github.io/`
}
