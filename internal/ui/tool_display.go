package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ToolDisplayInfo はツール実行結果の表示情報を保持する。
type ToolDisplayInfo struct {
	ToolName string
	Args     map[string]string
	Result   string
	Error    bool
}

var (
	searchSummaryPattern   = regexp.MustCompile(`Found (\d+) match(?:\(es\)|es?) in (\d+) file(?:\(s\)|s?)`)
	outlineSummaryPattern  = regexp.MustCompile(`\((\d+) lines total(?:\.[^)]*)?\)`)
	strReplaceRangePattern = regexp.MustCompile(`lines (\d+)-(\d+)`)
	strReplaceEditsPattern = regexp.MustCompile(`Successfully applied (\d+) edits`)
	writeFileLinesPattern  = regexp.MustCompile(`Successfully wrote \d+ bytes \((\d+) lines\) to `)
	leadingLineNumPattern  = regexp.MustCompile(`^(\d+): `)
)

// FormatToolLine はツール実行の1行サマリーを返す。
func FormatToolLine(info ToolDisplayInfo) string {
	trimmed := strings.TrimSpace(info.Result)
	blueName := Blue.Sprint(info.ToolName)
	if isToolDisplayError(info, trimmed) {
		target := toolTarget(info)
		if target == "" {
			return fmt.Sprintf("❌ %s → %s", blueName, firstLine(trimmed))
		}
		return fmt.Sprintf("❌ %s: %s → %s", blueName, target, firstLine(trimmed))
	}

	summary := formatToolSummary(info, trimmed)
	if summary == "" {
		return fmt.Sprintf("%s %s", toolIcon(info.ToolName), blueName)
	}
	return fmt.Sprintf("%s %s: %s", toolIcon(info.ToolName), blueName, summary)
}

// PrintParallelGroupStartToWriter は並列実行グループの開始行を指定 writer に表示する。
func PrintParallelGroupStartToWriter(w io.Writer, count int) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "┌ Parallel (%d calls)\n", count)
}

// PrintParallelGroupLineToWriter は並列実行グループ内の1行を指定 writer に表示する。
func PrintParallelGroupLineToWriter(w io.Writer, line string) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "│ %s\n", line)
}

// PrintParallelGroupEndToWriter は並列実行グループの終了行を指定 writer に表示する。
func PrintParallelGroupEndToWriter(w io.Writer, summary string) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "└ %s\n", summary)
}

func isToolDisplayError(info ToolDisplayInfo, trimmed string) bool {
	if info.Error {
		return true
	}
	return strings.HasPrefix(trimmed, "Error:") ||
		strings.HasPrefix(trimmed, "Unknown tool:") ||
		strings.HasPrefix(trimmed, "Cancelled")
}

func toolIcon(toolName string) string {
	switch {
	case strings.HasPrefix(toolName, "git_"):
		return "📦"
	case strings.HasPrefix(toolName, "mcp_"):
		return "🔌"
	}

	switch toolName {
	case "read_file", "read_files":
		return "📄"
	case "write_file":
		return "📝"
	case "str_replace":
		return "✏️"
	case "bash":
		return "⚙️"
	case "search_code":
		return "🔍"
	case "list_dir":
		return "🗂️"
	case "delete_file":
		return "🗑️"
	case "copy_file":
		return "📋"
	case "web_search":
		return "🌐"
	case "spawn_agent":
		return "🚀"
	case "wait_agent":
		return "⏳"
	case "lint":
		return "🔎"
	case "format":
		return "🎨"
	default:
		return "🔧"
	}
}

func formatToolSummary(info ToolDisplayInfo, trimmed string) string {
	switch {
	case info.ToolName == "read_file":
		return formatReadFileSummary(info.Args, trimmed)
	case info.ToolName == "read_files":
		return formatReadFilesSummary(info.Args, trimmed)
	case info.ToolName == "search_code":
		return formatSearchCodeSummary(info.Args, trimmed)
	case info.ToolName == "bash":
		return truncateText(info.Args["command"], 60)
	case info.ToolName == "str_replace":
		return formatStrReplaceSummary(info.Args, trimmed)
	case info.ToolName == "list_dir":
		return defaultPath(info.Args["path"])
	case info.ToolName == "write_file":
		return formatWriteFileSummary(info.Args, trimmed)
	case info.ToolName == "copy_file":
		return formatCopyFileSummary(info.Args)
	case info.ToolName == "delete_file":
		return defaultPath(info.Args["path"])
	case info.ToolName == "web_search":
		if q := strings.TrimSpace(info.Args["query"]); q != "" {
			return fmt.Sprintf("%q", q)
		}
	case info.ToolName == "spawn_agent":
		return truncateText(info.Args["message"], 60)
	case info.ToolName == "wait_agent":
		return formatWaitAgentSummary(info.Args["ids"])
	case info.ToolName == "lint":
		if path := defaultPath(info.Args["path"]); path != "" {
			if info.Args["auto_fix"] == "true" {
				return path + " (auto-fix)"
			}
			return path
		}
	case info.ToolName == "format":
		if path := defaultPath(info.Args["path"]); path != "" {
			return path
		}
	case strings.HasPrefix(info.ToolName, "git_"):
		if summary := formatGitSummary(info.Args); summary != "" {
			return summary
		}
	}

	if target := toolTarget(info); target != "" {
		return target
	}
	return firstLine(trimmed)
}

