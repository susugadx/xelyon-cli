package uitoolview

import (
	"fmt"
	"strings"
)

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

func formatApplyPatchSummary(args map[string]string, result string) string {
	var added, modified, deleted int

	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Added: ") {
			added = countApplyPaths(strings.TrimPrefix(line, "Added: "))
		} else if strings.HasPrefix(line, "Modified: ") {
			modified = countApplyPaths(strings.TrimPrefix(line, "Modified: "))
		} else if strings.HasPrefix(line, "Deleted: ") {
			deleted = countApplyPaths(strings.TrimPrefix(line, "Deleted: "))
		}
	}

	total := added + modified + deleted
	if total == 0 {
		return "No files changed"
	}

	var parts []string
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}
	if added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", added))
	}
	if deleted > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", deleted))
	}

	return fmt.Sprintf("%d files (%s)", total, strings.Join(parts, ", "))
}

func countApplyPaths(pathsStr string) int {
	if pathsStr == "(none)" || pathsStr == "" {
		return 0
	}
	return len(strings.Split(pathsStr, ","))
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

func formatSpawnAgentSummary(message string) string {
	return truncateText(firstLine(message), 200)
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
