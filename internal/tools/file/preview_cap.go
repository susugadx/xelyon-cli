package file

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	maxFullBodyPreviewBytes = 64 * 1024
)

type previewCapResult struct {
	lines            []ui.PatchPreviewLine
	totalLines       int
	truncated        bool
	truncatedByBytes bool
}

func buildCappedPatchPreviewLines(contentLines []string, lineType rune, maxPreviewLines int) previewCapResult {
	capacity := len(contentLines)
	if maxPreviewLines > 0 {
		capacity = min(capacity, maxPreviewLines)
	}
	result := previewCapResult{
		lines:      make([]ui.PatchPreviewLine, 0, capacity),
		totalLines: len(contentLines),
	}

	previewBytes := 0
	for i, line := range contentLines {
		lineBytes := len(line)
		if i > 0 {
			lineBytes++
		}
		if maxPreviewLines > 0 && len(result.lines) >= maxPreviewLines {
			result.truncated = true
			break
		}
		if previewBytes+lineBytes > maxFullBodyPreviewBytes {
			result.truncated = true
			result.truncatedByBytes = true
			break
		}
		result.lines = append(result.lines, ui.PatchPreviewLine{
			Type:    lineType,
			LineNum: i + 1,
			Text:    line,
		})
		previewBytes += lineBytes
	}

	return result
}

func resolvePreviewLineCap(cfg *config.Config) int {
	if cfg == nil {
		return 0
	}
	return cfg.Diff.MaxTotalLines
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
