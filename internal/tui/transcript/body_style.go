package transcript

import (
	"strings"
	"unicode"

	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// renderMessageBodyLines は role ごとの本文装飾を適用した transcript 行を生成する。
func renderMessageBodyLines(role string, content string) []string {
	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = NormalizeLine(lines[i])
	}
	if role == "assistant" {
		return renderAssistantBodyLines(lines)
	}
	return lines
}

func renderAssistantBodyLines(lines []string) []string {
	rendered := make([]string, len(lines))
	inCodeFence := false
	for i, line := range lines {
		rendered[i], inCodeFence = renderAssistantBodyLine(line, inCodeFence)
	}
	return rendered
}

func renderAssistantBodyLine(line string, inCodeFence bool) (string, bool) {
	if isMarkdownFenceLine(line) {
		return styleTranscriptLine(theme.Transcript.AssistantCodeFence, line), !inCodeFence
	}
	if inCodeFence {
		return styleTranscriptLine(theme.Transcript.AssistantCodeBlock, line), inCodeFence
	}
	if hasANSI(line) {
		return line, inCodeFence
	}
	if isMarkdownHeadingLine(line) {
		return styleTranscriptLine(theme.Transcript.AssistantHeading, line), inCodeFence
	}
	if styled, ok := styleAssistantAdmonitionLine(line); ok {
		return styled, inCodeFence
	}
	if styled, ok := styleAssistantQuoteLine(line); ok {
		return styled, inCodeFence
	}
	if styled, ok := styleAssistantListLine(line); ok {
		return styled, inCodeFence
	}
	if styled, ok := styleAssistantLeadingBoldLine(line); ok {
		return styled, inCodeFence
	}
	return styleAssistantText(line), inCodeFence
}

func hasANSI(line string) bool {
	return strings.ContainsRune(line, '\033')
}

func isMarkdownFenceLine(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func isMarkdownHeadingLine(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	hashes := 0
	for hashes < len(trimmed) && trimmed[hashes] == '#' {
		hashes++
	}
	return hashes > 0 && hashes <= 6 && len(trimmed) > hashes && unicode.IsSpace(rune(trimmed[hashes]))
}

func styleAssistantListLine(line string) (string, bool) {
	lead, rest := splitLeadingWhitespace(line)
	markerLen := unorderedListMarkerLen(rest)
	if markerLen == 0 {
		markerLen = orderedListMarkerLen(rest)
	}
	if markerLen == 0 {
		return "", false
	}
	return lead +
		styleTranscriptLine(theme.Transcript.AssistantListMarker, rest[:markerLen]) +
		styleAssistantText(rest[markerLen:]), true
}

func unorderedListMarkerLen(s string) int {
	if len(s) < 2 || !strings.ContainsRune("-*+", rune(s[0])) || !unicode.IsSpace(rune(s[1])) {
		return 0
	}
	if len(s) >= 6 && s[1] == ' ' && s[2] == '[' && (s[3] == ' ' || s[3] == 'x' || s[3] == 'X') && s[4] == ']' && unicode.IsSpace(rune(s[5])) {
		return 6
	}
	return 2
}

func orderedListMarkerLen(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i+1 >= len(s) {
		return 0
	}
	if (s[i] == '.' || s[i] == ')') && unicode.IsSpace(rune(s[i+1])) {
		return i + 2
	}
	return 0
}

func styleAssistantQuoteLine(line string) (string, bool) {
	lead, rest := splitLeadingWhitespace(line)
	if !strings.HasPrefix(rest, ">") {
		return "", false
	}
	return lead + styleInlineCodeSegments(rest, theme.Transcript.AssistantQuote), true
}

func styleAssistantAdmonitionLine(line string) (string, bool) {
	lead, rest := splitLeadingWhitespace(line)
	labelLen, style, ok := assistantAdmonitionLabel(rest)
	if !ok {
		return "", false
	}
	return lead +
		styleTranscriptLine(style, rest[:labelLen]) +
		styleAssistantText(rest[labelLen:]), true
}

func assistantAdmonitionLabel(s string) (int, string, bool) {
	labels := []struct {
		label string
		style string
	}{
		{label: "note:", style: theme.Transcript.AssistantNote},
		{label: "tip:", style: theme.Transcript.AssistantNote},
		{label: "important:", style: theme.Transcript.AssistantNote},
		{label: "warning:", style: theme.Transcript.AssistantWarning},
		{label: "caution:", style: theme.Transcript.AssistantWarning},
		{label: "error:", style: theme.Transcript.AssistantError},
	}

	lower := strings.ToLower(s)
	for _, candidate := range labels {
		if strings.HasPrefix(lower, candidate.label) {
			return len(candidate.label), candidate.style, true
		}
		boldLabel := "**" + candidate.label + "**"
		if strings.HasPrefix(lower, boldLabel) {
			return len(boldLabel), candidate.style, true
		}
	}
	return 0, "", false
}

func styleAssistantLeadingBoldLine(line string) (string, bool) {
	lead, rest := splitLeadingWhitespace(line)
	if !strings.HasPrefix(rest, "**") {
		return "", false
	}
	end := strings.Index(rest[2:], "**")
	if end < 0 {
		return "", false
	}
	labelEnd := end + 4
	return lead +
		styleTranscriptLine(theme.Transcript.AssistantHeading, rest[:labelEnd]) +
		styleAssistantText(rest[labelEnd:]), true
}

func splitLeadingWhitespace(line string) (string, string) {
	i := 0
	for i < len(line) {
		r := rune(line[i])
		if r != ' ' && r != '\t' {
			break
		}
		i++
	}
	return line[:i], line[i:]
}

func styleAssistantText(line string) string {
	return styleInlineCodeSegments(line, theme.Transcript.AssistantText)
}

func styleInlineCodeSegments(line string, baseStyle string) string {
	if line == "" {
		return ""
	}
	var out strings.Builder
	pos := 0
	for pos < len(line) {
		start := strings.IndexByte(line[pos:], '`')
		if start < 0 {
			out.WriteString(styleTranscriptLine(baseStyle, line[pos:]))
			break
		}
		start += pos
		end := strings.IndexByte(line[start+1:], '`')
		if end < 0 {
			out.WriteString(styleTranscriptLine(baseStyle, line[pos:]))
			break
		}
		end += start + 1
		out.WriteString(styleTranscriptLine(baseStyle, line[pos:start]))
		out.WriteString(styleTranscriptLine(theme.Transcript.AssistantInlineCode, line[start:end+1]))
		pos = end + 1
	}
	return out.String()
}
