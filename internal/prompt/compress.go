package prompt

import (
	"fmt"
	"regexp"
	"strings"
)

// Message represents a conversation message for summary generation.
// This is a simplified struct to avoid importing api package.
type Message struct {
	Role    string
	Content string
}

const (
	toolInspectMaxLen = 200
	toolPreviewMaxLen = 100
)

var (
	toolResultHeaderPattern = regexp.MustCompile(`^\[Tool Result for [^\]]+\]\s*\n?`)
	searchMatchesPattern    = regexp.MustCompile(`Found\s+(\d+)\s+match`)
)

// BuildSummaryPrompt はサマリー生成用のプロンプトを構築する。
// truncateLen は通常メッセージの最大長（tool は先頭情報を優先して別ルールで短縮）。
func BuildSummaryPrompt(messages []Message, truncateLen int) string {
	var sb strings.Builder

	sb.WriteString("Summarize this conversation into a concise continuation context.\n\n")
	sb.WriteString("Include:\n")
	sb.WriteString("- Current task and progress status\n")
	sb.WriteString("- Key decisions and their rationale\n")
	sb.WriteString("- Files created/modified and what changed\n")
	sb.WriteString("- Remaining work (if any)\n\n")
	sb.WriteString("Exclude:\n")
	sb.WriteString("- Failed attempts and error messages unless they are still unresolved\n")
	sb.WriteString("- Tool outputs that are no longer relevant\n")
	sb.WriteString("- Exploratory searches that did not affect the final direction\n\n")
	sb.WriteString("Output as bullet points (5-10 items).\n")
	sb.WriteString("Focus on what the next assistant turn needs to know.\n")
	sb.WriteString("Respond in the same language as the conversation.\n")
	sb.WriteString("Maximum 500 words.\n\n")
	sb.WriteString("---\n\n")

	for _, msg := range messages {
		// systemメッセージはスキップ
		if msg.Role == "system" {
			continue
		}

		switch msg.Role {
		case "assistant":
			fmt.Fprintf(&sb, "[Assistant]\n%s\n\n", truncateSummaryContent(msg.Content, truncateLen))
		case "tool":
			fmt.Fprintf(&sb, "%s\n\n", formatToolSummary(msg.Content, truncateLen))
		default:
			fmt.Fprintf(&sb, "[User]\n%s\n\n", truncateSummaryContent(msg.Content, truncateLen))
		}
	}

	sb.WriteString("---\n\n")
	sb.WriteString("Now provide the summary.")

	return sb.String()
}

func truncateSummaryContent(content string, truncateLen int) string {
	if truncateLen > 0 && len(content) > truncateLen {
		return content[:truncateLen] + "..."
	}
	return content
}

func formatToolSummary(content string, truncateLen int) string {
	body := extractToolBody(content)
	if body == "" {
		return "[Tool]"
	}

	toolLimit := toolSummaryLimit(truncateLen)
	preview, truncated := limitToolPreview(body, toolLimit)

	if strings.HasPrefix(preview, "Error:") {
		reason := strings.TrimSpace(strings.TrimPrefix(preview, "Error:"))
		reason = normalizeToolText(firstLine(reason))
		reason = summarizeToolText(reason, toolLimit, truncated)
		if reason == "" {
			return appendToolTruncatedMarker("[Tool: failed]", body)
		}
		return appendToolTruncatedMarker("[Tool: failed] "+reason, body)
	}

	if matches := searchMatchesPattern.FindStringSubmatch(preview); matches != nil {
		summary := fmt.Sprintf("[Tool: search] %s matches in %d files", matches[1], countSearchFiles(body))
		return appendToolTruncatedMarker(summary, body)
	}

	if path := extractPathLikePreview(preview); path != "" {
		summary := "[Tool: read] " + summarizeToolText(path, toolLimit, truncated)
		return appendToolTruncatedMarker(summary, body)
	}

	genericLimit := toolPreviewMaxLen
	if toolLimit > 0 && toolLimit < genericLimit {
		genericLimit = toolLimit
	}
	summary := summarizeToolText(normalizeToolText(preview), genericLimit, truncated)
	if summary == "" {
		return "[Tool]"
	}
	return appendToolTruncatedMarker("[Tool] "+summary, body)
}

func extractToolBody(content string) string {
	content = strings.TrimSpace(content)
	content = toolResultHeaderPattern.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

func toolSummaryLimit(truncateLen int) int {
	if truncateLen > 0 && truncateLen < toolInspectMaxLen {
		return truncateLen
	}
	return toolInspectMaxLen
}

func limitToolPreview(content string, limit int) (string, bool) {
	if limit > 0 && len(content) > limit {
		return content[:limit], true
	}
	return content, false
}

func firstLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func normalizeToolText(content string) string {
	if content == "" {
		return ""
	}
	return strings.Join(strings.Fields(content), " ")
}

func summarizeToolText(content string, limit int, truncated bool) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if limit > 0 && len(content) > limit {
		return content[:limit] + "..."
	}
	if truncated {
		return content + "..."
	}
	return content
}

func appendToolTruncatedMarker(summary, body string) string {
	if strings.Contains(body, "truncated") && !strings.Contains(strings.ToLower(summary), "truncated") {
		return summary + " (truncated)"
	}
	return summary
}

func countSearchFiles(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "📄 ") {
			count++
		}
	}
	return count
}

func extractPathLikePreview(preview string) string {
	seen := 0
	for _, rawLine := range strings.Split(preview, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		seen++
		candidate := strings.TrimSpace(strings.TrimPrefix(line, "📄 "))
		if idx := strings.Index(candidate, " ("); idx > 0 {
			candidate = candidate[:idx]
		}
		candidate = strings.TrimSpace(candidate)
		if isPathLike(candidate) {
			return candidate
		}
		if seen >= 3 {
			break
		}
	}
	return ""
}

func isPathLike(candidate string) bool {
	if candidate == "" || strings.Contains(candidate, "://") {
		return false
	}
	if strings.ContainsAny(candidate, " \t") {
		return false
	}
	if strings.HasPrefix(candidate, "/") ||
		strings.HasPrefix(candidate, "./") ||
		strings.HasPrefix(candidate, "../") ||
		strings.HasPrefix(candidate, "~/") {
		return true
	}

	if idx := strings.LastIndex(candidate, ":"); idx > 0 && idx < len(candidate)-1 {
		suffix := candidate[idx+1:]
		if isDigits(suffix) {
			candidate = candidate[:idx]
		}
	}

	if strings.ContainsAny(candidate, "/\\") {
		return true
	}

	dot := strings.LastIndex(candidate, ".")
	if dot <= 0 || dot == len(candidate)-1 {
		return false
	}
	ext := candidate[dot+1:]
	if len(ext) > 8 || !isAlphaNum(ext) {
		return false
	}
	return true
}

func isDigits(content string) bool {
	if content == "" {
		return false
	}
	for _, ch := range content {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isAlphaNum(content string) bool {
	if content == "" {
		return false
	}
	for _, ch := range content {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') {
			return false
		}
	}
	return true
}
