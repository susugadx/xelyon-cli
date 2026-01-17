package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// executeRestoreBackup はバックアップファイルからファイルを復元
// path: 復元対象のファイルパス（オリジナルファイル）
// backup_path: 特定のバックアップファイルパス（オプション）
func executeRestoreBackup(path, backupPath string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	// バックアップパスが指定されていない場合、最新のバックアップを探す
	if backupPath == "" {
		var err error
		backupPath, err = findLatestBackup(path)
		if err != nil {
			return "", err
		}
	}

	// バックアップファイルの存在確認
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return "", fmt.Errorf("backup file not found: %s", backupPath)
	}

	// バックアップの内容を読み込み
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to read backup file: %w", err)
	}

	// 確認プロンプト
	yellow.Printf("🔄 Restore file from backup\n")
	fmt.Printf("  Target: %s\n", path)
	fmt.Printf("  Backup: %s\n", backupPath)
	fmt.Printf("  Size: %d bytes\n", len(backupContent))

	decision := Confirm("Restore file? ")
	if decision.Action == ConfirmNo {
		return "Restore cancelled by user", nil
	}
	if decision.Action == ConfirmComment {
		return fmt.Sprintf("Restore cancelled with comment: %s", decision.Comment), nil
	}

	// 復元先ディレクトリの存在確認
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// ファイルを復元（パーミッションは0644）
	if err := os.WriteFile(path, backupContent, 0644); err != nil {
		return "", fmt.Errorf("failed to restore file: %w", err)
	}

	green.Printf("✅ Restored: %s\n", path)
	green.Printf("   From: %s\n", backupPath)

	return fmt.Sprintf("Successfully restored %s from backup %s", path, backupPath), nil
}

// findLatestBackup は指定ファイルの最新バックアップを検索
func findLatestBackup(filePath string) (string, error) {
	dir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)

	// バックアップファイルパターン: filename.bak.YYYYMMDD_HHMMSS
	pattern := fmt.Sprintf("%s.bak.*", baseName)

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return "", fmt.Errorf("failed to search backups: %w", err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no backup found for %s", filePath)
	}

	// タイムスタンプでソート（新しい順）
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))

	return matches[0], nil
}

// listBackups は指定ファイルのバックアップ一覧を取得
func listBackups(filePath string) ([]string, error) {
	dir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)

	pattern := fmt.Sprintf("%s.bak.*", baseName)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to search backups: %w", err)
	}

	// タイムスタンプでソート（新しい順）
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))

	return matches, nil
}

// executeListBackups はファイルのバックアップ一覧を表示
func executeListBackups(path string) string {
	if path == "" {
		return "Error: path is required"
	}

	backups, err := listBackups(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if len(backups) == 0 {
		return fmt.Sprintf("No backups found for %s", path)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Backups for %s:\n", path))
	for i, backup := range backups {
		info, err := os.Stat(backup)
		if err != nil {
			sb.WriteString(fmt.Sprintf("  %d. %s (error reading)\n", i+1, filepath.Base(backup)))
			continue
		}
		sb.WriteString(fmt.Sprintf("  %d. %s (%d bytes, %s)\n",
			i+1,
			filepath.Base(backup),
			info.Size(),
			info.ModTime().Format("2006-01-02 15:04:05"),
		))
	}
	return sb.String()
}
