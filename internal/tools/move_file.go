package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// executeMoveFile moves/renames a file atomically
func executeMoveFile(src, dest string) (string, string, error) {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return fmt.Sprintf("Error: Invalid source path: %v", err), "", nil
	}

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Sprintf("Error: Invalid destination path: %v", err), "", nil
	}

	// ソースファイル確認
	srcInfo, err := os.Stat(absSrc)
	if os.IsNotExist(err) {
		return "Error: Source file not found", "", nil
	}
	if err != nil {
		return fmt.Sprintf("Error: Cannot access source file: %v", err), "", nil
	}
	if srcInfo.IsDir() {
		return "Error: Source is a directory (file only)", "", nil
	}

	// 同一ファイルチェック（no-op）
	if absSrc == absDest {
		return "No operation: Source and destination are identical", "", nil
	}

	// 移動先の親ディレクトリ存在確認
	destDir := filepath.Dir(absDest)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Sprintf("Error: Destination directory does not exist: %s", destDir), "", nil
	}

	// 移動先の衝突処理
	destExists := false
	var destBackupPath string

	if _, err := os.Stat(absDest); err == nil {
		destExists = true

		// 確認UI表示
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("📂 Move File / ファイル移動\n")
		cyan.Printf("📄 Source / 移動元: %s\n", src)
		cyan.Printf("📄 Destination / 移動先: %s\n", dest)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: Destination file already exists")
		yellow.Println("⚠️  警告: 移動先ファイルが既に存在します")

		if !confirm("Overwrite destination? / 移動先を上書きしますか？") {
			return "Cancelled by user", "", nil
		}

		// 移動先のバックアップ作成（移動元ではない！）
		destBackupPath, err = createBackup(absDest)
		if err != nil {
			yellow.Printf("Warning: Failed to create backup: %v\n", err)
		} else {
			green.Printf("📦 Backup created: %s\n", destBackupPath)
		}
	}

	// アトミック移動（フォールバック付き）
	err = os.Rename(absSrc, absDest)

	if err != nil {
		// クロスファイルシステムエラーの場合はコピー+削除にフォールバック
		if strings.Contains(err.Error(), "cross-device") || strings.Contains(err.Error(), "invalid cross-device link") {
			yellow.Println("⚠️  Cross-filesystem move detected, using copy+delete fallback")

			// コピー処理
			srcFile, err := os.Open(absSrc)
			if err != nil {
				return fmt.Sprintf("Error: Cannot open source: %v", err), "", nil
			}
			defer srcFile.Close()

			destFile, err := os.Create(absDest)
			if err != nil {
				return fmt.Sprintf("Error: Cannot create destination: %v", err), "", nil
			}
			defer destFile.Close()

			if _, err = io.Copy(destFile, srcFile); err != nil {
				return fmt.Sprintf("Error: Copy failed: %v", err), "", nil
			}

			// パーミッション保持
			if err = os.Chmod(absDest, srcInfo.Mode()); err != nil {
				yellow.Printf("Warning: Failed to preserve permissions: %v\n", err)
			}

			// ソースファイル削除
			if err = os.Remove(absSrc); err != nil {
				yellow.Printf("⚠️  Warning: Copy succeeded but failed to delete source: %v\n", err)
				yellow.Printf("   Manual cleanup required: %s\n", absSrc)
				return fmt.Sprintf("⚠️  Partially completed: Destination created but source remains"), destBackupPath, nil
			}
		} else {
			return fmt.Sprintf("Error: Move failed: %v", err), "", nil
		}
	}

	if destExists {
		return fmt.Sprintf("✅ Moved (overwritten): %s → %s", src, dest), destBackupPath, nil
	}
	return fmt.Sprintf("✅ Moved: %s → %s", src, dest), "", nil
}
