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

// ExecuteWriteFileWithPromptIOAndOptions は確認設定を指定してファイルに書き込む。
func ExecuteWriteFileWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path string, content string) (string, error) {
	return ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(promptIO, options, nil, path, content)
}

// ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient は確認設定と LSP client を指定してファイルに書き込む。
func ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(promptIO ui.PromptIO, options common.ConfirmOptions, lspClient *lsplib.Client, path string, content string) (string, error) {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := common.NewOutput(promptIO.Out, promptIO.Err)
	cfg := options.Config
	if cfg == nil {
		return "", fmt.Errorf("missing confirm options config")
	}

	if path == "" {
		return "Error: path is empty", nil
	}

	// パストラバーサル防止
	absPath, err := common.ValidatePath(path)
	if err != nil {
		out.Red.Printf("🚫 Security: %v\n", err)
		return fmt.Sprintf("Error: %v", err), nil
	}

	// 派生パス警告: file.go_temp, file.go.new 等の疑わしいコピーファイル作成をブロック
	if basePath := detectDerivativeBase(absPath); basePath != "" {
		if _, err := os.Stat(basePath); err == nil {
			relBase, _ := filepath.Rel(filepath.Dir(absPath), basePath)
			return fmt.Sprintf("Error: '%s' looks like a derivative/copy of '%s'. Edit the original file using str_replace instead of creating a copy.", path, relBase), nil
		}
	}

	// ファイルが存在するか確認 + 元のパーミッション取得
	exists := false
	// 新規ファイルは 0644 固定（実行権限が必要な場合は別途 chmod を使用）
	perm := os.FileMode(0644)
	if info, err := os.Stat(absPath); err == nil {
		exists = true
		perm = info.Mode().Perm()
	}

	// 確認UI
	newLines := strings.Split(content, "\n")
	if !out.SuppressStdout() {
		w := out.StdoutWriter()
		out.Println()
		if exists {
			ui.FileOpHeader(w, "write_file", path+" (overwrite)")

			oldContent, _ := os.ReadFile(absPath)
			oldLines := strings.Split(string(oldContent), "\n")
			added, removed := ui.CountDiffLines(oldLines, newLines)
			ui.FileOpStatsLine(w, removed, added)

			// 既存ファイルの全体上書き警告
			lineDiff := len(newLines) - len(oldLines)
			absLineDiff := lineDiff
			if absLineDiff < 0 {
				absLineDiff = -absLineDiff
			}
			if absLineDiff < 10 && len(oldLines) > 50 {
				out.Yellow.Println("  Large file overwrite with minimal changes. Consider using str_replace.")
			}

			common.ShowDiffWithOutputAndConfig(out, cfg, string(oldContent), content, path)
		} else {
			ui.FileOpHeader(w, "write_file", path+" (create)")
			out.Dim.Printf("  new: %d lines, %d bytes\n", len(newLines), len(content))
			preview := buildCappedPatchPreviewLines(newLines, '+', resolvePreviewLineCap(cfg))
			ui.ShowSinglePatchPreview(w, ui.PatchFilePreview{
				Path:   path,
				Action: "created",
				Added:  preview.totalLines,
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
		}
	}

	dec := common.ConfirmToolAction(promptIO, options, "write_file", "Create/overwrite this file? / このファイルを作成・上書きしますか？", common.ToolConfirmContext{TargetPath: absPath})
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

IMPORTANT: Do NOT write the file until the user approves.`, strings.TrimSpace(dec.Comment)), nil
	default:
		return "Cancelled by user", nil
	}

	// ディレクトリ作成
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err), nil
	}

	// Atomic write: tmp ファイルに書き込み後、rename で差し替え
	tmpFile, err := os.CreateTemp(dir, ".xelyon-write-*.tmp")
	if err != nil {
		return fmt.Sprintf("Error creating temp file: %v", err), nil
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
		return fmt.Sprintf("Error writing temp file: %v", err), nil
	}
	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return fmt.Sprintf("Error setting permissions: %v", err), nil
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Sprintf("Error closing temp file: %v", err), nil
	}

	if err := os.Rename(tmpPath, absPath); err != nil {
		// Windows は既存ファイル上書き rename が失敗する場合がある
		if runtime.GOOS == "windows" {
			if rmErr := os.Remove(absPath); rmErr != nil && !os.IsNotExist(rmErr) {
				return fmt.Sprintf("Error replacing file: %v", err), nil
			}
			if rnErr := os.Rename(tmpPath, absPath); rnErr != nil {
				return fmt.Sprintf("Error replacing file: %v", rnErr), nil
			}
		} else {
			return fmt.Sprintf("Error replacing file: %v", err), nil
		}
	}
	success = true

	lineCount := strings.Count(content, "\n") + 1
	out.Green.Printf("✅ Written: %s (%d lines)\n", path, lineCount)
	msg := fmt.Sprintf("Successfully wrote %d bytes (%d lines) to %s", len(content), lineCount, path)
	msg += toolslsp.GetDiagnosticsSummaryWithClient(lspClient, absPath)
	return msg, nil
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
