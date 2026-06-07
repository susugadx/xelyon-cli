package commandoutputs

import (
	"fmt"
	"strings"
)

func buildGitCompact(command, content string, family commandFamily) (Replacement, string, bool) {
	switch family {
	case commandFamilyGitStatus:
		return buildGitStatusCompact(command, content)
	case commandFamilyGitLog:
		return buildGitListCompact(command, content, "compact_git_log_command_output", "git_log", "git_log")
	case commandFamilyGitBranch:
		return buildGitListCompact(command, content, "compact_git_branch_command_output", "git_branch", "git_branch")
	case commandFamilyGitFileList:
		return buildGitListCompact(command, content, "compact_git_file_list_command_output", "git_file_list", "git_file_list")
	default:
		return Replacement{}, "git_output_unknown_skip", false
	}
}

func buildGitStatusCompact(command, content string) (Replacement, string, bool) {
	lines := outputLines(content)
	if len(lines) < largeGenericLineThreshold && len(content) < largeGenericByteThreshold {
		return Replacement{}, "git_status_not_large", false
	}
	categories := classifyGitStatusLines(lines)
	total := 0
	for _, entries := range categories {
		total += len(entries)
	}
	if total == 0 {
		return Replacement{}, "git_status_unparseable", false
	}
	parts := []string{fmt.Sprintf(
		"[compacted old git status output; command=\"%s\"; classifier=git_status; entries=%d; staged=%d; unstaged=%d; untracked=%d; conflicted=%d]",
		sanitizeCommandForHeader(command),
		total,
		len(categories["staged"]),
		len(categories["unstaged"]),
		len(categories["untracked"]),
		len(categories["conflicted"]),
	)}
	for _, name := range []string{"staged", "unstaged", "untracked", "conflicted", "other"} {
		entries := categories[name]
		if len(entries) == 0 {
			continue
		}
		parts = append(parts, name+":")
		parts = append(parts, listEntriesWithOmission(entries, compactListSideEntries)...)
	}
	text := strings.Join(parts, "\n")
	return replacement("compact_git_status_command_output", "git_status", "git_status", content, text), "", true
}

func buildGitListCompact(command, content, kind, reason, classifier string) (Replacement, string, bool) {
	lines := outputLines(content)
	if len(lines) < largeGenericLineThreshold && len(content) < largeGenericByteThreshold {
		return Replacement{}, reason + "_not_large", false
	}
	body, omitted := firstLastEntries(lines, compactListSideEntries)
	if omitted <= 0 {
		return Replacement{}, reason + "_not_large", false
	}
	header := fmt.Sprintf(
		"[compacted old git output; command=\"%s\"; classifier=%s; entries=%d]",
		sanitizeCommandForHeader(command),
		classifier,
		len(lines),
	)
	text := header + "\n" + strings.Join(body, "\n")
	return replacement(kind, reason, classifier, content, text), "", true
}

func isParseableGitDiff(content string) bool {
	return strings.Contains(content, "diff --git ") ||
		strings.Contains(content, "Binary files ") ||
		strings.Contains(content, "GIT binary patch")
}

func classifyGitStatusLines(lines []string) map[string][]string {
	categories := map[string][]string{
		"staged":     {},
		"unstaged":   {},
		"untracked":  {},
		"conflicted": {},
		"other":      {},
	}
	for _, rawLine := range lines {
		line := stripANSI(rawLine)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "On branch ") || strings.HasPrefix(trimmed, "Your branch ") {
			continue
		}
		category := classifyGitStatusLine(line, trimmed)
		categories[category] = append(categories[category], "  "+sanitizeHeaderValue(trimmed))
	}
	return categories
}

func classifyGitStatusLine(line, trimmed string) string {
	if strings.Contains(trimmed, "Untracked files:") {
		return "untracked"
	}
	if len(line) >= 2 {
		indexStatus := line[0]
		worktreeStatus := line[1]
		if indexStatus == '?' && worktreeStatus == '?' {
			return "untracked"
		}
		if isGitStatusConflict(indexStatus, worktreeStatus) {
			return "conflicted"
		}
		if indexStatus != ' ' && indexStatus != '?' {
			return "staged"
		}
		if worktreeStatus != ' ' && worktreeStatus != '?' {
			return "unstaged"
		}
	}
	if strings.HasPrefix(trimmed, "??") {
		return "untracked"
	}
	if strings.HasPrefix(trimmed, "UU") || strings.HasPrefix(trimmed, "AA") || strings.HasPrefix(trimmed, "DD") {
		return "conflicted"
	}
	return "other"
}

func isGitStatusConflict(indexStatus, worktreeStatus byte) bool {
	switch string([]byte{indexStatus, worktreeStatus}) {
	case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
		return true
	default:
		return false
	}
}
