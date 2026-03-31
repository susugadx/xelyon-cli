package file

import (
	"fmt"
	"os"
	"strings"

	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	toolslsp "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ExecuteDeleteFileWithPromptIOAndOptions deletes a file with explicit confirm options.
func ExecuteDeleteFileWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path string) (string, error) {
	return ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(promptIO, options, nil, path)
}

// ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient deletes a file with explicit confirm options and LSP client.
func ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(promptIO ui.PromptIO, options common.ConfirmOptions, lspClient *lsplib.Client, path string) (string, error) {
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
	if lspClient != nil {
		language := lsplib.DetectLanguage(absPath)
		if language != "" {
			symbols := toolslsp.ExtractSymbolsFromContent(string(content), language)
			if len(symbols) > 0 {
				refs, hasExternal, _ := toolslsp.CheckReferencesBeforeDeleteWithClient(lspClient, absPath, symbols)
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
		w := out.StdoutWriter()
		ui.FileOpHeader(w, "delete_file", fmt.Sprintf("%s (%d bytes, %d lines)", path, fileInfo.Size(), len(lines)))
		out.Red.Println("  DESTRUCTIVE: file will be permanently deleted")

		preview := buildCappedPatchPreviewLines(lines, '-', resolvePreviewLineCap(options.Config))
		ui.ShowSinglePatchPreview(w, ui.PatchFilePreview{
			Path:    path,
			Action:  "deleted",
			Removed: len(lines),
			Hunks: []ui.PatchHunkPreview{{
				StartLine: 1,
				Lines:     preview.lines,
			}},
		})
		if preview.truncated {
			out.Dim.Printf("  preview truncated: showing first %d of %d lines\n", len(preview.lines), preview.totalLines)
			if preview.truncatedByBytes {
				out.Dim.Printf("  preview truncated at %dKB\n", maxFullBodyPreviewBytes/1024)
			}
		}

		// 外部参照の警告表示
		if len(externalRefs) > 0 {
			out.Println()
			out.Yellow.Printf("  LSP: %d external references to symbols in this file\n", len(externalRefs))

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
						out.Dim.Printf("      ... and %d more\n", len(refs)-3)
						break
					}
					out.Printf("      - %s:%d\n", ref.FilePath, ref.Line)
					shown++
				}
			}
			out.Println()
			out.Red.Println("  Deleting this file may break code that references these symbols!")
		}
	}

	dec := common.ConfirmToolAction(promptIO, options, "delete_file", "Delete this file? / このファイルを削除しますか？", common.ToolConfirmContext{TargetPath: absPath})
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
