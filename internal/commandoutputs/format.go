package commandoutputs

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func outputLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = stripANSI(line)
	}
	return lines
}

func firstLastEntries(lines []string, sideLimit int) ([]string, int) {
	first, last, omitted := firstLastRaw(lines, sideLimit)
	if omitted <= 0 {
		return nil, 0
	}
	out := append([]string{}, first...)
	out = append(out, fmt.Sprintf("[omitted %d entries]", omitted))
	out = append(out, last...)
	return out, omitted
}

func firstLastRaw(lines []string, sideLimit int) ([]string, []string, int) {
	if len(lines) <= sideLimit*2 {
		return nil, nil, 0
	}
	first := sanitizeLines(lines[:sideLimit])
	last := sanitizeLines(lines[len(lines)-sideLimit:])
	return first, last, len(lines) - len(first) - len(last)
}

func listEntriesWithOmission(entries []string, sideLimit int) []string {
	if len(entries) <= sideLimit*2 {
		return sanitizeLines(entries)
	}
	out := append([]string{}, sanitizeLines(entries[:sideLimit])...)
	out = append(out, fmt.Sprintf("  [omitted %d paths]", len(entries)-sideLimit*2))
	out = append(out, sanitizeLines(entries[len(entries)-sideLimit:])...)
	return out
}

func sanitizeLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, sanitizeHeaderValue(line))
	}
	return out
}

func redactLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, redactSecrets(line))
	}
	return out
}

func indentLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, "  "+line)
	}
	return out
}

func uniqueLines(lines, existing []string) []string {
	seen := map[string]struct{}{}
	for _, line := range existing {
		seen[line] = struct{}{}
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = sanitizeHeaderValue(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	return out
}

func redactSecrets(line string) string {
	line = secretAssignmentRegexp.ReplaceAllString(line, "$1=[redacted]")
	line = authHeaderRegexp.ReplaceAllStringFunc(line, redactAuthorizationHeader)
	line = secretHeaderRegexp.ReplaceAllString(line, "$1: [redacted]")
	line = cookieHeaderRegexp.ReplaceAllString(line, "$1: [redacted]")
	line = urlSecretQueryRegexp.ReplaceAllString(line, "${1}[redacted]")
	return line
}

func redactAuthorizationHeader(value string) string {
	match := authHeaderRegexp.FindStringSubmatch(value)
	if len(match) < 4 {
		return "[redacted authorization]"
	}
	scheme := match[2]
	if scheme != "" && looksLikeAuthorizationScheme(scheme) {
		return match[1] + scheme + " [redacted]"
	}
	return match[1] + "[redacted]"
}

func looksLikeAuthorizationScheme(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			continue
		}
		return false
	}
	return true
}

func sanitizeCommandForHeader(command string) string {
	command = sanitizeHeaderValue(command)
	if command == "" {
		return ""
	}
	return trimRunes(command, commandSummaryMaxRunes)
}

func sanitizeHeaderValue(value string) string {
	value = stripANSI(value)
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ", `"`, "'").Replace(value)
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func stripANSI(value string) string {
	return ansiEscapeRegexp.ReplaceAllString(value, "")
}

func trimRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

func savedBytes(originalBytes, replacementBytes int) int {
	if originalBytes <= replacementBytes {
		return 0
	}
	return originalBytes - replacementBytes
}

func savedTokens(originalTokens, replacementTokens int) int {
	if originalTokens <= replacementTokens {
		return 0
	}
	return originalTokens - replacementTokens
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func omittedFailureLineCount(originalLines, retainedParts int) int {
	if originalLines <= retainedParts {
		return 0
	}
	return originalLines - retainedParts
}
