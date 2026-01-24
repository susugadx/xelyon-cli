package common

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// CreateBackup はファイルのタイムスタンプ付きバックアップを作成
func CreateBackup(filePath string) (string, error) {
	// ファイルが存在しない場合はスキップ（新規作成）
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", nil
	}

	// .gitignore自動追加（初回のみ）
	if err := EnsureGitignore(filepath.Dir(filePath)); err != nil {
		Yellow.Printf("Warning: Failed to update .gitignore: %v\n", err)
		// .gitignore追加失敗してもバックアップは続行
	}

	// タイムスタンプ付きバックアップパス生成
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.bak.%s", filePath, timestamp)

	// バックアップ作成
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file for backup: %w", err)
	}

	// 元ファイルのパーミッションを引き継ぐ（少なくとも Perm() 部分）
	perm := os.FileMode(0644)
	if info, statErr := os.Stat(filePath); statErr == nil {
		perm = info.Mode().Perm()
	}
	if err := os.WriteFile(backupPath, content, perm); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// 古いバックアップを削除（maxGenerationsを超えたもの）
	if err := CleanupOldBackups(filePath); err != nil {
		Yellow.Printf("Warning: Failed to cleanup old backups: %v\n", err)
		// 削除失敗してもバックアップは成功とみなす
	}

	return backupPath, nil
}

// CleanupOldBackups は古いバックアップファイルを削除
func CleanupOldBackups(filePath string) error {
	// 設定から最大世代数を取得
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	maxGenerations := cfg.Backup.MaxGenerations
	if maxGenerations <= 0 {
		maxGenerations = 5 // デフォルト
	}

	// 同じファイルのバックアップを検索
	dir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)
	pattern := fmt.Sprintf("%s.bak.*", baseName)

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}

	// バックアップが最大世代数以下なら削除不要
	if len(matches) <= maxGenerations {
		return nil
	}

	// タイムスタンプでソート（古い順）
	sort.Strings(matches)

	// 古いものから削除
	deleteCount := len(matches) - maxGenerations
	for i := 0; i < deleteCount; i++ {
		if err := os.Remove(matches[i]); err != nil {
			Yellow.Printf("Warning: Failed to delete old backup %s: %v\n", matches[i], err)
			// 個別のファイル削除失敗は続行
		}
	}

	return nil
}
