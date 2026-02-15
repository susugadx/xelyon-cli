package file

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	toolslsp "github.com/susugadx/xelyon-cli/internal/tools/lsp"
)

// ExecuteDeleteFile deletes a file permanently
func ExecuteDeleteFile(path string) (string, error) {
	// パストラバーサル防止
	absPath, err := common.ValidatePath(path)
	if err != nil {
		common.Red.Printf("🚫 Security: %v\n", err)
		return fmt.Sprintf("Error: %v", err), nil
	}

	// ファイル存在確認
	fileInfo, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return "Error: File not found", nil
	}
	if err != nil {
		return fmt.Sprintf("Error: Cannot access file: %v", err), nil
	}

	// ディレクトリでないことを確認
	if fileInfo.IsDir() {
		return "Error: Cannot delete directory (path is a directory)", nil
	}

	// ファイル内容読み込み
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error: Failed to read file: %v", err), nil
	}
	lines := strings.Split(string(content), "\n")

	// LSP参照チェック（LSPクライアントが有効な場合のみ）
	var externalRefs []toolslsp.ReferenceInfo
	if toolslsp.LSPClient != nil {
		language := lsp.DetectLanguage(absPath)
		if language != "" {
			symbols := toolslsp.ExtractSymbolsFromContent(string(content), language)
			if len(symbols) > 0 {
				refs, hasExternal, _ := toolslsp.CheckReferencesBeforeDelete(absPath, symbols)
				if hasExternal {
					externalRefs = toolslsp.GetExternalReferences(refs)
				}
			}
		}
	}

	// 確認UI表示（ファイルプレビュー付き）
	common.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	common.Cyan.Printf("🗑️  Delete File / ファイル削除\n")
	common.Cyan.Printf("📂 Path / パス: %s\n", path)
	common.Cyan.Printf("📏 Size / サイズ: %d bytes (%d lines)\n", fileInfo.Size(), len(lines))
	common.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	common.Red.Println("⚠️  DESTRUCTIVE: File will be permanently deleted!")
	common.Red.Println("⚠️  破壊的操作: ファイルは完全に削除されます!")

	// ファイルプレビュー（最初20行）
	common.Yellow.Println("\nFile preview (first 20 lines) / ファイルプレビュー:")
	previewLines := 20
	if len(lines) < previewLines {
		previewLines = len(lines)
	}
	for i := 0; i < previewLines; i++ {
		fmt.Printf("  %4d: %s\n", i+1, lines[i])
	}
	if len(lines) > 20 {
		common.Yellow.Printf("  ... (%d more lines)\n", len(lines)-20)
	}

	// 外部参照の警告表示
	if len(externalRefs) > 0 {
		fmt.Println()
		common.Yellow.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		common.Yellow.Printf("⚠️  LSP Warning: This file contains %d external references!\n", len(externalRefs))
		common.Yellow.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// シンボルごとにグループ化して表示
		symbolRefs := make(map[string][]toolslsp.ReferenceInfo)
		for _, ref := range externalRefs {
			symbolRefs[ref.Symbol] = append(symbolRefs[ref.Symbol], ref)
		}

		for symbol, refs := range symbolRefs {
			common.Yellow.Printf("   %s (%d references):\n", symbol, len(refs))
			shown := 0
			for _, ref := range refs {
				if shown >= 3 {
					common.Yellow.Printf("      ... and %d more\n", len(refs)-3)
					break
				}
				fmt.Printf("      - %s:%d\n", ref.FilePath, ref.Line)
				shown++
			}
		}
		fmt.Println()
		common.Red.Println("⚠️  Deleting this file may break the code that references these symbols!")
	}

	dec := common.ConfirmWithAutoApproveDecision("delete_file", "Delete this file? / このファイルを削除しますか？")
	switch dec.Action {
	case common.ConfirmYes:
		// continue
	case common.ConfirmComment:
		return fmt.Sprintf(`[COMMENT] User provided feedback for delete_file.

Comment:
%s

Next actions:
- Use read_file to confirm the correct target.
- Consider delete_lines or move_file if appropriate.

IMPORTANT: Do NOT delete the file until the user approves.`, strings.TrimSpace(dec.Comment)), nil
	default:
		return "Cancelled by user", nil
	}

	// ファイル削除
	if err := os.Remove(absPath); err != nil {
		return fmt.Sprintf("Error: Failed to delete file: %v", err), nil
	}

	return fmt.Sprintf("✅ Deleted: %s", path), nil
}
