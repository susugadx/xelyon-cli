package repomap

import (
	"fmt"
	"strings"
)

func writeManifestList(b *strings.Builder, title string, values []string, limit int, isDirectory bool) {
	if len(values) == 0 {
		return
	}
	b.WriteString(title)
	b.WriteString(":\n")
	shown := values
	if len(shown) > limit {
		shown = shown[:limit]
	}
	for _, value := range shown {
		if isDirectory && !strings.HasSuffix(value, "/") {
			value += "/"
		}
		b.WriteString("- ")
		b.WriteString(value)
		b.WriteByte('\n')
	}
	if len(values) > len(shown) {
		fmt.Fprintf(b, "- ... (+%d more)\n", len(values)-len(shown))
	}
}

func writeManifestChanges(b *strings.Builder, changes []GitChange, limit int) {
	if len(changes) == 0 || limit <= 0 {
		return
	}

	b.WriteString("Uncommitted changes:\n")
	shown := changes
	if len(shown) > limit {
		shown = shown[:limit]
	}
	for _, change := range shown {
		fmt.Fprintf(b, "- %s %s\n", change.Status, change.Path)
	}
	if len(changes) > len(shown) {
		fmt.Fprintf(b, "- ... (+%d more)\n", len(changes)-len(shown))
	}
}

func renderManifest(topDirs []string, dirLimit int, topFiles []string, fileLimit int, priorityFiles []string, priorityLimit int, changes []GitChange, changeLimit int) string {
	var b strings.Builder
	b.WriteString("## Project Map\n\n")

	if dirLimit > 0 {
		writeManifestList(&b, "Top-level directories", topDirs, dirLimit, true)
	}
	if fileLimit > 0 {
		writeManifestList(&b, "Top-level files", topFiles, fileLimit, false)
	}
	if priorityLimit > 0 {
		writeManifestList(&b, "Priority files", priorityFiles, priorityLimit, false)
	}
	if changeLimit > 0 {
		writeManifestChanges(&b, changes, changeLimit)
	}

	return strings.TrimRight(b.String(), "\n")
}

func (pm *ProjectMap) generateManifestFallback(fileCount, changeCount int) string {
	fallback := fmt.Sprintf("## Project Map\n\n- Project map omitted to stay within budget (%d files, %d changes)\n", fileCount, changeCount)
	if pm.fitsBudget(fallback) {
		return strings.TrimRight(fallback, "\n")
	}
	return ""
}
