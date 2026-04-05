package file

import (
	"strconv"
	"strings"
)

// parsePath はパス文字列を解析し、ファイルパスと行範囲を返す
// "path" → path, 0, 0
// "path:10" → path, 10, 0
// "path:10-20" → path, 10, 20
func parsePath(entry string) (string, int, int) {
	lastColon := strings.LastIndex(entry, ":")
	if lastColon < 0 {
		return entry, 0, 0
	}

	suffix := entry[lastColon+1:]
	path := entry[:lastColon]

	if dashIdx := strings.Index(suffix, "-"); dashIdx >= 0 {
		startStr := suffix[:dashIdx]
		endStr := suffix[dashIdx+1:]
		start, err1 := strconv.Atoi(startStr)
		end, err2 := strconv.Atoi(endStr)
		if err1 == nil && err2 == nil && start > 0 && end > 0 {
			return path, start, end
		}
		return entry, 0, 0
	}

	start, err := strconv.Atoi(suffix)
	if err == nil && start > 0 {
		return path, start, 0
	}

	return entry, 0, 0
}
