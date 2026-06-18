package readtool

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func readObservationForBatchResult(result readFileBatchResult, failed bool) *tools.RuntimeObservation {
	if failed {
		return nil
	}
	location := readObservationPath(result)
	if strings.TrimSpace(location.Path) == "" && strings.TrimSpace(location.ResolvedPath) == "" {
		return nil
	}

	observation := &tools.RuntimeObservation{
		TouchedFiles: []tools.ObservationPath{location},
	}
	if startLine, endLine, ok := readObservationLineRange(result); ok {
		observation.Evidence = append(observation.Evidence, tools.ObservationEvidence{
			Path:         location.Path,
			ResolvedPath: location.ResolvedPath,
			StartLine:    startLine,
			EndLine:      endLine,
			Excerpt:      firstReadObservationExcerpt(result.result),
		})
	}
	return observation
}

func readObservationPath(result readFileBatchResult) tools.ObservationPath {
	path := strings.TrimSpace(result.filePath)
	if path == "" {
		path = strings.TrimSpace(result.entry)
	}
	return tools.ObservationPath{
		Path:         path,
		ResolvedPath: strings.TrimSpace(result.resolvedPath),
	}
}

func readObservationLineRange(result readFileBatchResult) (int, int, bool) {
	startLine := result.startLine
	endLine := result.endLine
	if startLine <= 0 && endLine <= 0 {
		return 0, 0, false
	}
	if startLine <= 0 {
		startLine = endLine
	}
	if endLine <= 0 || endLine < startLine {
		endLine = startLine
	}
	return startLine, endLine, true
}

func firstReadObservationExcerpt(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 240 {
			return line[:240] + "..."
		}
		return line
	}
	return ""
}
