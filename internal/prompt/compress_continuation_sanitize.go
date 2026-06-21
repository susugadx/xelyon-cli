package prompt

import "github.com/susugadx/xelyon-cli/internal/taskstate"

func summaryContinuationSingleLine(value string) string {
	return taskstate.FormatSnapshotExcerpt(value, 0)
}
