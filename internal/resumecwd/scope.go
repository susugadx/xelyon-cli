package resumecwd

import (
	"path/filepath"
	"runtime"
	"strings"
)

// Matches は保存済み session の working dir と現在の working dir が同じ resume scope かを返す。
func Matches(sessionWorkingDir, currentWorkingDir string) bool {
	sessionDir := normalize(sessionWorkingDir)
	if sessionDir == "" {
		return true
	}
	currentDir := normalize(currentWorkingDir)
	if currentDir == "" {
		return true
	}
	return sameNormalizedPathForOS(sessionDir, currentDir, runtime.GOOS)
}

func normalize(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func sameNormalizedPathForOS(left, right, goos string) bool {
	if goos == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
