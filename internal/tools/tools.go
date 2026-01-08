package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

var (
	yellow = color.New(color.FgYellow)
	green  = color.New(color.FgGreen)
	red    = color.New(color.FgRed)
	cyan   = color.New(color.FgCyan)
)

// ToolCall はAIからのツール呼び出し
type ToolCall struct {
	Tool string            `json:"tool"`
	Args map[string]string `json:"args"`
}

// FileChange はファイル変更履歴
type FileChange struct {
	FilePath    string
	BackupPath  string
	Timestamp   time.Time
	Tool        string
	Description string
}

// 自動実行可能なコマンド（安全なもの）
var safeCommands = map[string]bool{
	"ls": true, "cat": true, "pwd": true, "echo": true, "which": true,
	"head": true, "tail": true, "wc": true, "grep": true, "find": true,
	"git status": true, "git log": true, "git diff": true, "git branch": true,
	"git ls-files": true, "git show": true, "git remote": true,
	"go version": true, "go mod tidy": true,
	"node -v": true, "npm -v": true, "npm list": true,
	"python --version": true, "pip list": true,
}

// ブロックするコマンド（危険すぎ）
var blockedCommands = []string{
	"rm -rf /", "rm -rf ~", "rm -rf *",
	"sudo rm", "sudo chmod", "sudo chown",
	"chmod 777", "chmod -R 777",
	"mkfs", "dd if=", ":(){:|:&};:",
	"> /dev/sda", "mv / ",
	"sed -i", "sed -e", "sed '", // ファイル編集系sed
	"awk -i", "perl -i", "perl -p", // その他のインライン編集コマンド
}

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

// Execute はツールを実行（Registry経由）
func Execute(tc *ToolCall) (string, *FileChange) {
	// ツール名のみ表示（詳細は各ツールの確認画面で表示）
	cyan.Printf("🔧 Tool: %s\n", tc.Tool)

	// 簡潔な引数表示（JSON不使用）
	switch tc.Tool {
	case "read_file":
		fmt.Printf("   File: %s\n", tc.Args["path"])
	case "write_file":
		lines := strings.Split(tc.Args["content"], "\n")
		fmt.Printf("   File: %s (%d lines)\n", tc.Args["path"], len(lines))
	case "str_replace":
		fmt.Printf("   File: %s\n", tc.Args["path"])
	case "bash":
		fmt.Printf("   Command: %s\n", truncate(tc.Args["command"], 60))
	case "list_dir":
		path := tc.Args["path"]
		if path == "" {
			path = "."
		}
		fmt.Printf("   Directory: %s\n", path)
	case "git_add", "git_commit", "git_push", "git_status", "git_diff", "git_log",
		"git_branch", "git_checkout", "git_stash":
		// Git操作は引数を簡潔に表示
		for k, v := range tc.Args {
			if v != "" {
				fmt.Printf("   %s: %s\n", k, truncate(v, 60))
			}
		}
	case "search_code", "search_file":
		fmt.Printf("   Pattern: %s\n", tc.Args["pattern"])
		if tc.Args["path"] != "" {
			fmt.Printf("   Path: %s\n", tc.Args["path"])
		}
	case "insert_after", "insert_before":
		fmt.Printf("   File: %s\n", tc.Args["path"])
		fmt.Printf("   Pattern: %s\n", truncate(tc.Args["pattern"], 60))
		fmt.Printf("   Content: %s\n", truncate(tc.Args["content"], 60))
	case "copy_file":
		fmt.Printf("   Source: %s\n", tc.Args["src"])
		fmt.Printf("   Destination: %s\n", tc.Args["dest"])
	case "web_search":
		fmt.Printf("   Query: %s\n", tc.Args["query"])
	default:
		// その他のツール（MCPツール等）
		if len(tc.Args) > 0 {
			for k, v := range tc.Args {
				fmt.Printf("   %s: %s\n", k, truncate(v, 60))
			}
		}
	}
	fmt.Println()

	// デフォルト値の設定（Registry実行前）
	// list_dir, git_add, search_code, search_fileでpathが空の場合"."を設定
	if tc.Args["path"] == "" {
		switch tc.Tool {
		case "list_dir", "git_add", "search_code", "search_file":
			tc.Args["path"] = "."
		}
	}

	// Registry経由でツール実行
	result, change := DefaultRegistry.Execute(tc)

	// ページング表示
	pager := ui.NewPager()
	pager.Display(result)

	return result, change
}

// executeBash はシェルコマンドを実行
func executeBash(command string) string {
	if command == "" {
		return "Error: command is empty"
	}

	// ブロックチェック
	for _, blocked := range blockedCommands {
		if strings.Contains(command, blocked) {
			red.Printf("🚫 Blocked dangerous command: %s\n", command)
			return "Error: This command is blocked for safety"
		}
	}

	// 安全なコマンドか確認
	needConfirm := true
	for safe := range safeCommands {
		if strings.HasPrefix(command, safe) {
			needConfirm = false
			break
		}
	}

	// 確認が必要な場合
	if needConfirm {
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("⚙️  Shell Command / シェルコマンド実行\n")
		cyan.Printf("📜 Command / コマンド: %s\n", command)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: This command may modify your system / 警告: システムに変更が加わる可能性があります")

		if !confirm("Run this command? / 実行しますか？") {
			return "Cancelled by user"
		}
	}

	// 実行
	green.Printf("▶ Running: %s\n", command)
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir, _ = os.Getwd()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\nOutput: %s", err, string(output))
	}

	result := string(output)
	if len(result) > 5000 {
		result = result[:5000] + "\n... (truncated)"
	}

	return result
}

