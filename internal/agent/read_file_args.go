package agent

import (
	"encoding/json"
	"strings"
)

func readFilePathsFromArgs(args map[string]string) []string {
	if args == nil {
		return nil
	}
	if rawPaths := strings.TrimSpace(args["paths"]); rawPaths != "" {
		var paths []string
		if err := json.Unmarshal([]byte(rawPaths), &paths); err != nil {
			return nil
		}
		out := make([]string, 0, len(paths))
		for _, path := range paths {
			path = strings.TrimSpace(path)
			if path != "" {
				out = append(out, path)
			}
		}
		return out
	}
	if path := strings.TrimSpace(args["path"]); path != "" {
		return []string{path}
	}
	return nil
}

func readFileEntryHasRange(entry string) bool {
	lastColon := strings.LastIndex(entry, ":")
	if lastColon < 0 {
		return false
	}
	suffix := entry[lastColon+1:]
	return suffix != "" && isDigitOrRange(suffix)
}

func readFileBasePath(entry string) string {
	if !readFileEntryHasRange(entry) {
		return entry
	}
	lastColon := strings.LastIndex(entry, ":")
	if lastColon < 0 {
		return entry
	}
	return entry[:lastColon]
}

func readFileHasExplicitRange(args map[string]string) bool {
	if args == nil {
		return false
	}
	if args["start_line"] != "" || args["end_line"] != "" {
		return true
	}
	for _, path := range readFilePathsFromArgs(args) {
		if readFileEntryHasRange(path) {
			return true
		}
	}
	return false
}

func readFileTrackerKey(args map[string]string) string {
	paths := readFilePathsFromArgs(args)
	if len(paths) != 1 {
		return ""
	}
	return readFileBasePath(paths[0])
}
