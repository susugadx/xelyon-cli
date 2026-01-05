package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
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

// 自動実行可能なコマンド（安全なもの）
var safeCommands = map[string]bool{
	"ls": true, "cat": true, "pwd": true, "echo": true, "which": true,
	"head": true, "tail": true, "wc": true, "grep": true, "find": true,
	"git status": true, "git log": true, "git diff": true, "git branch": true,
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

// Execute はツールを実行
func Execute(tc *ToolCall) string {
	cyan.Printf("🔧 %s", tc.Tool)
	if len(tc.Args) > 0 {
		fmt.Printf(": %v\n", tc.Args)
	} else {
		fmt.Println()
	}

	switch tc.Tool {
    case "git_status":
        return executeGitStatus()
    case "git_diff":
        path := tc.Args["path"]
        return executeGitDiff(path)
    case "git_add":
        path := tc.Args["path"]
        if path == "" {
            path = "."
        }
        return executeGitAdd(path)
    case "git_commit":
        message := tc.Args["message"]
        return executeGitCommit(message)
    case "git_push":
        return executeGitPush()
    case "git_log":
        return executeGitLog()
	case "bash":
		return executeBash(tc.Args["command"])
	case "read_file":
		return executeReadFile(tc.Args["path"])
	case "write_file":
		return executeWriteFile(tc.Args["path"], tc.Args["content"])
	case "list_dir":
		path := tc.Args["path"]
		if path == "" {
			path = "."
		}
		return executeListDir(path)
	default:
		return fmt.Sprintf("Unknown tool: %s", tc.Tool)
	}
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
		yellow.Printf("⚠️  Execute: %s\n", command)
		if !confirm("Run this command?") {
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

	fmt.Println(result)
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
func executeWriteFile(path string, content string) string {
	if path == "" {
		return "Error: path is empty"
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// ファイルが存在するか確認
	exists := false
	if _, err := os.Stat(absPath); err == nil {
		exists = true
	}

	// diff表示（既存ファイルの場合）
	if exists {
		oldContent, _ := os.ReadFile(absPath)
		showDiff(string(oldContent), content, path)
	} else {
		yellow.Printf("📝 New file: %s\n", path)
		showPreview(content)
	}

	// 確認
	action := "Create"
	if exists {
		action = "Overwrite"
	}
	if !confirm(fmt.Sprintf("%s this file?", action)) {
		return "Cancelled by user"
	}

	// ディレクトリ作成
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err)
	}

	// 書き込み
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}

	green.Printf("✅ Written: %s\n", path)
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)
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
	fmt.Println(result)
	return result
}

// confirm はユーザーに確認を求める
func confirm(message string) bool {
	yellow.Printf("%s (y/n): ", message)

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
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
    fmt.Println(result)
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
    fmt.Println(result)
    return result
}

// executeGitAdd は git add を実行
func executeGitAdd(path string) string {
    green.Printf("➕ git add %s\n", path)

    // 確認
    if !confirm(fmt.Sprintf("Stage '%s'?", path)) {
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

    yellow.Printf("💾 git commit -m \"%s\"\n", message)

    // 確認
    if !confirm("Commit with this message?") {
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
    yellow.Println("🚀 git push")

    // 確認
    if !confirm("Push to remote?") {
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
    fmt.Println(result)
    return result
}