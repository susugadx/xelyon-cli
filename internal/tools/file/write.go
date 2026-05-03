package file

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	toolslsp "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type writeExecutionDetails struct {
	result       fileMutationResult
	resolvedPath string
}

// ExecuteWriteFileWithPromptIOAndOptions は確認設定を指定してファイルに書き込む。
func ExecuteWriteFileWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path string, content string) (string, error) {
	return ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(promptIO, options, nil, path, content)
}

// ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient は確認設定と LSP client を指定してファイルに書き込む。
func ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(promptIO ui.PromptIO, options common.ConfirmOptions, lspClient *lsplib.Client, path string, content string) (string, error) {
	details, err := executeWriteFileWithPromptIOAndOptionsAndLSPClientDetails(promptIO, options, lspClient, path, content)
	return details.result.message, err
}

func executeWriteFileWithPromptIOAndOptionsAndLSPClientDetails(promptIO ui.PromptIO, options common.ConfirmOptions, lspClient *lsplib.Client, path string, content string) (writeExecutionDetails, error) {
	ctx, result, err := prepareFileMutation(promptIO, options, path, "path is empty")
	if result.message != "" || err != nil {
		return writeExecutionDetails{result: result}, err
	}
	out := ctx.out
	absPath := ctx.absPath
	newLines := strings.Split(content, "\n")
	exists := false
	perm := os.FileMode(0644)

	workflowResult, workflowErr := executeFileMutationWorkflow(ctx, options, fileMutationWorkflow{
		toolName:       "write_file",
		confirmMessage: "Create/overwrite this file? / このファイルを作成・上書きしますか？",
		preview: func() fileMutationResult {
			if ctx.cfg == nil {
				return newErrorMutationResult("Error: missing confirm options config")
			}
			if basePath := detectDerivativeBase(absPath); basePath != "" {
				if _, err := os.Stat(basePath); err == nil {
					relBase, _ := filepath.Rel(filepath.Dir(absPath), basePath)
					return newErrorMutationResult(fmt.Sprintf("Error: '%s' looks like a derivative/copy of '%s'. Edit the original file using str_replace instead of creating a copy.", path, relBase))
				}
			}

			if info, err := os.Stat(absPath); err == nil {
				exists = true
				perm = info.Mode().Perm()
			}

			if out.SuppressStdout() {
				return fileMutationResult{}
			}

			w := out.StdoutWriter()
			out.Println()
			if exists {
				ui.FileOpHeader(w, "write_file", path+" (overwrite)")

				oldContent, _ := os.ReadFile(absPath)
				oldLines := strings.Split(string(oldContent), "\n")
				added, removed := ui.CountDiffLines(oldLines, newLines)
				ui.FileOpStatsLine(w, removed, added)

				lineDiff := len(newLines) - len(oldLines)
				if lineDiff < 0 {
					lineDiff = -lineDiff
				}
				if lineDiff < 10 && len(oldLines) > 50 {
					out.Yellow.Println("  Large file overwrite with minimal changes. Consider using str_replace.")
				}

				common.ShowDiffWithOutputAndConfig(out, ctx.cfg, string(oldContent), content, path)
				return fileMutationResult{}
			}

			ui.FileOpHeader(w, "write_file", path+" (create)")
			out.Dim.Printf("  new: %d lines, %d bytes\n", len(newLines), len(content))
			showCappedSinglePatchPreview(out, ctx.cfg, ui.PatchFilePreview{
				Path:   path,
				Action: "created",
			}, newLines, '+')
			return fileMutationResult{}
		},
		confirm: mutationConfirmHandlers{
			onComment: func(comment string) fileMutationResult {
				return newCommentMutationResult(buildCommentResponse("write_file", comment,
					"Use read_file to verify current file contents.",
					"Consider using str_replace for partial modifications.",
				))
			},
			onCancel: func() fileMutationResult { return newCancelledMutationResult("Cancelled by user") },
		},
		apply: func() (fileMutationResult, error) {
			dir := filepath.Dir(absPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return newErrorMutationResult(fmt.Sprintf("Error creating directory: %v", err)), nil
			}

			tmpFile, err := os.CreateTemp(dir, ".xelyon-write-*.tmp")
			if err != nil {
				return newErrorMutationResult(fmt.Sprintf("Error creating temp file: %v", err)), nil
			}
			tmpPath := tmpFile.Name()

			success := false
			defer func() {
				if !success {
					_ = os.Remove(tmpPath)
				}
			}()

			if _, err := tmpFile.WriteString(content); err != nil {
				_ = tmpFile.Close()
				return newErrorMutationResult(fmt.Sprintf("Error writing temp file: %v", err)), nil
			}
			if err := tmpFile.Chmod(perm); err != nil {
				_ = tmpFile.Close()
				return newErrorMutationResult(fmt.Sprintf("Error setting permissions: %v", err)), nil
			}
			if err := tmpFile.Close(); err != nil {
				return newErrorMutationResult(fmt.Sprintf("Error closing temp file: %v", err)), nil
			}

			if err := os.Rename(tmpPath, absPath); err != nil {
				if runtime.GOOS == "windows" {
					if rmErr := os.Remove(absPath); rmErr != nil && !os.IsNotExist(rmErr) {
						return newErrorMutationResult(fmt.Sprintf("Error replacing file: %v", err)), nil
					}
					if rnErr := os.Rename(tmpPath, absPath); rnErr != nil {
						return newErrorMutationResult(fmt.Sprintf("Error replacing file: %v", rnErr)), nil
					}
				} else {
					return newErrorMutationResult(fmt.Sprintf("Error replacing file: %v", err)), nil
				}
			}
			success = true

			lineCount := strings.Count(content, "\n") + 1
			out.Green.Printf("✅ Written: %s (%d lines)\n", path, lineCount)
			msg := fmt.Sprintf("Successfully wrote %d bytes (%d lines) to %s", len(content), lineCount, path)
			msg += toolslsp.GetDiagnosticsSummaryWithClient(lspClient, absPath)
			return newAppliedMutationResult(msg), nil
		},
	})
	return writeExecutionDetails{
		result:       workflowResult,
		resolvedPath: absPath,
	}, workflowErr
}

// detectDerivativeBase は派生パス（file.go_temp, file.go.new 等）のベースパスを返す。
// 派生パスでない場合は空文字を返す。
func detectDerivativeBase(absPath string) string {
	// パターン1: file.go_temp, file.go_new 等（拡張子の後にサフィックス付加）
	for _, suffix := range []string{"_temp", "_new", "_backup", "_copy", "_old", "_bak", "_orig", "_tmp"} {
		if strings.HasSuffix(absPath, suffix) {
			return strings.TrimSuffix(absPath, suffix)
		}
	}
	// パターン2: file.go.tmp, file.go.new 等（二重拡張子）
	ext := filepath.Ext(absPath)
	for _, dext := range []string{".tmp", ".new", ".bak", ".orig", ".backup", ".copy", ".old", ".temp"} {
		if ext == dext {
			return strings.TrimSuffix(absPath, dext)
		}
	}
	return ""
}
