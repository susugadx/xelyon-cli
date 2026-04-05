package file

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func newFileChange(path, toolName, description string, linesAdded, linesRemoved int) *tools.FileChange {
	return &tools.FileChange{
		FilePath:     path,
		Timestamp:    common.GetCurrentTime(),
		Tool:         toolName,
		Description:  description,
		LinesAdded:   linesAdded,
		LinesRemoved: linesRemoved,
	}
}
