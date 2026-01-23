package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// executeSearchCode はコード内を検索（grep）
func executeSearchCode(pattern string, path string) string {
	if pattern == "" {
		return "Error: pattern is required"
	}

	// キャッシュチェック
	if GlobalToolCache != nil {
		if cached, ok := GlobalToolCache.GetSearch(pattern, path); ok {
			return cached
		}
	}

	green.Printf("🔍 Searching for '%s' in %s\n", pattern, path)

	// grepで検索（-r: 再帰, -n: 行番号, -I: バイナリ除外）
	cmd := exec.Command("grep", "-rn", "-I", "--include=*.go", "--include=*.js", "--include=*.ts", "--include=*.py", "--include=*.md", "--include=*.json", "--include=*.yaml", "--include=*.yml", pattern, path)
	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		// grepは見つからない時もエラーを返す
		if result == "" {
			noMatchResult := fmt.Sprintf("No matches found for '%s'", pattern)
			if GlobalToolCache != nil {
				GlobalToolCache.SetSearch(pattern, path, noMatchResult)
			}
			return noMatchResult
		}
	}

	// 結果が長すぎる場合は切り詰め
	lines := strings.Split(result, "\n")
	if len(lines) > 50 {
		result = strings.Join(lines[:50], "\n") + fmt.Sprintf("\n... (%d more matches)", len(lines)-50)
	}

	if GlobalToolCache != nil {
		GlobalToolCache.SetSearch(pattern, path, result)
	}
	return result
}
