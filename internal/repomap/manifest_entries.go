package repomap

import (
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
		candidate = normalizeManifestCandidatePath(candidate)
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

func normalizeManifestCandidatePath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	path = strings.Trim(path, "/")
	return path
}