// executeReadFile はファイルを読み込む
func executeReadFile(path string) string {
	if path == "" {
		return "Error: path is empty"
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	result := string(content)
	green.Printf("📄 Read: %s (%d bytes)\n", path, len(result))

	// 長すぎる場合は切り詰め
	if len(result) > 10000 {
		result = result[:10000] + "\n... (truncated, showing first 10000 chars)"
	}

	return result
}

// executeWriteFile はファイルに書き込む
func executeWriteFile(path string, content string) (string, string, error) {
	if path == "" {
		return "Error: path is empty", "", nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	// ファイルが存在するか確認
	exists := false
	if _, err := os.Stat(absPath); err == nil {
		exists = true
	}

	// 確認UI
	lines := strings.Split(content, "\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if exists {
		cyan.Printf("📝 Create/Overwrite File / ファイルの上書き\n")
	} else {
		cyan.Printf("📝 Create File / ファイルの新規作成\n")
	}
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Printf("📏 Size / サイズ: %d lines / 行\n", len(lines))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// diff表示（既存ファイルの場合）
	if exists {
		oldContent, _ := os.ReadFile(absPath)
		showDiff(string(oldContent), content, path)
	} else {
		showPreview(content)
	}

	if !confirm("Create/overwrite this file? / このファイルを作成・上書きしますか？") {
		return "Cancelled by user", "", nil
	}

	// バックアップ作成（既存ファイルの場合のみ）
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Warning: failed to create backup: %v (continuing anyway)", err), "", nil
	}

	// ディレクトリ作成
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err), "", nil
	}

	// 書き込み
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), "", nil
	}

	green.Printf("✅ Written: %s\n", path)
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), backupPath, nil
}

// executeAppendFile はファイル末尾にコンテンツを追加
func executeAppendFile(path, content string) (string, string, error) {
	// 引数検証
	if path == "" {
		return "Error: path is empty", "", nil
	}
	if content == "" {
		return "Error: content is empty", "", nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	// ファイルが存在するかチェック
	exists := false
	if _, err := os.Stat(absPath); err == nil {
		exists = true
	}

	// バックアップ作成（既存ファイルのみ）
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Error creating backup: %v", err), "", nil
	}

	// プレビュー表示（確認なし、非破壊的操作）
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if exists {
		cyan.Printf("➕ Append to File / ファイルに追記\n")
	} else {
		cyan.Printf("➕ Create File with Content / ファイルを作成\n")
	}
	cyan.Printf("📂 Path / パス: %s\n", path)
	contentLines := strings.Split(content, "\n")
	cyan.Printf("📏 Adding / 追加: %d lines / 行\n", len(contentLines))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 既存ファイルの最後の部分を表示
	if exists {
		oldContent, _ := os.ReadFile(absPath)
		if len(oldContent) > 0 {
			oldLines := strings.Split(string(oldContent), "\n")
			yellow.Println("\nExisting file (last 10 lines) / 既存ファイル（最終10行）:")
			cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
			startLine := len(oldLines) - 10
			if startLine < 0 {
				startLine = 0
			}
			for i := startLine; i < len(oldLines) && i < startLine+10; i++ {
				fmt.Printf("│ %s\n", oldLines[i])
			}
			cyan.Println("└" + strings.Repeat("─", 60) + "┘")
		}
	}

	// 追加するコンテンツをプレビュー
	yellow.Println("\nContent to append / 追記する内容:")
	cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
	for i, line := range contentLines {
		if i >= 10 {
			yellow.Printf("│ ... (%d more lines / 行省略)\n", len(contentLines)-10)
			break
		}
		green.Printf("│ + %s\n", line)
	}
	cyan.Println("└" + strings.Repeat("─", 60) + "┘")
	fmt.Println()

	// ファイルを開く（追記モード）
	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Sprintf("Error opening file: %v", err), "", nil
	}
	defer file.Close()

	// コンテンツを追記
	if _, err := file.WriteString(content); err != nil {
		return fmt.Sprintf("Error appending to file: %v", err), "", nil
	}

	green.Printf("✅ Appended: %s\n", path)
	return fmt.Sprintf("Successfully appended %d bytes to %s", len(content), path), backupPath, nil
}

// executePrependFile はファイル先頭にコンテンツを追加
func executePrependFile(path, content string) (string, string, error) {
	if path == "" {
		return "Error: path is empty", "", nil
	}
	if content == "" {
		return "Error: content is empty", "", nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	exists := false
	var oldContent []byte
	if _, err := os.Stat(absPath); err == nil {
		exists = true
		oldContent, _ = os.ReadFile(absPath)
	}

	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Error creating backup: %v", err), "", nil
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if exists {
		cyan.Printf("⬆️  Prepend to File / ファイル先頭に追記\n")
	} else {
		cyan.Printf("⬆️  Create File with Content / ファイルを作成\n")
	}
	cyan.Printf("📂 Path / パス: %s\n", path)
	contentLines := strings.Split(content, "\n")
	cyan.Printf("📏 Adding / 追加: %d lines / 行\n", len(contentLines))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	yellow.Println("\nContent to prepend / 先頭に追加する内容:")
	cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
	for i, line := range contentLines {
		if i >= 10 {
			yellow.Printf("│ ... (%d more lines / 行省略)\n", len(contentLines)-10)
			break
		}
		green.Printf("│ + %s\n", line)
	}
	cyan.Println("└" + strings.Repeat("─", 60) + "┘")

	if exists && len(oldContent) > 0 {
		oldLines := strings.Split(string(oldContent), "\n")
		yellow.Println("\nExisting file (first 10 lines) / 既存ファイル（最初10行）:")
		cyan.Println("┌" + strings.Repeat("─", 60) + "┐")
		for i := 0; i < len(oldLines) && i < 10; i++ {
			fmt.Printf("│ %s\n", oldLines[i])
		}
		cyan.Println("└" + strings.Repeat("─", 60) + "┘")
	}
	fmt.Println()

	newContent := content
	if !strings.HasSuffix(content, "\n") {
		newContent += "\n"
	}
	newContent += string(oldContent)

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), "", nil
	}

	green.Printf("✅ Prepended: %s\n", path)
	return fmt.Sprintf("Successfully prepended %d bytes to %s", len(content), path), backupPath, nil
}

