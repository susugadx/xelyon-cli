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

func isDigitOrRange(s string) bool {
	dashSeen := false
	for i, c := range s {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && !dashSeen && i > 0 && i < len(s)-1 {
			dashSeen = true
			continue
		}
		return false
	}
	return len(s) > 0
}
