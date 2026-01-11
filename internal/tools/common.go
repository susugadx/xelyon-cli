package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	// JSONブロックを探す（複数パターン対応）
	patterns := []string{
		"{\"tool\"",   // {"tool"
		"{ \"tool\"",  // { "tool"
		"{\"tool\":",  // {"tool":
		"{ \"tool\":", // { "tool":
	}

	start := -1
	for _, pattern := range patterns {
		idx := strings.Index(response, pattern)
		if idx != -1 && (start == -1 || idx < start) {
			start = idx
		}
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

	// RawArgs（any型）をArgs（string型）に変換
	toolCall.NormalizeArgs()

	return &toolCall
}

// getCurrentTime は現在時刻を返す（builtin.goから使用）
func getCurrentTime() time.Time {
	return time.Now()
}

// createBackup はファイルのタイムスタンプ付きバックアップを作成
func createBackup(filePath string) (string, error) {
	// ファイルが存在しない場合はスキップ（新規作成）
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", nil
	}

	// .gitignore自動追加（初回のみ）
	if err := ensureGitignore(filepath.Dir(filePath)); err != nil {
		yellow.Printf("Warning: Failed to update .gitignore: %v\n", err)
		// .gitignore追加失敗してもバックアップは続行
	}

	// タイムスタンプ付きバックアップパス生成
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.bak.%s", filePath, timestamp)

	// バックアップ作成
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// 古いバックアップを削除（maxGenerationsを超えたもの）
	if err := cleanupOldBackups(filePath); err != nil {
		yellow.Printf("Warning: Failed to cleanup old backups: %v\n", err)
		// 削除失敗してもバックアップは成功とみなす
	}

	return backupPath, nil
}

// cleanupOldBackups は古いバックアップファイルを削除
func cleanupOldBackups(filePath string) error {
	// 設定から最大世代数を取得
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	maxGenerations := cfg.Backup.MaxGenerations
	if maxGenerations <= 0 {
		maxGenerations = 5 // デフォルト
	}

	// 同じファイルのバックアップを検索
	dir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)
	pattern := fmt.Sprintf("%s.bak.*", baseName)

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}

	// バックアップが最大世代数以下なら削除不要
	if len(matches) <= maxGenerations {
		return nil
	}

	// タイムスタンプでソート（古い順）
	sort.Strings(matches)

	// 古いものから削除
	deleteCount := len(matches) - maxGenerations
	for i := 0; i < deleteCount; i++ {
		if err := os.Remove(matches[i]); err != nil {
			yellow.Printf("Warning: Failed to delete old backup %s: %v\n", matches[i], err)
			// 個別のファイル削除失敗は続行
		}
	}

	return nil
}

// globalAutoApprove は --auto-approve フラグの状態を保持
// Agent から SetAutoApprove() で設定される
var globalAutoApprove = false

// SetAutoApprove は --auto-approve フラグを設定
func SetAutoApprove(enabled bool) {
	globalAutoApprove = enabled
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

// confirmWithAutoApprove は危険度を考慮した確認プロンプト
// toolName: 実行するツール名
// message: 確認メッセージ
func confirmWithAutoApprove(toolName, message string) bool {
	// --auto-approve が有効 かつ ツールが自動承認可能な場合
	if IsAutoApprovable(toolName, globalAutoApprove) {
		safety := GetToolSafety(toolName)
		green.Printf("✓ Auto-approved (%s): %s\n", GetSafetyDescription(safety), toolName)
		return true
	}

	// それ以外は通常の確認プロンプト
	return confirm(message)
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
		endPos += len(contentLines[i])
		if i < endLine {
			endPos += 1 // +1 for \n between lines
		}
	}

	return true, startPos, endPos - 1 // -1 because endIdx is inclusive
}

// showImprovedDiff は改善された差分表示
func showImprovedDiff(oldStr, newStr string) {
	oldLines := strings.Split(oldStr, "\n")
	newLines := strings.Split(newStr, "\n")

	cfg := config.GetGlobalConfig()
	maxLines := cfg.Diff.ContextLines // 最大表示行数（0なら全行表示）

	cyan.Println("\nBefore / 変更前:")
	cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
	for i, line := range oldLines {
		if maxLines > 0 && i >= maxLines {
			yellow.Printf("│ ... (%d lines omitted / 行省略)\n", len(oldLines)-maxLines)
			break
		}
		red.Printf("│ - %s\n", line)
	}
	cyan.Println("└" + strings.Repeat("─", 60) + "┘")

	cyan.Println("\nAfter / 変更後:")
	cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
	for i, line := range newLines {
		if maxLines > 0 && i >= maxLines {
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

	cfg := config.GetGlobalConfig()
	contextLines := cfg.Diff.ContextLines // 0なら全行表示

	diffCount := 0
	for i := 0; i < maxLines && (contextLines == 0 || diffCount < contextLines); i++ {
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
	} else if contextLines > 0 && diffCount >= contextLines {
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
