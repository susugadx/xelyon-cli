package agent

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

func providerHistoryRawOutputBodyExcerpt(body string, budgetTokens int) string {
	body = strings.TrimSpace(body)
	if body == "" || budgetTokens <= 0 {
		return ""
	}
	if token.EstimateTokenCount(body) <= budgetTokens {
		return body
	}
	maxRunes := budgetTokens * 2
	if maxRunes < 256 {
		maxRunes = 256
	}
	runes := []rune(body)
	if len(runes) <= maxRunes {
		return body
	}
	headRunes := maxRunes / 2
	tailRunes := maxRunes - headRunes
	head := strings.TrimSpace(string(runes[:headRunes]))
	tail := strings.TrimSpace(string(runes[len(runes)-tailRunes:]))
	return head + "\n...\n" + tail
}

func providerHistoryRawOutputBodyCoverageExcerpt(body string, budgetTokens int, hints []string) (string, string) {
	body = strings.TrimSpace(body)
	if body == "" || budgetTokens <= 0 {
		return "", providerHistoryRawOutputRequiredRefsMissingReason
	}
	if strings.HasPrefix(body, "[matched raw output excerpt;") {
		if token.EstimateTokenCount(body) <= budgetTokens {
			return body, ""
		}
		if excerpt, ok := providerHistoryRawOutputShrinkMatchedExcerpt(body, hints, budgetTokens); ok {
			return excerpt, ""
		}
		return "", providerHistoryRawOutputActiveContextCoverageInsufficientReason
	}
	if token.EstimateTokenCount(body) <= budgetTokens {
		return body, ""
	}
	excerpt, ok := providerHistoryRawOutputMatchedBodyExcerpt(body, hints, budgetTokens)
	if ok {
		return excerpt, ""
	}
	return "", providerHistoryRawOutputActiveContextCoverageInsufficientReason
}

func providerHistoryRawOutputMatchedBodyExcerpt(body string, hints []string, budgetTokens int) (string, bool) {
	if budgetTokens <= 0 || len(hints) == 0 {
		return "", false
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	lineIndex, matchedTerm := providerHistoryRawOutputMatchedLine(lines, hints)
	if lineIndex < 0 {
		return "", false
	}
	maxRadius := providerHistoryRawOutputMinInt(4, providerHistoryRawOutputMaxInt(lineIndex, len(lines)-lineIndex-1))
	for radius := maxRadius; radius >= 0; radius-- {
		start := providerHistoryRawOutputMaxInt(0, lineIndex-radius)
		end := providerHistoryRawOutputMinInt(len(lines), lineIndex+radius+1)
		selected := append([]string(nil), lines[start:end]...)
		if radius == 0 {
			selected[0] = providerHistoryRawOutputTrimMatchedLine(selected[0], matchedTerm, budgetTokens)
		}
		excerpt := providerHistoryRawOutputRenderMatchedExcerpt(selected, matchedTerm, lineIndex, len(lines), start, end)
		if token.EstimateTokenCount(excerpt) <= budgetTokens {
			return excerpt, true
		}
	}
	return "", false
}

type providerHistoryRawOutputMatchedExcerpt struct {
	matchedTerm string
	lineIndex   int
	totalLines  int
	start       int
	lines       []string
}

func providerHistoryRawOutputShrinkMatchedExcerpt(body string, hints []string, budgetTokens int) (string, bool) {
	if budgetTokens <= 0 {
		return "", false
	}
	parsed, ok := providerHistoryRawOutputParseMatchedExcerpt(body)
	if !ok || len(parsed.lines) == 0 {
		return "", false
	}
	matchedTerm := parsed.matchedTerm
	matchOffset := parsed.lineIndex - parsed.start
	if matchOffset < 0 || matchOffset >= len(parsed.lines) {
		searchHints := hints
		if matchedTerm != "" {
			searchHints = append([]string{matchedTerm}, hints...)
		}
		if offset, term := providerHistoryRawOutputMatchedLine(parsed.lines, searchHints); offset >= 0 {
			matchOffset = offset
			parsed.lineIndex = parsed.start + offset
			if matchedTerm == "" {
				matchedTerm = term
			}
		}
	}
	if matchedTerm == "" || matchOffset < 0 || matchOffset >= len(parsed.lines) {
		return "", false
	}
	maxRadius := providerHistoryRawOutputMinInt(4, providerHistoryRawOutputMaxInt(matchOffset, len(parsed.lines)-matchOffset-1))
	for radius := maxRadius; radius >= 0; radius-- {
		startOffset := providerHistoryRawOutputMaxInt(0, matchOffset-radius)
		endOffset := providerHistoryRawOutputMinInt(len(parsed.lines), matchOffset+radius+1)
		selected := append([]string(nil), parsed.lines[startOffset:endOffset]...)
		if radius == 0 {
			selected[0] = providerHistoryRawOutputTrimMatchedLine(selected[0], matchedTerm, budgetTokens)
		}
		start := parsed.start + startOffset
		end := parsed.start + endOffset
		excerpt := providerHistoryRawOutputRenderMatchedExcerpt(selected, matchedTerm, parsed.lineIndex, parsed.totalLines, start, end)
		if token.EstimateTokenCount(excerpt) <= budgetTokens {
			return excerpt, true
		}
	}
	line := providerHistoryRawOutputTrimMatchedLine(parsed.lines[matchOffset], matchedTerm, budgetTokens/2)
	excerpt := providerHistoryRawOutputRenderMatchedExcerpt([]string{line}, matchedTerm, parsed.lineIndex, parsed.totalLines, parsed.lineIndex, parsed.lineIndex+1)
	if token.EstimateTokenCount(excerpt) <= budgetTokens {
		return excerpt, true
	}
	return "", false
}

func providerHistoryRawOutputParseMatchedExcerpt(body string) (providerHistoryRawOutputMatchedExcerpt, bool) {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "[matched raw output excerpt;") {
		return providerHistoryRawOutputMatchedExcerpt{}, false
	}
	matchedTerm := providerHistoryRawOutputParseMatchedExcerptTerm(lines[0])
	lineIndex, totalLines, ok := providerHistoryRawOutputParseMatchedExcerptLine(lines[0])
	if !ok {
		lineIndex = -1
	}
	content := append([]string(nil), lines[1:]...)
	start := 0
	if len(content) > 0 {
		if omitted, ok := providerHistoryRawOutputParseOmittedLineCount(content[0], "before"); ok {
			start = omitted
			content = content[1:]
		}
	}
	if len(content) > 0 {
		if _, ok := providerHistoryRawOutputParseOmittedLineCount(content[len(content)-1], "after"); ok {
			content = content[:len(content)-1]
		}
	}
	if totalLines <= 0 {
		totalLines = start + len(content)
	}
	return providerHistoryRawOutputMatchedExcerpt{
		matchedTerm: matchedTerm,
		lineIndex:   lineIndex,
		totalLines:  totalLines,
		start:       start,
		lines:       content,
	}, true
}

