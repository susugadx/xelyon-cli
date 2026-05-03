package file

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func newFileChange(displayPath, resolvedPath, toolName, description string, linesAdded, linesRemoved int) *tools.FileChange {
	change := &tools.FileChange{
		FilePath:     displayPath,
		Timestamp:    common.GetCurrentTime(),
		Tool:         toolName,
		Description:  description,
		LinesAdded:   linesAdded,
		LinesRemoved: linesRemoved,
	}
	if detailPath := normalizeResolvedPath(resolvedPath); detailPath != "" {
		change.Details = []tools.FileChangeDetail{{
			FilePath:     detailPath,
			Action:       resolveFileChangeAction(toolName),
			LinesAdded:   linesAdded,
			LinesRemoved: linesRemoved,
		}}
	}
	return change
}

func normalizeResolvedPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return ""
	}
	return filepath.Clean(path)
}

func resolveFileChangeAction(toolName string) string {
	switch toolName {
	case "delete_file":
		return "deleted"
	case "write_file", "str_replace":
		return "modified"
	default:
		return ""
	}
}
