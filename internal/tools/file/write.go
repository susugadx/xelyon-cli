package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ExecuteWriteFile はファイルに書き込む
func ExecuteWriteFile(path string, content string) (string, string, error) {
	if path == "" {
		return "Error: path is empty", "", nil
	}

	// パストラバーサル防止
	absPath, err := common.ValidatePath(path)
	if err != nil {
		common.Red.Printf("🚫 Security: %v\n", err)
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	// ファイルが存在するか確認 + 元のパーミッション取得
	exists := false
	perm := os.FileMode(0644)
	if info, err := os.Stat(absPath); err == nil {
		exists = true
		perm = info.Mode().Perm()
	}

	// 確認UI - 変更サマリーを明確に表示
	newLines := strings.Split(content, "\n")

	common.Cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if exists {
		common.Cyan.Printf("📝 write_file (overwrite): %s\n", path)
	} else {
		common.Cyan.Printf("📝 write_file (create): %s\n", path)
	}
	common.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 変更サマリー
	common.Yellow.Println("\n📊 Summary / 変更サマリー:")
	if exists {
		oldContent, _ := os.ReadFile(absPath)
		oldLines := strings.Split(string(oldContent), "\n")
		lineDiff := len(newLines) - len(oldLines)
		fmt.Printf("   • Before: %d lines / 変更前: %d行\n", len(oldLines), len(oldLines))
		fmt.Printf("   • After: %d lines / 変更後: %d行\n", len(newLines), len(newLines))
		if lineDiff > 0 {
			common.Green.Printf("   • Net: +%d lines\n", lineDiff)
		} else if lineDiff < 0 {
			common.Red.Printf("   • Net: %d lines\n", lineDiff)
		} else {
			fmt.Printf("   • Net: 0 lines (same size)\n")
		}

		// 既存ファイルの全体上書き警告
		// 行数の変化が少ないのに大きなファイルを全体上書きしようとしている場合
		absLineDiff := lineDiff
		if absLineDiff < 0 {
			absLineDiff = -absLineDiff
		}
		if absLineDiff < 10 && len(oldLines) > 50 {
			common.Red.Println("\n🚨 WARNING: Large file overwrite with minimal changes!")
			common.Red.Println("   あなたは大きなファイルを少ない変更で全体上書きしようとしています。")
			common.Yellow.Println("💡 Consider using str_replace for partial edits instead.")
			common.Yellow.Println("   部分的な編集には str_replace の使用を検討してください。")
		}

		common.ShowDiff(string(oldContent), content, path)
	} else {
		fmt.Printf("   • New file: %d lines / 新規: %d行\n", len(newLines), len(newLines))
		fmt.Printf("   • Size: %d bytes\n", len(content))
		common.ShowPreview(content)
	}

	dec := common.ConfirmWithAutoApproveDecision("write_file", "Create/overwrite this file? / このファイルを作成・上書きしますか？")
	switch dec.Action {
	case common.ConfirmYes:
		// continue
	case common.ConfirmComment:
		return fmt.Sprintf(`[COMMENT] User provided feedback for write_file.

Comment:
%s

Next actions:
- Use read_file to verify current file contents.
- Consider using str_replace for partial modifications.

IMPORTANT: Do NOT write the file until the user approves.`, strings.TrimSpace(dec.Comment)), "", nil
	default:
		return "Cancelled by user", "", nil
	}

	// バックアップ作成（既存ファイルの場合のみ）
	backupPath, err := common.CreateBackup(absPath)
	if err != nil {
		return fmt.Sprintf("Warning: failed to create backup: %v (continuing anyway)", err), "", nil
	}

	// ディレクトリ作成
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err), "", nil
	}

	// 書き込み（既存ファイルがある場合は元のパーミッションを維持）
	if err := os.WriteFile(absPath, []byte(content), perm); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), "", nil
	}

	common.Green.Printf("✅ Written: %s\n", path)
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), backupPath, nil
}
