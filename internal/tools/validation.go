package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validatePathImpl はパストラバーサル攻撃を防ぐためにパスを検証（実装）
func validatePathImpl(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	// 絶対パスに変換
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// カレントディレクトリを取得
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// カレントディレクトリの絶対パス
	allowedDir, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to resolve allowed directory: %w", err)
	}

	// パスがカレントディレクトリ配下にあるかチェック
	if !strings.HasPrefix(absPath, allowedDir) {
		return "", fmt.Errorf("path escape attempt detected: %s is outside of %s", absPath, allowedDir)
	}

	// Clean してシンボリックリンク攻撃を防ぐ
	cleanPath := filepath.Clean(absPath)

	// 再度チェック（Clean後も境界内か）
	if !strings.HasPrefix(cleanPath, allowedDir) {
		return "", fmt.Errorf("path escape attempt detected after clean: %s", cleanPath)
	}

	return cleanPath, nil
}

// ValidatePath はパストラバーサル攻撃を防ぐためにパスを検証（テスト用にグローバル変数として定義）
var ValidatePath = validatePathImpl

// ValidatePathAllowParent は親ディレクトリへのアクセスを許可
// （特定の用途でのみ使用）
func ValidatePathAllowParent(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	// 絶対パスに変換
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// システムの重要ディレクトリへのアクセスを防ぐ
	blockedPaths := []string{
		"/etc", "/var", "/usr", "/bin", "/sbin", "/boot", "/sys", "/proc",
		"/root", "/home", // ホームディレクトリ以外
	}

	for _, blocked := range blockedPaths {
		if strings.HasPrefix(absPath, blocked) {
			return "", fmt.Errorf("access to system directory is blocked: %s", blocked)
		}
	}

	return filepath.Clean(absPath), nil
}
