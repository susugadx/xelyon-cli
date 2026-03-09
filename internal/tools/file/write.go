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

	// 確認UI - 変更サマリーを明確に表示
	newLines := strings.Split(content, "\n")
	if !out.SuppressStdout() {
		out.Cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if exists {
			out.Cyan.Printf("📝 write_file (overwrite): %s\n", path)
		} else {
			out.Cyan.Printf("📝 write_file (create): %s\n", path)
		}
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// 変更サマリー
		out.Yellow.Println("\n📊 Summary / 変更サマリー:")
		if exists {
			oldContent, _ := os.ReadFile(absPath)
			oldLines := strings.Split(string(oldContent), "\n")
			lineDiff := len(newLines) - len(oldLines)
			out.Printf("   • Before: %d lines / 変更前: %d行\n", len(oldLines), len(oldLines))
			out.Printf("   • After: %d lines / 変更後: %d行\n", len(newLines), len(newLines))
			if lineDiff > 0 {
				out.Green.Printf("   • Net: +%d lines\n", lineDiff)
			} else if lineDiff < 0 {
				out.Red.Printf("   • Net: %d lines\n", lineDiff)
			} else {
				out.Printf("   • Net: 0 lines (same size)\n")
			}

			// 既存ファイルの全体上書き警告
			// 行数の変化が少ないのに大きなファイルを全体上書きしようとしている場合
			absLineDiff := lineDiff
			if absLineDiff < 0 {
				absLineDiff = -absLineDiff
			}
			if absLineDiff < 10 && len(oldLines) > 50 {
				out.Red.Println("\n🚨 WARNING: Large file overwrite with minimal changes!")
				out.Red.Println("   あなたは大きなファイルを少ない変更で全体上書きしようとしています。")
				out.Yellow.Println("💡 Consider using str_replace for partial edits instead.")
				out.Yellow.Println("   部分的な編集には str_replace の使用を検討してください。")
			}

			common.ShowDiffWithOutputAndConfig(out, cfg, string(oldContent), content, path)
		} else {
			out.Printf("   • New file: %d lines / 新規: %d行\n", len(newLines), len(newLines))
			out.Printf("   • Size: %d bytes\n", len(content))
			common.ShowPreviewWithOutputAndConfig(out, cfg, content)
		}
	}

	dec := common.ConfirmWithAutoApproveDecisionAndOptions(promptIO, options, "write_file", "Create/overwrite this file? / このファイルを作成・上書きしますか？")
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
