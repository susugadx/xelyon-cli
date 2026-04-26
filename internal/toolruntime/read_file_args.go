package toolruntime

import (
	"encoding/json"
	"strings"
)

// ReadFilePathsFromArgs は read_file の path/paths 引数から有効な path 一覧を取り出す。
func ReadFilePathsFromArgs(args map[string]string) []string {
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

// ReadFileEntryHasRange は path entry が末尾に line range 指定を持つか返す。
func ReadFileEntryHasRange(entry string) bool {
	lastColon := strings.LastIndex(entry, ":")
	if lastColon < 0 {
		return false
	}
	suffix := entry[lastColon+1:]
	return suffix != "" && IsDigitOrRange(suffix)
}

// ReadFileHasExplicitRange は read_file 引数が明示的な range 指定を持つか返す。
func ReadFileHasExplicitRange(args map[string]string) bool {
	if args == nil {
		return false
	}
	if args["start_line"] != "" || args["end_line"] != "" {
		return true
	}
	for _, path := range ReadFilePathsFromArgs(args) {
		if ReadFileEntryHasRange(path) {
			return true
		}
	}
	return false
}

// IsDigitOrRange は文字列が行番号または行番号 range として解釈できるか返す。
func IsDigitOrRange(s string) bool {
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