// executeCreateDir はディレクトリを作成
func executeCreateDir(path string) string {
	if path == "" {
		return "Error: path is empty"
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if stat, err := os.Stat(absPath); err == nil {
		if stat.IsDir() {
			green.Printf("✅ Directory already exists: %s\n", path)
			return fmt.Sprintf("Directory already exists (idempotent): %s", path)
		}
		return fmt.Sprintf("Error: path exists but is not a directory: %s", path)
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err)
	}

	green.Printf("✅ Created directory: %s\n", path)
	return fmt.Sprintf("Successfully created directory: %s", path)
}

// executeRunTest はテストを自動検出して実行
func executeRunTest(path string) string {
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	framework, command := detectTestFramework(absPath)

	if framework == "" {
		return `No test framework detected.

Supported frameworks:
  - Go: go.mod → go test ./...
  - JavaScript/TypeScript: package.json → npm test
  - Python: pytest.ini/setup.py → pytest
  - Rust: Cargo.toml → cargo test

Please ensure your project has the appropriate configuration files.`
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🧪 Running Tests / テスト実行\n")
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Printf("🔧 Framework / フレームワーク: %s\n", framework)
	cyan.Printf("⚙️  Command / コマンド: %s\n", command)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	green.Printf("▶ Running: %s\n", command)

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = absPath
	output, err := cmd.CombinedOutput()

	result := string(output)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	if exitCode == 0 {
		green.Printf("\n✅ Tests passed (exit code: %d)\n", exitCode)
	} else {
		red.Printf("\n❌ Tests failed (exit code: %d)\n", exitCode)
	}

	return fmt.Sprintf("%s\n\nExit code: %d", result, exitCode)
}

// detectTestFramework はテストフレームワークを検出
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

// executeFormat はフォーマッターを自動検出して実行
func executeFormat(path string) (string, string, error) {
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	formatter, command := detectFormatter(absPath)

	if formatter == "" {
		return `No formatter detected.

Supported formatters:
  - Go: *.go files → go fmt ./...
  - JavaScript/TypeScript: .prettierrc → prettier --write .
  - Python: *.py files → black . or autopep8
  - Rust: Cargo.toml → cargo fmt

Please ensure your project has the appropriate files or configuration.`, "", nil
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("✨ Formatting Code / コード整形\n")
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Printf("🔧 Formatter / フォーマッター: %s\n", formatter)
	cyan.Printf("⚙️  Command / コマンド: %s\n", command)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	green.Printf("▶ Running: %s\n", command)

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = absPath
	output, err := cmd.CombinedOutput()

	result := string(output)
	if len(result) > 2000 {
		result = result[:2000] + "\n... (truncated)"
	}

	if err != nil {
		return fmt.Sprintf("Formatter failed:\n%s\n\nError: %v", result, err), "", nil
	}

	green.Println("\n✅ Formatting completed")

	return fmt.Sprintf("Successfully formatted code:\n%s", result), "", nil
}

// detectFormatter はフォーマッターを検出
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

// executeListDir はディレクトリ一覧を取得
func executeListDir(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return fmt.Sprintf("Error reading directory: %v", err)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("📂 %s", absPath))

	for _, entry := range entries {
		prefix := "  📄 "
		if entry.IsDir() {
			prefix = "  📁 "
		}
		info, _ := entry.Info()
		size := ""
		if info != nil && !entry.IsDir() {
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}
		lines = append(lines, prefix+entry.Name()+size)
	}

	result := strings.Join(lines, "\n")
	return result
}

// confirm はユーザーに確認を求める
func confirm(message string) bool {
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

	maxLines := 15 // 最大表示行数

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
	for i := 0; i < maxLines && diffCount < 20; i++ {
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
	} else if diffCount >= 20 {
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

// =====================
// Git系ツール
// =====================

// executeGitStatus は git status を実行
func executeGitStatus() string {
	green.Println("📊 git status")
	cmd := exec.Command("git", "status", "--short")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	if result == "" {
		result = "✨ Working tree clean"
	}
	return result
}

// executeGitDiff は git diff を実行
func executeGitDiff(path string) string {
	args := []string{"diff"}
	if path != "" {
		args = append(args, path)
	}
	green.Printf("📝 git diff %s\n", path)
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	if result == "" {
		result = "No changes"
	}
	if len(result) > 3000 {
		result = result[:3000] + "\n... (truncated)"
	}
	return result
}

// executeGitAdd は git add を実行
func executeGitAdd(path string) string {
	// 確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("➕ Git Stage / Gitステージング\n")
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !confirm("Stage this file? / このファイルをステージングしますか？") {
		return "Cancelled by user"
	}

	cmd := exec.Command("git", "add", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	return fmt.Sprintf("✅ Staged: %s", path)
}

// executeGitCommit は git commit を実行
func executeGitCommit(message string) string {
	if message == "" {
		return "Error: commit message is required"
	}

	// 確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("💾 Git Commit / Gitコミット\n")
	cyan.Printf("📝 Message / メッセージ:\n%s\n", message)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !confirm("Commit with this message? / この内容でコミットしますか？") {
		return "Cancelled by user"
	}

	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	fmt.Println(result)
	return result
}

// executeGitPush は git push を実行
func executeGitPush() string {
	// 確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🚀 Git Push / リモートへプッシュ\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	yellow.Println("⚠️  Warning: Changes will be published to remote / 警告: リモートリポジトリに変更が公開されます")

	if !confirm("Push to remote? / プッシュしますか？") {
		return "Cancelled by user"
	}

	cmd := exec.Command("git", "push")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	if result == "" {
		result = "✅ Pushed successfully"
	}
	fmt.Println(result)
	return result
}

// executeGitLog は git log を実行
func executeGitLog() string {
	green.Println("📜 git log")
	cmd := exec.Command("git", "log", "--oneline", "-10")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	return result
}

// executeGitBranch はブランチ操作（list/create/switch）
func executeGitBranch(action, branchName string) string {
	// デフォルトはlist
	if action == "" {
		action = "list"
	}

	// Validation
	if (action == "create" || action == "switch") && branchName == "" {
		return fmt.Sprintf("Error: branch_name is required for action '%s'", action)
	}

	// Action: list
	if action == "list" {
		green.Println("📋 git branch")
		cmd := exec.Command("git", "branch", "-a")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}
		result := string(output)
		if result == "" {
			result = "No branches found"
		}
		return result
	}

	// Action: create
	if action == "create" {
		green.Printf("🌿 Creating branch: %s\n", branchName)
		cmd := exec.Command("git", "branch", branchName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}
		return fmt.Sprintf("✅ Created branch: %s", branchName)
	}

	// Action: switch
	if action == "switch" {
		// 未コミット変更チェック
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusOutput, _ := statusCmd.CombinedOutput()
		hasChanges := len(strings.TrimSpace(string(statusOutput))) > 0

		if hasChanges {
			// 確認UI
			cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			cyan.Printf("🔀 Git Branch Switch / ブランチ切り替え\n")
			cyan.Printf("🌿 Target Branch / 対象ブランチ: %s\n", branchName)
			cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			yellow.Println("⚠️  Warning: You have uncommitted changes / 警告: コミットされていない変更があります")
			fmt.Println("\nUncommitted changes / 未コミットの変更:")
			fmt.Println(string(statusOutput))

			if !confirm("Switch branch anyway? / それでもブランチを切り替えますか？") {
				return "Cancelled by user"
			}
		} else {
			green.Printf("🔀 Switching to branch: %s\n", branchName)
		}

		cmd := exec.Command("git", "checkout", branchName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}
		return fmt.Sprintf("✅ Switched to branch: %s\n%s", branchName, string(output))
	}

	return fmt.Sprintf("Error: Unknown action '%s' (use 'list', 'create', or 'switch')", action)
}

// executeGitCheckout はファイル復元またはブランチチェックアウト
func executeGitCheckout(target string) (string, string, error) {
	// Validation
	if target == "" {
		return "Error: target is required (file path or branch name)", "", nil
	}

	// ターゲットがファイルかブランチか判定
	absTarget, _ := filepath.Abs(target)
	isFile := false
	if _, err := os.Stat(absTarget); err == nil {
		isFile = true
	} else if strings.Contains(target, "/") || strings.Contains(target, ".") {
		// ファイルっぽいパス（削除されたファイルの復元に対応）
		isFile = true
	}

	// ファイル復元（破壊的 - ALWAYS confirm）
	if isFile {
		// 現在の内容を読み込み
		oldContent := "[File does not exist or was deleted]"
		if content, err := os.ReadFile(absTarget); err == nil {
			oldContent = string(content)
		}

		// HEAD内容を取得
		headCmd := exec.Command("git", "show", "HEAD:"+target)
		headOutput, headErr := headCmd.CombinedOutput()
		headContent := ""
		if headErr == nil {
			headContent = string(headOutput)
		} else {
			headContent = "[Unable to read from HEAD - file may be new/untracked]"
		}

		// 確認UI
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("⚠️  Git Checkout File / ファイル復元\n")
		cyan.Printf("📂 File / ファイル: %s\n", target)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		red.Println("⚠️  DESTRUCTIVE: This will discard all local changes!")
		red.Println("⚠️  破壊的操作: このファイルのローカル変更はすべて失われます!")

		// Diff簡易表示
		if oldContent != "[File does not exist or was deleted]" && headContent != "[Unable to read from HEAD - file may be new/untracked]" {
			yellow.Println("\nChanges that will be lost / 失われる変更:")
			// 簡易diff: 最初10行と最後10行を表示
			oldLines := strings.Split(oldContent, "\n")
			headLines := strings.Split(headContent, "\n")
			if len(oldLines) > 20 {
				fmt.Println(strings.Join(oldLines[:10], "\n"))
				yellow.Printf("... (%d lines omitted) ...\n", len(oldLines)-20)
				fmt.Println(strings.Join(oldLines[len(oldLines)-10:], "\n"))
			} else {
				fmt.Println(oldContent)
			}
			yellow.Printf("\nWill be restored to (%d lines from HEAD)\n", len(headLines))
		} else {
			fmt.Printf("\nCurrent: %s\n", oldContent)
			fmt.Printf("HEAD:    %s\n", headContent)
		}

		if !confirm("Restore from HEAD? / HEADから復元しますか？") {
			return "Cancelled by user", "", nil
		}

		// バックアップ作成
		backupPath := ""
		if oldContent != "[File does not exist or was deleted]" {
			var err error
			backupPath, err = createBackup(absTarget)
			if err != nil {
				yellow.Printf("Warning: Failed to create backup: %v (continuing anyway)\n", err)
			} else {
				green.Printf("📦 Backup created: %s\n", backupPath)
			}
		}

		// Checkout実行
		cmd := exec.Command("git", "checkout", "HEAD", "--", target)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output)), "", nil
		}

		return fmt.Sprintf("✅ Restored from HEAD: %s", target), backupPath, nil
	}

	// ブランチチェックアウト
	green.Printf("🔀 Detected branch target: %s\n", target)
	yellow.Println("ℹ️  Tip: Use git_branch with action='switch' for more options")

	// 未コミット変更チェック
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusOutput, _ := statusCmd.CombinedOutput()
	hasChanges := len(strings.TrimSpace(string(statusOutput))) > 0

	if hasChanges {
		// 確認UI
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("🔀 Git Checkout Branch / ブランチチェックアウト\n")
		cyan.Printf("🌿 Target / 対象: %s\n", target)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: You have uncommitted changes / 警告: 未コミットの変更があります")
		fmt.Println("\nUncommitted changes / 未コミットの変更:")
		fmt.Println(string(statusOutput))

		if !confirm("Checkout anyway? / それでもチェックアウトしますか？") {
			return "Cancelled by user", "", nil
		}
	}

	cmd := exec.Command("git", "checkout", target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output)), "", nil
	}

	return fmt.Sprintf("✅ Checked out: %s\n%s", target, string(output)), "", nil
}

