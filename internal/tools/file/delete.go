package file

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	toolslsp "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ExecuteDeleteFile deletes a file permanently
func ExecuteDeleteFile(path string) (string, error) {
	return ExecuteDeleteFileWithOutput(common.DefaultOutput(), path)
}

// ExecuteDeleteFileWithOutput deletes a file permanently with explicit output writers.
func ExecuteDeleteFileWithOutput(out common.Output, path string) (string, error) {
	return ExecuteDeleteFileWithPromptIO(ui.NewPromptIO(nil, out.StdoutWriter(), out.StderrWriter(), nil), path)
}

// ExecuteDeleteFileWithPromptIO deletes a file permanently with explicit interactive I/O.
func ExecuteDeleteFileWithPromptIO(promptIO ui.PromptIO, path string) (string, error) {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := common.NewOutput(promptIO.Out, promptIO.Err)

	// パストラバーサル防止
	absPath, err := common.ValidatePath(path)
	if err != nil {
		out.Red.Printf("🚫 Security: %v\n", err)
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
	} else {
		out.Yellow.Println("ℹ️  LSP not connected — external reference check skipped")
	}

	// 確認UI表示（ファイルプレビュー付き）
	if !out.SuppressStdout() {
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		out.Cyan.Printf("🗑️  Delete File / ファイル削除\n")
		out.Cyan.Printf("📂 Path / パス: %s\n", path)
		out.Cyan.Printf("📏 Size / サイズ: %d bytes (%d lines)\n", fileInfo.Size(), len(lines))
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		out.Red.Println("⚠️  DESTRUCTIVE: File will be permanently deleted!")
		out.Red.Println("⚠️  破壊的操作: ファイルは完全に削除されます!")

		cfg := config.GetGlobalConfig()
		maxPreviewLines := cfg.Diff.MaxTotalLines
		if maxPreviewLines <= 0 {
			maxPreviewLines = len(lines)
		}

		if len(lines) > maxPreviewLines {
			out.Yellow.Printf("\nFile preview (first %d of %d lines) / ファイルプレビュー:\n", maxPreviewLines, len(lines))
		} else {
			out.Yellow.Printf("\nFile contents (%d lines) / ファイル内容:\n", len(lines))
		}
		for i := 0; i < len(lines) && i < maxPreviewLines; i++ {
			out.Printf("  %4d: %s\n", i+1, lines[i])
		}
		if len(lines) > maxPreviewLines {
			out.Yellow.Printf("  ... (%d more lines)\n", len(lines)-maxPreviewLines)
		}

		// 外部参照の警告表示
		if len(externalRefs) > 0 {
			out.Println()
			out.Yellow.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			out.Yellow.Printf("⚠️  LSP Warning: This file contains %d external references!\n", len(externalRefs))
			out.Yellow.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			// シンボルごとにグループ化して表示
			symbolRefs := make(map[string][]toolslsp.ReferenceInfo)
			for _, ref := range externalRefs {
				symbolRefs[ref.Symbol] = append(symbolRefs[ref.Symbol], ref)
			}

			for symbol, refs := range symbolRefs {
				out.Yellow.Printf("   %s (%d references):\n", symbol, len(refs))
				shown := 0
				for _, ref := range refs {
					if shown >= 3 {
						out.Yellow.Printf("      ... and %d more\n", len(refs)-3)
						break
					}
					out.Printf("      - %s:%d\n", ref.FilePath, ref.Line)
					shown++
				}
			}
			out.Println()
			out.Red.Println("⚠️  Deleting this file may break the code that references these symbols!")
		}
	}

	dec := common.ConfirmWithAutoApproveDecision(promptIO, "delete_file", "Delete this file? / このファイルを削除しますか？")
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
