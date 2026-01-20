package tools

import (
	"fmt"
	"os"
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
		headOutput, headErr := ExecuteGitCommand("show", "HEAD:"+target)
		headContent := ""
		if headErr == nil {
			headContent = headOutput
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

		dec := Confirm("Restore from HEAD? / HEADから復元しますか？")
		switch dec.Action {
		case ConfirmYes:
			// continue
		case ConfirmComment:
			return fmt.Sprintf(`[COMMENT] User provided feedback for git_checkout (file restore).

Comment:
%s

Next actions:
- Confirm the correct target file and whether restoring from HEAD is intended.
- Consider stashing changes before restoring.

IMPORTANT: Do NOT discard local changes until the user approves.`, strings.TrimSpace(dec.Comment)), "", nil
		default:
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
		output, err := ExecuteGitCommand("checkout", "HEAD", "--", target)
		if err != nil {
			return FormatGitError(err, output), "", nil
		}

		return fmt.Sprintf("✅ Restored from HEAD: %s", target), backupPath, nil
	}

	// ブランチチェックアウト
	green.Printf("🔀 Detected branch target: %s\n", target)
	yellow.Println("ℹ️  Tip: Use git_branch with action='switch' for more options")

	// 未コミット変更チェック
	status, hasChanges, _ := CheckUncommittedChanges()

	if hasChanges {
		// 確認UI
		DisplayGitConfirmHeader(
			"🔀 Git Checkout Branch / ブランチチェックアウト",
			"🌿 Target / 対象",
			target,
		)
		DisplayUncommittedChangesWarning(status)

		dec := Confirm("Checkout anyway? / それでもチェックアウトしますか？")
		switch dec.Action {
		case ConfirmYes:
			// continue
		case ConfirmComment:
			return fmt.Sprintf(`[COMMENT] User provided feedback for git_checkout (branch switch).

Comment:
%s

Next actions:
- Consider using git_stash to save changes before switching.
- Or commit changes before switching.

IMPORTANT: Do NOT switch branches until the user approves.`, strings.TrimSpace(dec.Comment)), "", nil
		default:
			return "Cancelled by user", "", nil
		}
	}

	output, err := ExecuteGitCommand("checkout", target)
	if err != nil {
		return FormatGitError(err, output), "", nil
	}

	return fmt.Sprintf("✅ Checked out: %s\n%s", target, output), "", nil
}