// executeGitStash はスタッシュ操作（save/list/pop/apply/drop）
func executeGitStash(action, message string) string {
	// デフォルトはsave
	if action == "" {
		action = "save"
	}

	// Validation
	validActions := map[string]bool{
		"save": true, "list": true, "pop": true, "apply": true, "drop": true,
	}
	if !validActions[action] {
		return fmt.Sprintf("Error: Unknown action '%s' (use 'save', 'list', 'pop', 'apply', or 'drop')", action)
	}

	// Action: save
	if action == "save" {
		// 変更があるかチェック
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusOutput, _ := statusCmd.CombinedOutput()
		if len(strings.TrimSpace(string(statusOutput))) == 0 {
			yellow.Println("⚠️  No changes to stash / スタッシュする変更がありません")
			return "No changes to stash (working tree clean)"
		}

		green.Println("📦 Stashing changes / 変更をスタッシュ")
		yellow.Println("\nChanges to stash / スタッシュする変更:")
		fmt.Println(string(statusOutput))

		args := []string{"stash", "push"}
		if message != "" {
			args = append(args, "-m", message)
		}

		cmd := exec.Command("git", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}

		return fmt.Sprintf("✅ Stashed changes\n%s", string(output))
	}

	// Action: list
	if action == "list" {
		green.Println("📋 git stash list")
		cmd := exec.Command("git", "stash", "list")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}
		result := string(output)
		if result == "" {
			result = "No stashes found"
		}
		return result
	}

	// Action: pop
	if action == "pop" {
		stashRef := "stash@{0}" // デフォルト: 最新
		if message != "" {
			stashRef = "stash@{" + message + "}"
		}

		// スタッシュプレビュー取得
		showCmd := exec.Command("git", "stash", "show", "-p", stashRef)
		showOutput, showErr := showCmd.CombinedOutput()

		// 確認UI
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("📦 Git Stash Pop / スタッシュ適用・削除\n")
		cyan.Printf("📋 Stash / スタッシュ: %s\n", stashRef)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: This may cause merge conflicts / 警告: マージ競合が発生する可能性があります")

		if showErr == nil {
			yellow.Println("\nStash preview / スタッシュプレビュー (first 20 lines):")
			lines := strings.Split(string(showOutput), "\n")
			maxLines := 20
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				fmt.Println(lines[i])
			}
			if len(lines) > 20 {
				yellow.Printf("... (%d more lines)\n", len(lines)-20)
			}
		}

		if !confirm("Pop this stash? / このスタッシュを適用・削除しますか？") {
			return "Cancelled by user"
		}

		cmd := exec.Command("git", "stash", "pop", stashRef)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error (may have conflicts): %v\n%s", err, string(output))
		}

		return fmt.Sprintf("✅ Popped stash: %s\n%s", stashRef, string(output))
	}

	// Action: apply
	if action == "apply" {
		stashRef := "stash@{0}"
		if message != "" {
			stashRef = "stash@{" + message + "}"
		}

		// スタッシュプレビュー取得
		showCmd := exec.Command("git", "stash", "show", "-p", stashRef)
		showOutput, showErr := showCmd.CombinedOutput()

		// 確認UI
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("📦 Git Stash Apply / スタッシュ適用\n")
		cyan.Printf("📋 Stash / スタッシュ: %s\n", stashRef)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: This may cause merge conflicts / 警告: マージ競合が発生する可能性があります")
		yellow.Println("ℹ️  Note: Stash will be kept after apply / 注意: スタッシュは適用後も保持されます")

		if showErr == nil {
			yellow.Println("\nStash preview / スタッシュプレビュー (first 20 lines):")
			lines := strings.Split(string(showOutput), "\n")
			maxLines := 20
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				fmt.Println(lines[i])
			}
			if len(lines) > 20 {
				yellow.Printf("... (%d more lines)\n", len(lines)-20)
			}
		}

		if !confirm("Apply this stash? / このスタッシュを適用しますか？") {
			return "Cancelled by user"
		}

		cmd := exec.Command("git", "stash", "apply", stashRef)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error (may have conflicts): %v\n%s", err, string(output))
		}

		return fmt.Sprintf("✅ Applied stash: %s\n%s", stashRef, string(output))
	}

	// Action: drop
	if action == "drop" {
		stashRef := "stash@{0}"
		if message != "" {
			stashRef = "stash@{" + message + "}"
		}

		// 確認UI
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("🗑️  Git Stash Drop / スタッシュ削除\n")
		cyan.Printf("📋 Stash / スタッシュ: %s\n", stashRef)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		red.Println("⚠️  DESTRUCTIVE: This stash will be permanently deleted!")
		red.Println("⚠️  破壊的操作: このスタッシュは完全に削除されます!")

		if !confirm("Delete this stash? / このスタッシュを削除しますか？") {
			return "Cancelled by user"
		}

		cmd := exec.Command("git", "stash", "drop", stashRef)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}

		return fmt.Sprintf("✅ Dropped stash: %s\n%s", stashRef, string(output))
	}

	return fmt.Sprintf("Error: Unknown action '%s'", action)
}

