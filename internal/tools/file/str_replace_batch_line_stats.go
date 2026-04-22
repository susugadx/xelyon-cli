package file

import (
	"strings"
)

func resolveBatchDiffLineStats(oldContent, newContent string) (added, removed int, exact bool) {
	return resolveBatchDiffLineStatsWithTuning(oldContent, newContent, resolveMyersDiffTuning())
}

func resolveBatchDiffLineStatsWithPolicy(oldContent, newContent string, policy batchDiffLineStatsPolicy) (added, removed int, exact bool) {
	if !policy.resolveExact {
		return 0, 0, false
	}
	return resolveBatchDiffLineStatsWithTuning(oldContent, newContent, policy.tuning)
}

func resolveBatchDiffLineStatsWithTuning(oldContent, newContent string, tuning myersDiffTuning) (added, removed int, exact bool) {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")
	trimmedOldLines, trimmedNewLines := trimSharedLineEdges(oldLines, newLines)
	if len(trimmedOldLines) == 0 && len(trimmedNewLines) == 0 {
		return 0, 0, true
	}
	if len(trimmedOldLines) == 0 {
		return len(trimmedNewLines), 0, true
	}
	if len(trimmedNewLines) == 0 {
		return 0, len(trimmedOldLines), true
	}
	return tryCountLineDiffWithMyers(trimmedOldLines, trimmedNewLines, resolveDynamicMyersStepLimit(len(trimmedOldLines), len(trimmedNewLines), tuning), tuning.lineSpanLimit)
}

func tryCountLineDiffWithMyers(oldLines, newLines []string, stepLimit, lineSpanLimit int) (added, removed int, ok bool) {
	n := len(oldLines)
	m := len(newLines)
	if lineSpanLimit > 0 && n+m > lineSpanLimit {
		return 0, 0, false
	}

	distance, ok := shortestEditDistanceMyersWithLimit(oldLines, newLines, stepLimit)
	if !ok {
		return 0, 0, false
	}

	removed = (distance - m + n) / 2
	added = (distance + m - n) / 2
	return added, removed, true
}
