package file

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func renderReadFilesResults(results []readFileBatchResult, reg *locator.Registry) string {
	var sb strings.Builder
	for i, result := range results {
		if i > 0 {
			sb.WriteString("\n")
		}
		header := fmt.Sprintf("📄 File: %s", result.entry)
		if reg != nil {
			id := reg.Register(locator.Location{
				FilePath: result.filePath,
				Line:     result.startLine,
				EndLine:  result.endLine,
			})
			header += " " + id
		}
		fmt.Fprintf(&sb, "%s\n", header)
		sb.WriteString(result.result)
	}
	return sb.String()
}