// =====================
// 検索系ツール
// =====================

// executeSearchCode はコード内を検索（grep）
func executeSearchCode(pattern string, path string) string {
	if pattern == "" {
		return "Error: pattern is required"
	}

	green.Printf("🔍 Searching for '%s' in %s\n", pattern, path)

	// grepで検索（-r: 再帰, -n: 行番号, -I: バイナリ除外）
	cmd := exec.Command("grep", "-rn", "-I", "--include=*.go", "--include=*.js", "--include=*.ts", "--include=*.py", "--include=*.md", "--include=*.json", "--include=*.yaml", "--include=*.yml", pattern, path)
	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		// grepは見つからない時もエラーを返す
		if result == "" {
			return fmt.Sprintf("No matches found for '%s'", pattern)
		}
	}

	// 結果が長すぎる場合は切り詰め
	lines := strings.Split(result, "\n")
	if len(lines) > 50 {
		result = strings.Join(lines[:50], "\n") + fmt.Sprintf("\n... (%d more matches)", len(lines)-50)
	}

	return result
}

// executeSearchFile はファイル名で検索（find）
func executeSearchFile(pattern string, path string) string {
	if pattern == "" {
		return "Error: pattern is required"
	}

	green.Printf("📁 Searching for files matching '%s' in %s\n", pattern, path)

	// findで検索（.gitは除外）
	cmd := exec.Command("find", path, "-type", "f", "-name", pattern, "-not", "-path", "*/.git/*")
	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, result)
	}

	if strings.TrimSpace(result) == "" {
		return fmt.Sprintf("No files found matching '%s'", pattern)
	}

	// 結果が長すぎる場合は切り詰め
	lines := strings.Split(result, "\n")
	if len(lines) > 30 {
		result = strings.Join(lines[:30], "\n") + fmt.Sprintf("\n... (%d more files)", len(lines)-30)
	}

	return result
}

