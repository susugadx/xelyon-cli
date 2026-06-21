package listtool

import (
	"fmt"
	"strings"
)

func renderListDirSummary(absPath string, depth int, section *listDirSection) []string {
	lines := []string{
		fmt.Sprintf("📂 %s", absPath),
		fmt.Sprintf("summary: depth=%d, dirs=%d, files=%d", depth, section.totalDirs, section.totalFiles),
	}
	if section.readErr != nil {
		return append(lines, "Error: failed to read directory")
	}
	appendSectionSummary(&lines, "", section, depth > 1)
	return lines
}

func appendSectionSummary(lines *[]string, indent string, section *listDirSection, includeSubtrees bool) {
	if len(section.dirs) > 0 {
		*lines = append(*lines, indent+"dirs: "+formatListDirNames(section.dirs, section.moreDirs))
	}
	if len(section.files) > 0 {
		*lines = append(*lines, indent+"files: "+formatListDirFiles(section.files, section.moreFiles))
	}
	if !includeSubtrees || len(section.subtrees) == 0 {
		return
	}

	subtreeSummary := fmt.Sprintf("subtrees: %d shown", len(section.subtrees))
	if section.moreSubtree > 0 {
		subtreeSummary = fmt.Sprintf("subtrees: %d shown (+%d more)", len(section.subtrees), section.moreSubtree)
	}
	*lines = append(*lines, indent+subtreeSummary)
	for _, child := range section.subtrees {
		renderListDirSection(lines, indent, child)
	}
}

func renderListDirSection(lines *[]string, indent string, section *listDirSection) {
	prefix := indent + "- "
	if section.readErr != nil {
		*lines = append(*lines, fmt.Sprintf("%s%s -> [error reading directory]", prefix, section.relPath))
		return
	}

	*lines = append(*lines, fmt.Sprintf("%s%s -> dirs=%d, files=%d", prefix, section.relPath, section.totalDirs, section.totalFiles))
	appendSectionSummary(lines, indent+"  ", section, len(section.subtrees) > 0)
}

func formatListDirNames(names []string, more int) string {
	if more == 0 {
		return strings.Join(names, ", ")
	}
	return strings.Join(names, ", ") + fmt.Sprintf(", (+%d more)", more)
}

func formatListDirFiles(files []listDirFileSummary, more int) string {
	parts := make([]string, 0, len(files)+1)
	for _, file := range files {
		parts = append(parts, fmt.Sprintf("%s (%d bytes)", file.name, file.size))
	}
	if more > 0 {
		parts = append(parts, fmt.Sprintf("(+%d more)", more))
	}
	return strings.Join(parts, ", ")
}
