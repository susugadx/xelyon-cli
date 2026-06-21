package uitoolview

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func formatGatherContextSummary(args map[string]string) string {
	query := strings.TrimSpace(args["query"])
	if query == "" {
		return ""
	}
	target := fmt.Sprintf("%q", query)
	if path := strings.TrimSpace(args["path"]); path != "" {
		target += " in " + path
	}
	if filter := strings.TrimSpace(args["file_filter"]); filter != "" {
		target += " [" + filter + "]"
	}
	return target
}

func formatReadFileSummary(args map[string]string, result string) string {
	paths := readFileArgsPaths(args)
	if len(paths) > 1 {
		return formatMultiplePathNames(paths)
	}
	path := ""
	if len(paths) == 1 {
		path = paths[0]
	}

	if matches := outlineSummaryPattern.FindStringSubmatch(result); len(matches) == 2 {
		if total, err := strconv.Atoi(matches[1]); err == nil {
			return joinTargetDetails(path, fmt.Sprintf("outline of %d lines", total))
		}
	}

	startLine, endLine, ok := lineRangeFromResult(result)
	if ok {
		if startLine == 1 {
			return joinTargetDetails(path, fmt.Sprintf("%d lines", endLine))
		}
		return joinTargetDetails(path, fmt.Sprintf("lines %d-%d", startLine, endLine))
	}

	if header := firstLine(result); header != "" && strings.HasPrefix(header, path+" (") {
		if detail := strings.TrimPrefix(header, path+" "); detail != header {
			return path + " " + detail
		}
	}

	return path
}

func formatReadFilesSummary(args map[string]string, result string) string {
	count := strings.Count(result, "📄 File: ")
	if count == 0 {
		count = countPathsArg(args["paths"])
	}
	if count > 0 {
		if names := readFilePathsArg(args["paths"]); len(names) > 0 {
			return formatMultiplePathNames(names)
		}
		return fmt.Sprintf("%d files", count)
	}
	return "multiple files"
}

func formatSearchCodeSummary(args map[string]string, result string) string {
	target := toolTarget(ToolDisplayInfo{ToolName: "search_code", Args: args})
	if totalMatches, totalFiles, ok := summarizeMultiPatternSearchResult(result); ok {
		return fmt.Sprintf("%s → %d matches, %d files", target, totalMatches, totalFiles)
	}
	for _, line := range strings.Split(result, "\n") {
		matches := searchSummaryPattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		return fmt.Sprintf("%s → %s matches, %s files", target, matches[1], matches[2])
	}
	if strings.Contains(result, "No matches found") {
		return target + " → No matches found"
	}
	return target
}

func summarizeMultiPatternSearchResult(result string) (int, int, bool) {
	matchKeys := make(map[string]struct{})
	files := make(map[string]struct{})
	sawMultiSection := false
	currentFile := ""

	for _, line := range strings.Split(result, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "━━ Pattern ") || strings.HasPrefix(trimmed, "━━ Symbol Bundle: ") {
			sawMultiSection = true
			currentFile = ""
		}
		if fileHeader := searchFileHeaderPattern.FindStringSubmatch(trimmed); len(fileHeader) == 2 {
			currentFile = normalizeSearchSummaryPath(fileHeader[1])
			files[currentFile] = struct{}{}
			continue
		}
		if symbolHeader := searchSymbolHeaderPattern.FindStringSubmatch(trimmed); len(symbolHeader) == 3 {
			currentFile = normalizeSearchSummaryPath(symbolHeader[2])
			files[currentFile] = struct{}{}
			matchKeys[currentFile+":"+symbolHeader[1]] = struct{}{}
			continue
		}
		if bundleItem := searchBundleItemPattern.FindStringSubmatch(trimmed); len(bundleItem) == 3 {
			currentFile = normalizeSearchSummaryPath(bundleItem[1])
			files[currentFile] = struct{}{}
			matchKeys[currentFile+":"+bundleItem[2]] = struct{}{}
			continue
		}
		if formattedLine := searchFormattedMatchLinePattern.FindStringSubmatch(trimmed); len(formattedLine) == 2 && currentFile != "" {
			matchKeys[currentFile+":"+formattedLine[1]] = struct{}{}
			continue
		}
		if lineNum := leadingLineNumPattern.FindStringSubmatch(trimmed); len(lineNum) == 2 && currentFile != "" {
			matchKeys[currentFile+":"+lineNum[1]] = struct{}{}
		}
	}

	if !sawMultiSection || len(matchKeys) == 0 {
		return 0, 0, false
	}
	return len(matchKeys), len(files), true
}

func normalizeSearchSummaryPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.ReplaceAll(path, "\\", "/")
	return filepath.Clean(path)
}

func lineRangeFromResult(result string) (int, int, bool) {
	startLine := 0
	endLine := 0
	scanner := bufio.NewScanner(strings.NewReader(result))
	for scanner.Scan() {
		line := scanner.Text()
		matches := leadingLineNumPattern.FindStringSubmatch(line)
		if len(matches) != 2 {
			continue
		}
		n, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if startLine == 0 {
			startLine = n
		}
		endLine = n
	}
	return startLine, endLine, startLine > 0 && endLine >= startLine
}