// =====================
// 編集系ツール
// =====================

// executeStrReplace はファイル内の文字列を置換
func executeStrReplace(path string, oldStr string, newStr string) (string, string, error) {
	if path == "" {
		return "Error: path is required", "", nil
	}
	if oldStr == "" {
		return "Error: old_str is required", "", nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	// ファイルを読み込む
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err), "", nil
	}

	oldContent := string(content)
	var newContent string

	// まず完全一致を試行
	exactMatch := strings.Contains(oldContent, oldStr)
	exactCount := strings.Count(oldContent, oldStr)

	if exactMatch && exactCount == 1 {
		// 完全一致が1つ → そのまま使用
		newContent = strings.Replace(oldContent, oldStr, newStr, 1)
	} else if exactMatch && exactCount > 1 {
		// 完全一致が複数 → エラー
		lines := strings.Split(oldContent, "\n")
		previewLines := min(50, len(lines))
		preview := strings.Join(lines[:previewLines], "\n")

		return fmt.Sprintf(`Error: old_str appears %d times in %s (must be unique).

Hint: Include more context (surrounding lines) to make old_str unique.
For example, include the function signature or class definition.

File preview (first %d lines):
---
%s
---

Please use read_file to see the full content and choose a unique old_str.`,
			exactCount, path, previewLines, preview), "", nil
	} else {
		// 完全一致しない → 正規化マッチを試行
		yellow.Println("⚠️  Exact match failed, trying normalized whitespace matching...")

		found, startIdx, endIdx := findWithNormalizedWhitespace(oldContent, oldStr)

		if !found {
			return fmt.Sprintf("Error: old_str not found in %s (tried both exact and normalized matching)", path), "", nil
		}

		// 正規化マッチで見つかった部分を置換
		actualOldStr := oldContent[startIdx : endIdx+1]
		newContent = oldContent[:startIdx] + newStr + oldContent[endIdx+1:]

		yellow.Printf("ℹ️  Matched with normalized whitespace (indentation may differ)\n")
		yellow.Printf("   Actual match in file:\n")
		// 実際のマッチ部分をプレビュー表示
		matchLines := strings.Split(actualOldStr, "\n")
		for i, line := range matchLines {
			if i >= 5 {
				yellow.Printf("   ... (%d more lines)\n", len(matchLines)-5)
				break
			}
			yellow.Printf("   │ %s\n", line)
		}
		fmt.Println()
	}

	// 確認UI
	oldStrLines := strings.Split(oldStr, "\n")
	newStrLines := strings.Split(newStr, "\n")

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🔧 Text Replacement / テキスト置換\n")
	cyan.Printf("📂 File / ファイル: %s\n", path)
	cyan.Printf("📊 Changes / 変更: -%d lines, +%d lines / %d 行削除・%d 行追加\n",
		len(oldStrLines), len(newStrLines), len(oldStrLines), len(newStrLines))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 改善された差分表示
	showImprovedDiff(oldStr, newStr)

	// 確認
	if !confirm("Apply this replacement? / この置換を適用しますか？") {
		yellow.Println("⚠️  User cancelled the replacement")
		return fmt.Sprintf(`[CANCELLED] User cancelled str_replace for %s.

Hint: The replacement was not applied. If you need to make this change:
1. Check if the old_str is correct by using read_file
2. Try a smaller, more specific replacement
3. Ask the user for clarification

Do not retry the same replacement.`, path), "", nil
	}

	// バックアップ作成
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Warning: failed to create backup: %v (continuing anyway)", err), "", nil
	}

	// 保存
	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), "", nil
	}

	green.Printf("✅ Replaced in: %s\n", path)
	return fmt.Sprintf("Successfully replaced text in %s", path), backupPath, nil
}

