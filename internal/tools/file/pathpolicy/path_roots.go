package pathpolicy

import (
	"path/filepath"
	"strings"
)

// NormalizeWorkspaceRoot は workspace root / cwd の比較用正規化を行う。
func NormalizeWorkspaceRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

// EvaluateWorkspaceRoot は可能なら symlink 解決後の root を返す。
func EvaluateWorkspaceRoot(path string) string {
	path = NormalizeWorkspaceRoot(path)
	if path == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved)
	}
	return path
}

// AppendUniqueString は空文字を除外しながら候補を一度だけ追加する。
func AppendUniqueString(values []string, candidate string) []string {
	if candidate == "" || containsString(values, candidate) {
		return values
	}
	return append(values, candidate)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
