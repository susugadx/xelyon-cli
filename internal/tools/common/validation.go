package common

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

	// Clean 前チェック
	cleanPath := filepath.Clean(absPath)
	if !strings.HasPrefix(cleanPath, allowedDir) {
		return "", fmt.Errorf("path escape attempt detected: %s is outside of %s", cleanPath, allowedDir)
	}

	// シンボリックリンク解決後に再チェック（ファイルが存在する場合）
	if _, statErr := os.Lstat(cleanPath); statErr == nil {
		realPath, err := filepath.EvalSymlinks(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve symlinks: %w", err)
		}

		realAllowed, err := filepath.EvalSymlinks(allowedDir)
		if err != nil {
			realAllowed = allowedDir
		}

		if !strings.HasPrefix(realPath, realAllowed) {
			return "", fmt.Errorf("symlink escape attempt detected: %s resolves to %s (outside %s)", cleanPath, realPath, realAllowed)
		}
		return realPath, nil
	}

	// ファイルが存在しない場合は親ディレクトリのシンボリックリンクを確認
	parentDir := filepath.Dir(cleanPath)
	if _, statErr := os.Lstat(parentDir); statErr == nil {
		realParent, err := filepath.EvalSymlinks(parentDir)
		if err == nil {
			realAllowed, _ := filepath.EvalSymlinks(allowedDir)
			if realAllowed == "" {
				realAllowed = allowedDir
			}
			if !strings.HasPrefix(realParent, realAllowed) {
				return "", fmt.Errorf("symlink escape attempt in parent directory: %s", parentDir)
			}
		}
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