func providerHistoryRawOutputParseMatchedExcerptTerm(header string) string {
	const marker = "matched_term="
	start := strings.Index(header, marker)
	if start < 0 {
		return ""
	}
	rest := strings.TrimSpace(header[start+len(marker):])
	if quoted, err := strconv.QuotedPrefix(rest); err == nil {
		unquoted, err := strconv.Unquote(quoted)
		if err == nil {
			return unquoted
		}
	}
	end := strings.Index(rest, ";")
	if end < 0 {
		return ""
	}
	value := strings.TrimSpace(rest[:end])
	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return strings.Trim(value, `"`)
	}
	return unquoted
}

func providerHistoryRawOutputParseMatchedExcerptLine(header string) (int, int, bool) {
	const marker = "line="
	start := strings.Index(header, marker)
	if start < 0 {
		return -1, 0, false
	}
	rest := strings.TrimSuffix(strings.TrimSpace(header[start+len(marker):]), "]")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return -1, 0, false
	}
	lineNumber, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || lineNumber <= 0 {
		return -1, 0, false
	}
	totalLines, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || totalLines <= 0 {
		return -1, 0, false
	}
	return lineNumber - 1, totalLines, true
}

func providerHistoryRawOutputParseOmittedLineCount(line, direction string) (int, bool) {
	prefix := "[omitted "
	suffix := " lines " + direction + " match]"
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		return 0, false
	}
	value := strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix)
	count, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || count < 0 {
		return 0, false
	}
	return count, true
}

func providerHistoryRawOutputMatchedLine(lines []string, hints []string) (int, string) {
	for i, line := range lines {
		lowerLine := strings.ToLower(line)
		for _, hint := range hints {
			if hint != "" && strings.Contains(lowerLine, hint) {
				return i, hint
			}
		}
	}
	return -1, ""
}

func providerHistoryRawOutputRenderMatchedExcerpt(lines []string, matchedTerm string, lineIndex, totalLines, start, end int) string {
	parts := []string{fmt.Sprintf(
		"[matched raw output excerpt; matched_term=%q; line=%d/%d]",
		rawoutputs.RedactDisplaySecrets(matchedTerm),
		lineIndex+1,
		totalLines,
	)}
	if start > 0 {
		parts = append(parts, fmt.Sprintf("[omitted %d lines before match]", start))
	}
	parts = append(parts, lines...)
	if end < totalLines {
		parts = append(parts, fmt.Sprintf("[omitted %d lines after match]", totalLines-end))
	}
	return strings.Join(parts, "\n")
}

func providerHistoryRawOutputTrimMatchedLine(line, matchedTerm string, budgetTokens int) string {
	line = strings.TrimSpace(line)
	if line == "" || budgetTokens <= 0 || token.EstimateTokenCount(line) <= budgetTokens {
		return line
	}
	maxRunes := budgetTokens * 2
	if maxRunes < 256 {
		maxRunes = 256
	}
	runes := []rune(line)
	if len(runes) <= maxRunes {
		return line
	}
	index := providerHistoryRawOutputMatchedRuneIndex(line, matchedTerm)
	if index < 0 {
		return providerHistoryRawOutputBodyExcerpt(line, budgetTokens)
	}
	start := index - maxRunes/2
	if start < 0 {
		start = 0
	}
	end := start + maxRunes
	if end > len(runes) {
		end = len(runes)
		start = providerHistoryRawOutputMaxInt(0, end-maxRunes)
	}
	trimmed := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		trimmed = "..." + trimmed
	}
	if end < len(runes) {
		trimmed += "..."
	}
	return trimmed
}

func providerHistoryRawOutputMatchedRuneIndex(line, matchedTerm string) int {
	byteIndex := strings.Index(strings.ToLower(line), strings.ToLower(matchedTerm))
	if byteIndex < 0 {
		return -1
	}
	return len([]rune(line[:byteIndex]))
}

func providerHistoryRawOutputMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func providerHistoryRawOutputMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
