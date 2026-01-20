package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecuteGitCommand はGitコマンドを実行し結果を返す
func ExecuteGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// FormatGitError はGitコマンドのエラーをフォーマット
func FormatGitError(err error, output string) string {
	if output != "" {
		return fmt.Sprintf("Error: %v\n%s", err, output)
	}
	return fmt.Sprintf("Error: %v", err)
}

// FormatGitSuccess はGit成功メッセージをフォーマット
func FormatGitSuccess(action, target string) string {
	return fmt.Sprintf("✅ %s: %s", action, target)
}

// CheckUncommittedChanges は未コミット変更をチェック
// status: git status --porcelain の出力
// hasChanges: 変更があるかどうか
func CheckUncommittedChanges() (status string, hasChanges bool, err error) {
	output, err := ExecuteGitCommand("status", "--porcelain")
	if err != nil {
		return "", false, err
	}
	return output, len(strings.TrimSpace(output)) > 0, nil
}

// TruncateOutput は出力を指定行数に切り詰め
func TruncateOutput(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}
	result := strings.Join(lines[:maxLines], "\n")
	return fmt.Sprintf("%s\n... (%d more lines)", result, len(lines)-maxLines)
}

// DisplayGitConfirmHeader は確認UIのヘッダーを表示
func DisplayGitConfirmHeader(title, subtitle, target string) {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("%s\n", title)
	cyan.Printf("%s: %s\n", subtitle, target)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// DisplayUncommittedChangesWarning は未コミット変更の警告を表示
func DisplayUncommittedChangesWarning(status string) {
	yellow.Println("⚠️  Warning: You have uncommitted changes / 警告: 未コミットの変更があります")
	fmt.Println("\nUncommitted changes / 未コミットの変更:")
	fmt.Println(status)
}