func toolTarget(info ToolDisplayInfo) string {
	switch {
	case info.ToolName == "search_code":
		pattern := strings.TrimSpace(info.Args["pattern"])
		if pattern == "" {
			return ""
		}
		target := fmt.Sprintf("%q", pattern)
		if path := strings.TrimSpace(info.Args["path"]); path != "" {
			target += " in " + path
		}
		return target
	case info.ToolName == "read_file":
		return readFileDisplayTarget(info.Args)
	case info.ToolName == "write_file", info.ToolName == "str_replace", info.ToolName == "delete_file":
		return strings.TrimSpace(info.Args["path"])
	case info.ToolName == "copy_file":
		src := strings.TrimSpace(info.Args["src"])
		dest := strings.TrimSpace(info.Args["dest"])
		switch {
		case src != "" && dest != "":
			return src + " -> " + dest
		case src != "":
			return src
		default:
			return dest
		}
	case info.ToolName == "bash":
		return truncateText(info.Args["command"], 60)
	case info.ToolName == "list_dir":
		return defaultPath(info.Args["path"])
	case info.ToolName == "web_search":
		if q := strings.TrimSpace(info.Args["query"]); q != "" {
			return fmt.Sprintf("%q", q)
		}
	case strings.HasPrefix(info.ToolName, "git_"):
		return formatGitSummary(info.Args)
	}

	return firstNonEmpty(info.Args, sortedKeys(info.Args)...)
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

func formatWaitAgentSummary(rawIDs string) string {
	count := countPathsArg(rawIDs)
	switch count {
	case 0:
		return "0 agents"
	case 1:
		return "1 agent"
	default:
		return fmt.Sprintf("%d agents", count)
	}
}

func formatStrReplaceSummary(args map[string]string, result string) string {
	path := strings.TrimSpace(args["path"])

	if matches := strReplaceEditsPattern.FindStringSubmatch(result); len(matches) == 2 {
		return joinTargetDetails(path, fmt.Sprintf("%s edits", matches[1]))
	}

	if matches := strReplaceRangePattern.FindStringSubmatch(result); len(matches) == 3 {
		startLine := matches[1]
		endLine := matches[2]
		if startLine == endLine {
			return fmt.Sprintf("%s:%s", path, startLine)
		}
		return fmt.Sprintf("%s:%s-%s", path, startLine, endLine)
	}

	if path != "" {
		return path
	}
	return firstLine(result)
}

func formatWriteFileSummary(args map[string]string, result string) string {
	path := strings.TrimSpace(args["path"])
	if matches := writeFileLinesPattern.FindStringSubmatch(result); len(matches) == 2 {
		return joinTargetDetails(path, fmt.Sprintf("%s lines", matches[1]))
	}
	return path
}

func formatCopyFileSummary(args map[string]string) string {
	src := strings.TrimSpace(args["src"])
	dest := strings.TrimSpace(args["dest"])
	if src != "" && dest != "" {
		return src + " -> " + dest
	}
	return firstNonEmpty(args, "src", "dest")
}

func formatGitSummary(args map[string]string) string {
	return firstNonEmpty(args, "path", "branch", "message", "commit", "target", "name")
}

func joinTargetDetails(target string, details ...string) string {
	var filtered []string
	for _, detail := range details {
		detail = strings.TrimSpace(detail)
		if detail != "" {
			filtered = append(filtered, detail)
		}
	}
	if len(filtered) == 0 {
		return target
	}
	if target == "" {
		return strings.Join(filtered, ", ")
	}
	return fmt.Sprintf("%s (%s)", target, strings.Join(filtered, ", "))
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func truncateText(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func defaultPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "."
	}
	return path
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

// FormatParallelElapsed は並列実行の経過時間を短い文字列で返す。
func FormatParallelElapsed(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < 10*time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
}
