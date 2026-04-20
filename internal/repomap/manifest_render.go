package repomap

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func (pm *ProjectMap) collectTopLevelEntries() ([]string, []string) {
	dirSet := make(map[string]struct{})
	fileSet := make(map[string]struct{})

	for _, file := range pm.Files {
		if file == nil || file.Path == "" {
			continue
		}
		cleanPath := filepath.ToSlash(file.Path)
		parts := strings.Split(cleanPath, "/")
		if len(parts) <= 1 {
			fileSet[cleanPath] = struct{}{}
			continue
		}
		dirSet[parts[0]] = struct{}{}
	}

	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir+"/")
	}
	sort.Strings(dirs)

	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	sort.Strings(files)

	return dirs, files
}

func (pm *ProjectMap) collectPriorityFiles(prioritizedPaths []string) []string {
	if len(prioritizedPaths) == 0 {
		return nil
	}

	available := make(map[string]struct{}, len(pm.Files))
	for _, file := range pm.Files {
		if file == nil || file.Path == "" {
			continue
		}
		available[file.Path] = struct{}{}
	}

	var priority []string
	for _, candidate := range prioritizedPaths {
		candidate = normalizeIgnoreDir(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := available[candidate]; ok {
			priority = append(priority, candidate)
			continue
		}
		for path := range available {
			if strings.HasPrefix(path, candidate+"/") {
				priority = append(priority, path)
			}
		}
	}

	return dedupeStrings(priority)
}

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

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeIgnoreDir(dir string) string {
	dir = filepath.ToSlash(strings.TrimSpace(dir))
	dir = strings.Trim(dir, "/")
	return dir
}
