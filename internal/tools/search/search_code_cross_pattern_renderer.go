package search

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func renderCrossPatternIndex(fileMap map[string]*crossPatternIndexEntry, order []string, sections crossPatternIndexSections, reg *locator.Registry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n━━ File Index (%d unique files) ━━\n", len(order))
	writeCrossPatternIndexGroup(&sb, "Impl", sections.implKeys, fileMap, reg)
	writeCrossPatternIndexGroup(&sb, "Test", sections.testKeys, fileMap, reg)
	writeCrossPatternIndexGroup(&sb, "Config", sections.configKeys, fileMap, reg)
	return sb.String()
}

func writeCrossPatternIndexGroup(sb *strings.Builder, label string, keys []string, fileMap map[string]*crossPatternIndexEntry, reg *locator.Registry) {
	if len(keys) == 0 {
		return
	}
	fmt.Fprintf(sb, "%s:\n", label)
	for _, key := range keys {
		entry := fileMap[key]
		line := formatCrossPatternIndexEntryLine(entry)
		if reg != nil {
			id := reg.Register(newSearchLocator(entry.ref.DisplayPath, entry.ref.ResolvedPath, 0, 0, ""))
			line += " " + id
		}
		fmt.Fprintf(sb, "%s\n", line)
	}
}

func formatCrossPatternIndexEntryLine(entry *crossPatternIndexEntry) string {
	path := entry.ref.DisplayPath
	if entry.patternCount > 1 {
		return fmt.Sprintf("  %s (★%d patterns)", path, entry.patternCount)
	}
	return fmt.Sprintf("  %s", path)
}
