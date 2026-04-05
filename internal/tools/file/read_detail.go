package file

import (
	"fmt"
	"strings"
)

type readDetailMode string

const (
	readDetailAuto    readDetailMode = "auto"
	readDetailCompact readDetailMode = "compact"
	readDetailFull    readDetailMode = "full"
	readDetailOutline readDetailMode = "outline"
)

func resolveReadDetail(rawDetail, rawFullBudget string) (readDetailMode, string) {
	detail := strings.TrimSpace(rawDetail)
	if detail == "" {
		return readDetailAuto, ""
	}

	mode := readDetailMode(strings.ToLower(detail))
	switch mode {
	case readDetailAuto, readDetailCompact, readDetailFull, readDetailOutline:
		return mode, ""
	default:
		return "", fmt.Sprintf(`Error: invalid detail %q (expected one of: auto, compact, full, outline)`, rawDetail)
	}
}

func (m readDetailMode) wholeFileOverride() bool {
	return m == readDetailFull || m == readDetailOutline
}

func resolveReadBudgetOverride(rawDetail, rawFullBudget string) int {
	if strings.TrimSpace(rawDetail) != "" {
		return 0
	}
	if strings.EqualFold(strings.TrimSpace(rawFullBudget), "true") {
		return DefaultFullLines
	}
	return 0
}