// min は2つの整数の小さい方を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =====================
// Web検索ツール
// =====================

// executeWebSearch は Serper API を使って Web 検索を実行
func executeWebSearch(query string) string {
	if query == "" {
		return "Error: query is required"
	}

	green.Printf("🔍 Searching the web for: %s\n", query)

	result, err := api.WebSearch(query)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return result
}

// ===== Phase 3: Advanced File Editing Tools =====

// executeInsertAfter はパターンマッチした行の後に内容を挿入
func executeInsertAfter(path, pattern, content string) (string, string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: Invalid file path: %v", err), "", nil
	}

	// ファイル読み込み
	fileContent, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Cannot read file: %v", err), "", nil
	}

	lines := strings.Split(string(fileContent), "\n")

	// Tier 1: Exact match
	matchIdx := -1
	matchCount := 0
	var matchIndices []int

	for i, line := range lines {
		if line == pattern {
			matchIdx = i
			matchCount++
			matchIndices = append(matchIndices, i)
		}
	}

	// Tier 2: Normalized whitespace match
	if matchIdx == -1 {
		normalizedPattern := normalizeLeadingWhitespace(pattern)
		for i, line := range lines {
			if normalizeLeadingWhitespace(line) == normalizedPattern {
				matchIdx = i
				matchCount++
				matchIndices = append(matchIndices, i)
			}
		}
	}

	// パターンが見つからない場合
	if matchIdx == -1 {
		yellow.Printf("⚠️  Pattern not found / パターンが見つかりません: %s\n\n", pattern)
		yellow.Println("File preview (first 50 lines) / ファイルプレビュー (最初50行):")
		for i := 0; i < min(len(lines), 50); i++ {
			fmt.Printf("%4d: %s\n", i+1, lines[i])
		}
		if len(lines) > 50 {
			yellow.Printf("... (%d more lines)\n", len(lines)-50)
		}
		return fmt.Sprintf("Error: Pattern not found in %s", path), "", nil
	}

	// 複数マッチの場合
	if matchCount > 1 {
		red.Printf("⚠️  Error: Pattern matches %d locations (must be unique)\n", matchCount)
		red.Println("⚠️  エラー: パターンが複数の場所にマッチします（一意である必要があります）\n")
		yellow.Println("All match locations / すべてのマッチ場所:")
		for _, idx := range matchIndices {
			start := max(0, idx-2)
			end := min(len(lines), idx+3)
			for i := start; i < end; i++ {
				prefix := "  "
				if i == idx {
					prefix = "→ "
				}
				fmt.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
			}
			fmt.Println()
		}
		return fmt.Sprintf("Error: Pattern matched %d times (must be unique)", matchCount), "", nil
	}

	// コンテキスト表示（マッチ行の前後5行）
	green.Printf("✅ Pattern found at line %d / パターンが見つかりました (行 %d)\n\n", matchIdx+1, matchIdx+1)
	yellow.Println("Context / コンテキスト (5 lines before/after):")
	start := max(0, matchIdx-5)
	end := min(len(lines), matchIdx+6)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == matchIdx {
			prefix = "→ "
		}
		fmt.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
	}
	cyan.Println("\n━━━━ Content to insert / 挿入する内容 ━━━━")
	fmt.Println(content)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// バックアップ作成
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Failed to create backup: %v", err), "", nil
	}
	green.Printf("📦 Backup created: %s\n", backupPath)

	// 挿入実行
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:matchIdx+1]...)
	newLines = append(newLines, content)
	newLines = append(newLines, lines[matchIdx+1:]...)

	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(absPath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Sprintf("Error: Failed to write file: %v", err), "", nil
	}

	return fmt.Sprintf("✅ Inserted after line %d in %s", matchIdx+1, path), backupPath, nil
}

