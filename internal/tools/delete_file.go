package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

// executeDeleteFile deletes a file permanently (with backup for Undo)
func executeDeleteFile(path string) (string, string, error) {
	// パストラバーサル防止
	absPath, err := ValidatePath(path)
	if err != nil {
		red.Printf("🚫 Security: %v\n", err)
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	// ファイル存在確認
	fileInfo, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return "Error: File not found", "", nil
	}
	if err != nil {
		return fmt.Sprintf("Error: Cannot access file: %v", err), "", nil
	}

	// ディレクトリでないことを確認
	if fileInfo.IsDir() {
		return "Error: Cannot delete directory (path is a directory)", "", nil
	}

	// ファイル内容読み込み
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Failed to read file: %v", err), "", nil
	}
	lines := strings.Split(string(content), "\n")

	// LSP参照チェック（LSPクライアントが有効な場合のみ）
	var externalRefs []ReferenceInfo
	if LSPClient != nil {
		language := lsp.DetectLanguage(absPath)
		if language != "" {
			symbols := ExtractSymbolsFromContent(string(content), language)
			if len(symbols) > 0 {
				refs, hasExternal, _ := CheckReferencesBeforeDelete(absPath, symbols)
				if hasExternal {
					externalRefs = GetExternalReferences(refs)
				}
			}
		}
	}

	// 確認UI表示（ファイルプレビュー付き）
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🗑️  Delete File / ファイル削除\n")
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Printf("📏 Size / サイズ: %d bytes (%d lines)\n", fileInfo.Size(), len(lines))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	red.Println("⚠️  DESTRUCTIVE: File will be permanently deleted!")
	red.Println("⚠️  破壊的操作: ファイルは完全に削除されます!")

	// ファイルプレビュー（最初20行）
	yellow.Println("\nFile preview (first 20 lines) / ファイルプレビュー:")
	previewLines := 20
	if len(lines) < previewLines {
		previewLines = len(lines)
	}
	for i := 0; i < previewLines; i++ {
		fmt.Printf("  %4d: %s\n", i+1, lines[i])
	}
	if len(lines) > 20 {
		yellow.Printf("  ... (%d more lines)\n", len(lines)-20)
	}

	// 外部参照の警告表示
	if len(externalRefs) > 0 {
		fmt.Println()
		yellow.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Printf("⚠️  LSP Warning: This file contains %d external references!\n", len(externalRefs))
		yellow.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// シンボルごとにグループ化して表示
		symbolRefs := make(map[string][]ReferenceInfo)
		for _, ref := range externalRefs {
			symbolRefs[ref.Symbol] = append(symbolRefs[ref.Symbol], ref)
		}

		for symbol, refs := range symbolRefs {
			yellow.Printf("   %s (%d references):\n", symbol, len(refs))
			shown := 0
			for _, ref := range refs {
				if shown >= 3 {
					yellow.Printf("      ... and %d more\n", len(refs)-3)
					break
				}
				fmt.Printf("      - %s:%d\n", ref.FilePath, ref.Line)
				shown++
			}
		}
		fmt.Println()
		red.Println("⚠️  Deleting this file may break the code that references these symbols!")
	}

	dec := confirmWithAutoApproveDecision("delete_file", "Delete this file? / このファイルを削除しますか？")
	switch dec.Action {
	case ConfirmYes:
		// continue
	case ConfirmComment:
		return fmt.Sprintf(`[COMMENT] User provided feedback for delete_file.

Comment:
%s

Next actions:
- Use read_file to confirm the correct target.
- Consider delete_lines or move_file if appropriate.

IMPORTANT: Do NOT delete the file until the user approves.`, strings.TrimSpace(dec.Comment)), "", nil
	default:
		return "Cancelled by user", "", nil
	}

	// 削除前に必ずバックアップ作成（削除後は不可能）
	backupPath, err := createBackup(absPath)
	if err != nil {
		// バックアップ失敗時は削除を中止（安全第一）
		return fmt.Sprintf("Error: Backup failed, deletion ABORTED: %v", err), "", nil
	}
	green.Printf("📦 Backup created: %s\n", backupPath)

	// ファイル削除
	if err := os.Remove(absPath); err != nil {
		return fmt.Sprintf("Error: Failed to delete file: %v", err), "", nil
	}

	return fmt.Sprintf("✅ Deleted: %s", path), backupPath, nil
}
