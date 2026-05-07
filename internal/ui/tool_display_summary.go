package ui

import (
	"fmt"
	"strings"
)

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
	case "gather_context":
		return "🧭"
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
	case info.ToolName == "gather_context":
		return formatGatherContextSummary(info.Args)
	case info.ToolName == "read_file":
		return formatReadFileSummary(info.Args, trimmed)
	case info.ToolName == "read_files":
		return formatReadFilesSummary(info.Args, trimmed)
	case info.ToolName == "search_code":
		return formatSearchCodeSummary(info.Args, trimmed)
	case info.ToolName == "bash":
		return truncateText(info.Args["command"], 60)
	case info.ToolName == "apply_patch":
		return formatApplyPatchSummary(info.Args, trimmed)
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
		return formatSpawnAgentSummary(info.Args["message"])
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

// ToolTarget は ToolDisplayInfo から表示用 target を返す。
func ToolTarget(info ToolDisplayInfo) string {
	return toolTarget(info)
}

func toolTarget(info ToolDisplayInfo) string {
	switch {
	case info.ToolName == "gather_context":
		query := strings.TrimSpace(info.Args["query"])
		if query == "" {
			return ""
		}
		target := fmt.Sprintf("%q", query)
		if path := strings.TrimSpace(info.Args["path"]); path != "" {
			target += " in " + path
		}
		return target
	case info.ToolName == "search_code":
		pattern := strings.TrimSpace(info.Args["pattern"])
		if pattern == "" {
			return ""
		}
		target := fmt.Sprintf("%q", pattern)
		if path := strings.TrimSpace(info.Args["path"]); path != "" {
			target += " in " + path
		}
		if strings.EqualFold(strings.TrimSpace(info.Args["intent"]), "impact") {
			target += " (impact)"
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
