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

// ExecuteDeleteFileWithPromptIOAndOptions は確認設定を指定してファイルを削除する。
func ExecuteDeleteFileWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path string) (string, error) {
	return ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(promptIO, options, nil, path)
}

// ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient は確認設定と LSP client を指定してファイルを削除する。
func ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(promptIO ui.PromptIO, options common.ConfirmOptions, lspClient *lsplib.Client, path string) (string, error) {
	result, err := executeDeleteFileWithPromptIOAndOptionsAndLSPClient(promptIO, options, lspClient, path)
	return result.message, err
}

func executeDeleteFileWithPromptIOAndOptionsAndLSPClient(promptIO ui.PromptIO, options common.ConfirmOptions, lspClient *lsplib.Client, path string) (fileMutationResult, error) {
	ctx, result, err := prepareFileMutation(promptIO, options, path, "path is empty")
	if result.message != "" || err != nil {
		return result, err
	}
	out := ctx.out
	absPath := ctx.absPath
	var fileInfo os.FileInfo
	var lines []string
	var externalRefs []toolslsp.ReferenceInfo

	return executeFileMutationWorkflow(ctx, options, fileMutationWorkflow{
		toolName:       "delete_file",
		confirmMessage: "Delete this file? / このファイルを削除しますか？",
		preview: func() fileMutationResult {
			info, err := os.Stat(absPath)
			if os.IsNotExist(err) {
				return newErrorMutationResult("Error: File not found")
			}
			if err != nil {
				return newErrorMutationResult(fmt.Sprintf("Error: Cannot access file: %v", err))
			}
			if info.IsDir() {
				return newErrorMutationResult("Error: Cannot delete directory (path is a directory)")
			}
			fileInfo = info

			content, err := os.ReadFile(absPath)
			if err != nil {
				return newErrorMutationResult(fmt.Sprintf("Error: Failed to read file: %v", err))
			}
			lines = strings.Split(string(content), "\n")

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

			if out.SuppressStdout() {
				return fileMutationResult{}
			}

			w := out.StdoutWriter()
			ui.FileOpHeader(w, "delete_file", fmt.Sprintf("%s (%d bytes, %d lines)", path, fileInfo.Size(), len(lines)))
			out.Red.Println("  DESTRUCTIVE: file will be permanently deleted")

			showCappedSinglePatchPreview(out, options.Config, ui.PatchFilePreview{
				Path:    path,
				Action:  "deleted",
				Removed: len(lines),
			}, lines, '-')

			if len(externalRefs) == 0 {
				return fileMutationResult{}
			}

			out.Println()
			out.Yellow.Printf("  LSP: %d external references to symbols in this file\n", len(externalRefs))
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
			return fileMutationResult{}
		},
		confirm: mutationConfirmHandlers{
			onComment: func(comment string) fileMutationResult {
				return newCommentMutationResult(buildCommentResponse("delete_file", comment,
					"Use read_file to confirm the correct target.",
					"Consider delete_lines or move_file if appropriate.",
				))
			},
			onCancel: func() fileMutationResult { return newCancelledMutationResult("Cancelled by user") },
		},
		apply: func() (fileMutationResult, error) {
			if err := os.Remove(absPath); err != nil {
				return newErrorMutationResult(fmt.Sprintf("Error: Failed to delete file: %v", err)), nil
			}
			return newAppliedMutationResult(fmt.Sprintf("✅ Deleted: %s", path)), nil
		},
	})
}
