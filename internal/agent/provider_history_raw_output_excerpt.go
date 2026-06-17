package agent

import (
	"fmt"
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
