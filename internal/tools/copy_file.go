package tools

import (
	"fmt"
	"io"
	"os"
)

// executeCopyFile はファイルをコピー
func executeCopyFile(src, dest string) (string, string, error) {
	// パストラバーサル防止
	absSrc, err := ValidatePath(src)
	if err != nil {
		red.Printf("🚫 Security (source): %v\n", err)
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	absDest, err := ValidatePath(dest)
	if err != nil {
		red.Printf("🚫 Security (destination): %v\n", err)
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	// ソースファイル確認
	srcInfo, err := os.Stat(absSrc)
	if err != nil {
		return fmt.Sprintf("Error: Source file not found: %v", err), "", nil
	}
	if srcInfo.IsDir() {
		return fmt.Sprintf("Error: Source is a directory (use recursive copy for directories): %s", src), "", nil
	}

	// 送信先が存在するかチェック
	destExists := false
	var destBackupPath string
	if _, err := os.Stat(absDest); err == nil {
		destExists = true

		// 確認UI表示
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("📋 Copy File / ファイルコピー\n")
		cyan.Printf("📂 Source / コピー元: %s\n", src)
		cyan.Printf("📂 Destination / コピー先: %s\n", dest)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: Destination file already exists / 警告: コピー先ファイルが既に存在します")

		if !confirm("Overwrite? / 上書きしますか？") {
			return "Cancelled by user", "", nil
		}

		// バックアップ作成
		destBackupPath, err = createBackup(absDest)
		if err != nil {
			yellow.Printf("Warning: Failed to create backup: %v\n", err)
		} else {
			green.Printf("📦 Backup created: %s\n", destBackupPath)
		}
	}

	// ファイルコピー実行
	srcFile, err := os.Open(absSrc)
	if err != nil {
		return fmt.Sprintf("Error: Cannot open source file: %v", err), "", nil
	}
	defer srcFile.Close()

	destFile, err := os.Create(absDest)
	if err != nil {
		return fmt.Sprintf("Error: Cannot create destination file: %v", err), "", nil
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Sprintf("Error: Failed to copy file: %v", err), "", nil
	}

	// パーミッション保持
	err = os.Chmod(absDest, srcInfo.Mode())
	if err != nil {
		yellow.Printf("Warning: Failed to preserve permissions: %v\n", err)
	}

	if destExists {
		return fmt.Sprintf("✅ File copied (overwritten): %s → %s", src, dest), destBackupPath, nil
	}
	return fmt.Sprintf("✅ File copied: %s → %s", src, dest), "", nil
}
