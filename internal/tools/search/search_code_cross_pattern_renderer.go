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
	writeCrossPatternIndexGroupLines(sb, buildCrossPatternIndexGroupLines(keys, fileMap, reg))
}

func buildCrossPatternIndexGroupLines(keys []string, fileMap map[string]*crossPatternIndexEntry, reg *locator.Registry) []string {
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		entry := fileMap[key]
		line := formatCrossPatternIndexEntryLine(entry)
		lines = append(lines, appendCrossPatternIndexLocator(line, entry, reg))
	}
	return lines
}

func writeCrossPatternIndexGroupLines(sb *strings.Builder, lines []string) {
	for _, line := range lines {
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

func appendCrossPatternIndexLocator(line string, entry *crossPatternIndexEntry, reg *locator.Registry) string {
	id := crossPatternIndexLocatorID(entry, reg)
	if id == "" {
		return line
	}
	return line + " " + id
}

func crossPatternIndexLocatorID(entry *crossPatternIndexEntry, reg *locator.Registry) string {
	if reg == nil {
		return ""
	}
	return reg.Register(newSearchLocator(entry.ref.DisplayPath, entry.ref.ResolvedPath, 0, 0, ""))
}
