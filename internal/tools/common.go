package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/config"
)

var (
	yellow = color.New(color.FgYellow)
	green  = color.New(color.FgGreen)
	red    = color.New(color.FgRed)
	cyan   = color.New(color.FgCyan)
)

// ParseToolCall はレスポンスからツール呼び出しを抽出
func ParseToolCall(response string) *ToolCall {
	// JSONブロックを探す
	start := strings.Index(response, "{\"tool\"")
	if start == -1 {
		start = strings.Index(response, "{ \"tool\"")
	}
	if start == -1 {
		return nil
	}

	// 対応する閉じ括弧を探す
	depth := 0
	end := -1
	for i := start; i < len(response); i++ {
		if response[i] == '{' {
			depth++
		} else if response[i] == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	if end == -1 {
		return nil
	}

	jsonStr := response[start:end]
	var toolCall ToolCall
	if err := json.Unmarshal([]byte(jsonStr), &toolCall); err != nil {
		return nil
	}

	return &toolCall
}

// getCurrentTime は現在時刻を返す（builtin.goから使用）
func getCurrentTime() time.Time {
	return time.Now()
}

// createBackup はファイルの.bakバックアップを作成
func createBackup(filePath string) (string, error) {
	// ファイルが存在しない場合はスキップ（新規作成）
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", nil
	}

	// バックアップパス生成
	backupPath := filePath + ".bak"

	// 既存の.bakを上書き（常に最新の1つだけ保持）
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// confirm はユーザーに確認を求める（テスト用にグローバル変数として定義）
var confirm = func(message string) bool {
	yellow.Printf("%s (y/n): ", message)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes" || response == "ｙ" || response == "はい"
}

// truncate は文字列を指定長で切り詰め
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// normalizeLeadingWhitespace は行頭の空白のみを正規化
// - タブをスペース4つに変換
// - 行頭の空白を削除
// - 行内の空白は保持（安全性重視）
func normalizeLeadingWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		// タブをスペース4つに変換
		line = strings.ReplaceAll(line, "\t", "    ")
		// 行頭の空白のみをトリム（行内は保持）
		trimmed := strings.TrimLeft(line, " ")
		normalized = append(normalized, trimmed)
	}
	return strings.Join(normalized, "\n")
}

// findWithNormalizedWhitespace は正規化した状態で文字列を検索
func findWithNormalizedWhitespace(content, pattern string) (found bool, startIdx, endIdx int) {
	normalizedContent := normalizeLeadingWhitespace(content)
	normalizedPattern := normalizeLeadingWhitespace(pattern)

	idx := strings.Index(normalizedContent, normalizedPattern)
	if idx == -1 {
		return false, -1, -1
	}

	// 正規化前の位置を計算（簡易実装：行番号ベース）
	contentLines := strings.Split(content, "\n")
	normalizedLines := strings.Split(normalizedContent, "\n")
	patternLines := strings.Split(normalizedPattern, "\n")

	// 正規化後の行番号を特定
	var currentPos int
	var lineNum int
	for i, line := range normalizedLines {
		if currentPos <= idx && idx < currentPos+len(line)+1 {
			lineNum = i
			break
		}
		currentPos += len(line) + 1 // +1 for \n
	}

	// 元のコンテンツから該当部分を抽出
	startLine := lineNum
	endLine := lineNum + len(patternLines) - 1

	if endLine >= len(contentLines) {
		return false, -1, -1
	}

	// 行単位で元の文字列を再構築
	var startPos int
	for i := 0; i < startLine; i++ {
		startPos += len(contentLines[i]) + 1
	}

	var endPos = startPos
	for i := startLine; i <= endLine; i++ {
		endPos += len(contentLines[i]) + 1
	}

	return true, startPos, endPos - 1 // -1 to exclude final \n
}

// showImprovedDiff は改善された差分表示
func showImprovedDiff(oldStr, newStr string) {
	oldLines := strings.Split(oldStr, "\n")
	newLines := strings.Split(newStr, "\n")

	maxLines := config.MaxDiffDisplayLines // 最大表示行数

	cyan.Println("\nBefore / 変更前:")
	cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
	for i, line := range oldLines {
		if i >= maxLines {
			yellow.Printf("│ ... (%d lines omitted / 行省略)\n", len(oldLines)-maxLines)
			break
		}
		red.Printf("│ - %s\n", line)
	}
	cyan.Println("└" + strings.Repeat("─", 60) + "┘")

	cyan.Println("\nAfter / 変更後:")
	cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
	for i, line := range newLines {
		if i >= maxLines {
			yellow.Printf("│ ... (%d lines omitted / 行省略)\n", len(newLines)-maxLines)
			break
		}
		green.Printf("│ + %s\n", line)
	}
	cyan.Println("└" + strings.Repeat("─", 60) + "┘\n")
}

