package dev

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ExecuteLint runs linter with optional auto-fix
func ExecuteLint(path, autoFixStr string) (string, string, error) {
	absPath := path
	if absPath == "" {
		absPath = "."
	}

	var err error
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Invalid path: %v", err), "", nil
	}

	// auto_fix文字列をboolに変換
	autoFix := (autoFixStr == "true")

	// リンター検出
	linterName, checkCmd, fixCmd := DetectLinter(absPath)
	if linterName == "" {
		red.Println("⚠️  No linter detected for this project")
		yellow.Println("Supported linters:")
		yellow.Println("  - Go: golangci-lint, go vet")
		yellow.Println("  - JavaScript/TypeScript: eslint")
		yellow.Println("  - Python: ruff, pylint")
		yellow.Println("  - Rust: clippy")
		return "Error: No linter detected", "", nil
	}

	green.Printf("🔍 Detected linter: %s\n", linterName)

	// Phase 1: チェック実行（自動修正なし）
	green.Printf("▶ Running: %s\n", checkCmd)
	checkCmdExec := exec.Command("bash", "-c", checkCmd)
	checkCmdExec.Dir = absPath
	checkOutput, checkErr := checkCmdExec.CombinedOutput()

	// 出力を表示
	fmt.Println(string(checkOutput))

	// 問題が検出されたか判定
	hasIssues := (checkErr != nil) || len(checkOutput) > 0

	if !hasIssues {
		green.Println("✅ No issues found")
		return "No issues found", "", nil
	}

	// Phase 2: 自動修正（オプション）
	if autoFix {
		if fixCmd == "" {
			yellow.Printf("⚠️  Auto-fix not supported for %s\n", linterName)
			return fmt.Sprintf("Issues found (auto-fix not supported)\n%s", string(checkOutput)), "", nil
		}

		// 確認UI
		cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("🔧 Lint Auto-fix / リント自動修正\n")
		cyan.Printf("📂 Path / パス: %s\n", path)
		cyan.Printf("🛠️  Linter / リンター: %s\n", linterName)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: Auto-fix will modify files")
		yellow.Println("⚠️  警告: 自動修正でファイルが変更されます")

		dec := common.Confirm("Run auto-fix? / 自動修正を実行しますか？")
		switch dec.Action {
		case common.ConfirmYes:
			// continue
		case common.ConfirmComment:
			return fmt.Sprintf(`[COMMENT] User provided feedback for lint auto-fix.

Comment:
%s

Next actions:
- Consider running lint without auto-fix first.
- Or apply fixes selectively.

IMPORTANT: Do NOT run auto-fix until the user approves.`, strings.TrimSpace(dec.Comment)), "", nil
		default:
			return "Auto-fix cancelled by user", "", nil
		}

		// バックアップ作成（制限: 単一パスのみ）
		var backupPath string
		if fileInfo, err := os.Stat(absPath); err == nil && !fileInfo.IsDir() {
			// パスがファイルの場合のみバックアップ
			var backupErr error
			backupPath, backupErr = common.CreateBackup(absPath)
			if backupErr != nil {
				yellow.Printf("Warning: Failed to create backup: %v\n", backupErr)
				yellow.Println("Continuing without backup (changes may be irreversible)")
			} else if backupPath != "" {
				green.Printf("📦 Backup created: %s\n", backupPath)
			}
		} else {
			// ディレクトリの場合はバックアップなし（制限）
			yellow.Println("⚠️  Note: Directory-wide auto-fix does not create backup (limitation)")
		}

		// 自動修正実行
		green.Printf("▶ Running: %s\n", fixCmd)
		fixCmdExec := exec.Command("bash", "-c", fixCmd)
		fixCmdExec.Dir = absPath
		fixOutput, fixErr := fixCmdExec.CombinedOutput()

		fmt.Println(string(fixOutput))

		if fixErr != nil {
			return fmt.Sprintf("Auto-fix completed with errors:\n%s", string(fixOutput)), backupPath, nil
		}

		return "✅ Auto-fix completed successfully", backupPath, nil
	}

	return fmt.Sprintf("Issues found (use auto_fix to apply fixes)\n%s", string(checkOutput)), "", nil
}
