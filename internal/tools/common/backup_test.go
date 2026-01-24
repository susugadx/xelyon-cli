package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestCreateBackup_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")

	// ファイルが存在しない場合、バックアップは作成されない
	backupPath, err := CreateBackup(testFile)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("CreateBackup() should return empty string for non-existent file, got %v", backupPath)
	}
}

func TestCreateBackup_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	testContent := "original content"

	// ファイル作成
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// バックアップ作成
	backupPath, err := CreateBackup(testFile)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	if backupPath == "" {
		t.Fatal("CreateBackup() should return backup path for existing file")
	}

	// バックアップファイルが存在することを確認
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file not created: %v", backupPath)
	}

	// バックアップの内容を確認
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != testContent {
		t.Errorf("Backup content = %v, want %v", string(backupContent), testContent)
	}

	// バックアップファイル名のフォーマット確認
	if !strings.Contains(backupPath, ".bak.") {
		t.Error("Backup path should contain '.bak.' timestamp")
	}
}

func TestCleanupOldBackups(t *testing.T) {
	tmpDir := testutil.SetupTempHome(t)
	testFile := filepath.Join(tmpDir, "test.txt")

	// デフォルト設定作成（MaxGenerations=5）
	cfg := config.DefaultConfig()
	cfg.Backup.MaxGenerations = 3 // テスト用に3世代に設定
	config.SetGlobalConfig(cfg)
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// 5つのバックアップファイルを作成
	for i := 0; i < 5; i++ {
		backupPath := filepath.Join(tmpDir, "test.txt.bak.2024010"+string(rune('1'+i))+"_120000")
		if err := os.WriteFile(backupPath, []byte("backup"), 0644); err != nil {
			t.Fatalf("Failed to create backup file: %v", err)
		}
	}

	// 元ファイル作成
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// クリーンアップ実行
	err := CleanupOldBackups(testFile)
	if err != nil {
		t.Fatalf("CleanupOldBackups() error = %v", err)
	}

	// バックアップファイル数を確認（3つ残る）
	matches, err := filepath.Glob(filepath.Join(tmpDir, "test.txt.bak.*"))
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}

	if len(matches) != 3 {
		t.Errorf("CleanupOldBackups() left %d backups, want 3", len(matches))
	}
}
