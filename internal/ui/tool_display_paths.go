package ui

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func countPathsArg(paths string) int {
	return len(readFilePathsArg(paths))
}

func readFileDisplayTarget(args map[string]string) string {
	paths := readFileArgsPaths(args)
	switch len(paths) {
	case 0:
		return ""
	case 1:
		return paths[0]
	default:
		return formatMultiplePathNames(paths)
	}
}

func readFileArgsPaths(args map[string]string) []string {
	if paths := readFilePathsArg(args["paths"]); len(paths) > 0 {
		return paths
	}
	if path := strings.TrimSpace(args["path"]); path != "" {
		return []string{path}
	}
	return nil
}

func readFilePathsArg(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var paths []string
	if err := json.Unmarshal([]byte(raw), &paths); err != nil {
		return nil
	}
	return paths
}

// formatMultiplePathNames returns a short summary for multiple paths.
func formatMultiplePathNames(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	names := make([]string, len(paths))
	for i, path := range paths {
		names[i] = smartShortPath(path)
	}

	seen := make(map[string][]int)
	for i, name := range names {
		seen[name] = append(seen[name], i)
	}
	for _, indices := range seen {
		if len(indices) <= 1 {
			continue
		}
		for _, idx := range indices {
			names[idx] = shortDirPath(paths[idx])
		}
	}

	const maxDisplay = 5
	if len(names) > maxDisplay {
		display := strings.Join(names[:maxDisplay], ", ")
		return fmt.Sprintf("%s ... +%d more", display, len(names)-maxDisplay)
	}
	return strings.Join(names, ", ")
}

// smartShortPath returns the base name of the path.
func smartShortPath(path string) string {
	return filepath.Base(stripReadFilePathRange(path))
}

// shortDirPath returns the parent directory and base name of the path.
func shortDirPath(path string) string {
	path = stripReadFilePathRange(path)
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	parent := filepath.Base(dir)
	if parent == "." || parent == "/" {
		return base
	}
	return parent + "/" + base
}

func stripReadFilePathRange(path string) string {
	if idx := strings.LastIndex(path, ":"); idx > 0 {
		suffix := path[idx+1:]
		if _, err := strconv.Atoi(strings.Split(suffix, "-")[0]); err == nil {
			return path[:idx]
		}
	}
	return path
}

func firstNonEmpty(args map[string]string, order ...string) string {
	for _, key := range order {
		if value := strings.TrimSpace(args[key]); value != "" {
			return value
		}
	}
	return ""
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