// executeInsertBefore はパターンマッチした行の前に内容を挿入
func executeInsertBefore(path, pattern, content string) (string, string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: Invalid file path: %v", err), "", nil
	}

	// ファイル読み込み
	fileContent, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Cannot read file: %v", err), "", nil
	}

	lines := strings.Split(string(fileContent), "\n")

	// Tier 1: Exact match
	matchIdx := -1
	matchCount := 0
	var matchIndices []int

	for i, line := range lines {
		if line == pattern {
			matchIdx = i
			matchCount++
			matchIndices = append(matchIndices, i)
		}
	}

	// Tier 2: Normalized whitespace match
	if matchIdx == -1 {
		normalizedPattern := normalizeLeadingWhitespace(pattern)
		for i, line := range lines {
			if normalizeLeadingWhitespace(line) == normalizedPattern {
				matchIdx = i
				matchCount++
				matchIndices = append(matchIndices, i)
			}
		}
	}

	// パターンが見つからない場合
	if matchIdx == -1 {
		yellow.Printf("⚠️  Pattern not found / パターンが見つかりません: %s\n\n", pattern)
		yellow.Println("File preview (first 50 lines) / ファイルプレビュー (最初50行):")
		for i := 0; i < min(len(lines), 50); i++ {
			fmt.Printf("%4d: %s\n", i+1, lines[i])
		}
		if len(lines) > 50 {
			yellow.Printf("... (%d more lines)\n", len(lines)-50)
		}
		return fmt.Sprintf("Error: Pattern not found in %s", path), "", nil
	}

	// 複数マッチの場合
	if matchCount > 1 {
		red.Printf("⚠️  Error: Pattern matches %d locations (must be unique)\n", matchCount)
		red.Println("⚠️  エラー: パターンが複数の場所にマッチします（一意である必要があります）\n")
		yellow.Println("All match locations / すべてのマッチ場所:")
		for _, idx := range matchIndices {
			start := max(0, idx-2)
			end := min(len(lines), idx+3)
			for i := start; i < end; i++ {
				prefix := "  "
				if i == idx {
					prefix = "→ "
				}
				fmt.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
			}
			fmt.Println()
		}
		return fmt.Sprintf("Error: Pattern matched %d times (must be unique)", matchCount), "", nil
	}

	// コンテキスト表示（マッチ行の前後5行）
	green.Printf("✅ Pattern found at line %d / パターンが見つかりました (行 %d)\n\n", matchIdx+1, matchIdx+1)
	yellow.Println("Context / コンテキスト (5 lines before/after):")
	start := max(0, matchIdx-5)
	end := min(len(lines), matchIdx+6)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == matchIdx {
			prefix = "→ "
		}
		fmt.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
	}
	cyan.Println("\n━━━━ Content to insert / 挿入する内容 ━━━━")
	fmt.Println(content)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// バックアップ作成
	backupPath, err := createBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Failed to create backup: %v", err), "", nil
	}
	green.Printf("📦 Backup created: %s\n", backupPath)

	// 挿入実行
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:matchIdx]...)
	newLines = append(newLines, content)
	newLines = append(newLines, lines[matchIdx:]...)

	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(absPath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Sprintf("Error: Failed to write file: %v", err), "", nil
	}

	return fmt.Sprintf("✅ Inserted before line %d in %s", matchIdx+1, path), backupPath, nil
}

// executeCopyFile はファイルをコピー
func executeCopyFile(src, dest string) (string, string, error) {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return fmt.Sprintf("Error: Invalid source path: %v", err), "", nil
	}

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Sprintf("Error: Invalid destination path: %v", err), "", nil
	}

	// ソースファイル確認
	srcInfo, err := os.Stat(absSrc)
	if err != nil {
		return fmt.Sprintf("Error: Source file not found: %v", err), "", nil
	}
	if srcInfo.IsDir() {
		return fmt.Sprintf("Error: Source is a directory (use recursive copy for directories): %s", src), "", nil
	}

	// 送信先が存在するかチェック
	destExists := false
	var destBackupPath string
	if _, err := os.Stat(absDest); err == nil {
		destExists = true

		// 確認UI表示
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("📋 Copy File / ファイルコピー\n")
		cyan.Printf("📂 Source / コピー元: %s\n", src)
		cyan.Printf("📂 Destination / コピー先: %s\n", dest)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: Destination file already exists / 警告: コピー先ファイルが既に存在します")

		if !confirm("Overwrite? / 上書きしますか？") {
			return "Cancelled by user", "", nil
		}

		// バックアップ作成
		destBackupPath, err = createBackup(absDest)
		if err != nil {
			yellow.Printf("Warning: Failed to create backup: %v\n", err)
		} else {
			green.Printf("📦 Backup created: %s\n", destBackupPath)
		}
	}

	// ファイルコピー実行
	srcFile, err := os.Open(absSrc)
	if err != nil {
		return fmt.Sprintf("Error: Cannot open source file: %v", err), "", nil
	}
	defer srcFile.Close()

	destFile, err := os.Create(absDest)
	if err != nil {
		return fmt.Sprintf("Error: Cannot create destination file: %v", err), "", nil
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Sprintf("Error: Failed to copy file: %v", err), "", nil
	}

	// パーミッション保持
	err = os.Chmod(absDest, srcInfo.Mode())
	if err != nil {
		yellow.Printf("Warning: Failed to preserve permissions: %v\n", err)
	}

	if destExists {
		return fmt.Sprintf("✅ File copied (overwritten): %s → %s", src, dest), destBackupPath, nil
	}
	return fmt.Sprintf("✅ File copied: %s → %s", src, dest), "", nil
}
