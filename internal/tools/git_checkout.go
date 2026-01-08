package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
