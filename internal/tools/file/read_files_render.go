package file

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func renderReadFilesResults(results []readFileBatchResult, reg *locator.Registry) string {
	return renderReadExecutionSections(renderReadFilesSections(results, reg))
}

func renderReadFilesSections(results []readFileBatchResult, reg *locator.Registry) []ReadExecutionSection {
	sections := make([]ReadExecutionSection, 0, len(results))
	var sb strings.Builder
	for _, result := range results {
		sb.Reset()
		header := fmt.Sprintf("📄 File: %s", result.entry)
		if reg != nil {
			id := reg.Register(newReadResultLocatorForBatch(result))
			header += " " + id
		}
		fmt.Fprintf(&sb, "%s\n", header)
		sb.WriteString(result.result)
		failed := isRenderedReadFailure(result.result)
		sections = append(sections, ReadExecutionSection{
			Output:      sb.String(),
			Failed:      failed,
			Observation: readObservationForBatchResult(result, failed),
		})
	}
	return sections
}

func isRenderedReadFailure(result string) bool {
	trimmed := strings.TrimSpace(result)
	return strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "Error reading file:")
}
