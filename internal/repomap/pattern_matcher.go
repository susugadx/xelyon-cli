package repomap

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func extensionForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		return ext
	}
	base := strings.ToLower(filepath.Base(path))
	switch base {
	case "dockerfile":
		return "dockerfile"
	default:
		return ""
	}
}

func supportsSymbols(path string) bool {
	if ast.IsSupportedFile(path) {
		return true
	}
	return defaultPatternEngine.supports(path)
}

func matchesSymbolPattern(path, line string) bool {
	return defaultPatternEngine.matches(path, line)
}

func isCommentLikeLine(path, line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	switch extensionForPath(path) {
	case ".py", ".rb", ".sh", ".bash", ".zsh":
		return strings.HasPrefix(trimmed, "#")
	default:
		return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*")
	}
}