// showDiff は差分を表示
func showDiff(old, new, filename string) {
	yellow.Printf("📝 Changes to: %s\n", filename)
	fmt.Println(strings.Repeat("─", 50))

	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	// 簡易diff（行数が違う部分を表示）
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	diffCount := 0
	for i := 0; i < maxLines && diffCount < config.MaxDiffIterations; i++ {
		oldLine := ""
		newLine := ""
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			diffCount++
			if oldLine != "" {
				red.Printf("- %s\n", oldLine)
			}
			if newLine != "" {
				green.Printf("+ %s\n", newLine)
			}
		}
	}

	if diffCount == 0 {
		fmt.Println("(no changes)")
	} else if diffCount >= config.MaxDiffIterations {
		yellow.Println("... (more changes)")
	}

	fmt.Println(strings.Repeat("─", 50))
}

// showPreview は新規ファイルのプレビューを表示
func showPreview(content string) {
	fmt.Println(strings.Repeat("─", 50))
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if i >= 20 {
			yellow.Printf("... (%d more lines)\n", len(lines)-20)
			break
		}
		fmt.Println(line)
	}
	fmt.Println(strings.Repeat("─", 50))
}

// min は2つの整数の小さい方を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max は2つの整数の大きい方を返す
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// detectTestFramework detects available test framework
func detectTestFramework(path string) (framework string, command string) {
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return "Go", "go test ./..."
	}

	if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
		if _, err := os.Stat(filepath.Join(path, "yarn.lock")); err == nil {
			return "JavaScript (yarn)", "yarn test"
		}
		return "JavaScript (npm)", "npm test"
	}

	if _, err := os.Stat(filepath.Join(path, "pytest.ini")); err == nil {
		return "Python (pytest)", "pytest"
	}
	if _, err := os.Stat(filepath.Join(path, "setup.py")); err == nil {
		return "Python (pytest)", "pytest"
	}

	if _, err := os.Stat(filepath.Join(path, "Cargo.toml")); err == nil {
		return "Rust", "cargo test"
	}

	return "", ""
}

// detectFormatter detects available formatter
func detectFormatter(path string) (formatter string, command string) {
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return "gofmt", "go fmt ./..."
	}

	prettierConfigs := []string{".prettierrc", ".prettierrc.json", ".prettierrc.js", "prettier.config.js"}
	for _, config := range prettierConfigs {
		if _, err := os.Stat(filepath.Join(path, config)); err == nil {
			return "prettier", "prettier --write ."
		}
	}

	if _, err := os.Stat(filepath.Join(path, "pyproject.toml")); err == nil {
		if exec.Command("which", "black").Run() == nil {
			return "black", "black ."
		}
	}

	if _, err := os.Stat(filepath.Join(path, "Cargo.toml")); err == nil {
		return "rustfmt", "cargo fmt"
	}

	return "", ""
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// hasGlobMatches checks if any files match the glob pattern
func hasGlobMatches(pattern string) bool {
	matches, _ := filepath.Glob(pattern)
	return len(matches) > 0
}

// detectLinter detects available linter for the project
func detectLinter(basePath string) (linterName, checkCmd, fixCmd string) {
	// Go: go.mod存在チェック
	if fileExists(filepath.Join(basePath, "go.mod")) {
		if commandExists("golangci-lint") {
			return "golangci-lint", "golangci-lint run", "golangci-lint run --fix"
		}
		if commandExists("go") {
			return "go vet", "go vet ./...", "" // fixコマンドなし
		}
	}

	// JavaScript/TypeScript: package.json + ESLint
	if fileExists(filepath.Join(basePath, "package.json")) {
		eslintConfigFiles := []string{".eslintrc", ".eslintrc.js", ".eslintrc.json", "eslint.config.js"}
		for _, configFile := range eslintConfigFiles {
			if fileExists(filepath.Join(basePath, configFile)) {
				return "eslint", "eslint .", "eslint . --fix"
			}
		}
	}

	// Python: *.pyファイル存在チェック
	if hasGlobMatches(filepath.Join(basePath, "*.py")) {
		if commandExists("ruff") {
			return "ruff", "ruff check .", "ruff check . --fix"
		}
		if commandExists("pylint") {
			return "pylint", "pylint .", "" // fixコマンドなし
		}
	}

	// Rust: Cargo.toml存在チェック
	if fileExists(filepath.Join(basePath, "Cargo.toml")) {
		return "clippy", "cargo clippy", "cargo clippy --fix --allow-dirty"
	}

	return "", "", "" // リンター未検出
}
