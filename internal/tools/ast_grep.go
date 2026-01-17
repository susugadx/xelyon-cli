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
  Go:     'if err != nil { $$$ }'          - error handling block
  Go:     'func $NAME($ARGS) error'        - function returning error
  JS/TS:  'async function $NAME($ARGS)'    - async function
  JS/TS:  'console.log($ARG)'              - console.log calls
  JS/TS:  'useState($INIT)'                - React useState hook
  JS/TS:  'try { $$$ } catch ($E) { $$$ }' - try-catch block
  Python: 'def $NAME($ARGS):'              - function definition
  Python: 'async def $NAME($ARGS):'        - async function
  Rust:   'fn $NAME($ARGS) -> Result<$T, $E>' - function returning Result

Metavariables:
  $NAME  - matches a single AST node (identifier, expression, etc.)
  $$$    - matches zero or more AST nodes (for body, statements)
  $_     - matches any single node (unnamed/don't care)

Language codes: go, js, ts, tsx, py, rs, java, rb, c, cpp, kt, swift, cs, scala, php

More info: https://ast-grep.github.io/`
}

// PatternSuggestion はパターン提案の構造体
type PatternSuggestion struct {
	Pattern string
	Lang    string
	Hint    string
}

// GetPatternSuggestion は自然言語クエリから推奨パターンを返す（LLM支援用）
// この関数はLLMが自然言語をast-grepパターンに変換する際の参考として使用できる
func GetPatternSuggestion(query string) PatternSuggestion {
	query = strings.ToLower(query)

	// Go patterns
	if strings.Contains(query, "go") || strings.Contains(query, "golang") {
		if strings.Contains(query, "error") && strings.Contains(query, "handling") {
			return PatternSuggestion{"if err != nil { $$$ }", "go", "Matches error handling blocks"}
		}
		if strings.Contains(query, "function") || strings.Contains(query, "func") {
			if strings.Contains(query, "error") {
				return PatternSuggestion{"func $NAME($ARGS) error", "go", "Matches functions returning error"}
			}
			return PatternSuggestion{"func $NAME($ARGS)", "go", "Matches function definitions"}
		}
		if strings.Contains(query, "method") {
			return PatternSuggestion{"func ($R $TYPE) $NAME($ARGS)", "go", "Matches method definitions"}
		}
		if strings.Contains(query, "struct") {
			return PatternSuggestion{"type $NAME struct { $$$ }", "go", "Matches struct definitions"}
		}
		if strings.Contains(query, "interface") {
			return PatternSuggestion{"type $NAME interface { $$$ }", "go", "Matches interface definitions"}
		}
	}

	// JavaScript/TypeScript patterns
	if strings.Contains(query, "javascript") || strings.Contains(query, "js") ||
		strings.Contains(query, "typescript") || strings.Contains(query, "ts") ||
		strings.Contains(query, "react") {
		if strings.Contains(query, "async") {
			return PatternSuggestion{"async function $NAME($ARGS)", "js", "Matches async functions"}
		}
		if strings.Contains(query, "usestate") || strings.Contains(query, "hook") {
			return PatternSuggestion{"useState($INIT)", "tsx", "Matches useState hooks"}
		}
		if strings.Contains(query, "console") {
			return PatternSuggestion{"console.log($ARG)", "js", "Matches console.log calls"}
		}
		if strings.Contains(query, "try") || strings.Contains(query, "catch") {
			return PatternSuggestion{"try { $$$ } catch ($E) { $$$ }", "js", "Matches try-catch blocks"}
		}
		if strings.Contains(query, "class") {
			return PatternSuggestion{"class $NAME { $$$ }", "js", "Matches class definitions"}
		}
		if strings.Contains(query, "function") {
			return PatternSuggestion{"function $NAME($ARGS)", "js", "Matches function definitions"}
		}
	}

	// Python patterns
	if strings.Contains(query, "python") || strings.Contains(query, "py") {
		if strings.Contains(query, "async") {
			return PatternSuggestion{"async def $NAME($ARGS):", "py", "Matches async functions"}
		}
		if strings.Contains(query, "function") || strings.Contains(query, "def") {
			return PatternSuggestion{"def $NAME($ARGS):", "py", "Matches function definitions"}
		}
		if strings.Contains(query, "class") {
			return PatternSuggestion{"class $NAME:", "py", "Matches class definitions"}
		}
	}

	// Rust patterns
	if strings.Contains(query, "rust") || strings.Contains(query, "rs") {
		if strings.Contains(query, "result") {
			return PatternSuggestion{"fn $NAME($ARGS) -> Result<$T, $E>", "rs", "Matches functions returning Result"}
		}
		if strings.Contains(query, "function") || strings.Contains(query, "fn") {
			return PatternSuggestion{"fn $NAME($ARGS)", "rs", "Matches function definitions"}
		}
		if strings.Contains(query, "struct") {
			return PatternSuggestion{"struct $NAME { $$$ }", "rs", "Matches struct definitions"}
		}
		if strings.Contains(query, "impl") {
			return PatternSuggestion{"impl $TYPE { $$$ }", "rs", "Matches impl blocks"}
		}
	}

	// Default: no specific pattern
	return PatternSuggestion{"", "", "No specific pattern found. Try: 'func $NAME($ARGS)' for Go, 'function $NAME($ARGS)' for JS"}
}
