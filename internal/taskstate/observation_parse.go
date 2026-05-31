package taskstate

import (
	"regexp"
	"strconv"
	"strings"
)

type renderedPathLocation struct {
	path      string
	startLine int
	endLine   int
	detail    string
}

type numberedSymbolCandidate struct {
	path          string
	startLine     int
	endLine       int
	repoPreferred bool
}

var (
	readHeaderRe          = regexp.MustCompile(`^\s*📄\s+File:\s+(.+?)\s*$`)
	searchHeaderRe        = regexp.MustCompile(`^\s*📄\s+(.+?)\s+\(\d+\s+match\(es\)\)`)
	symbolHeaderRe        = regexp.MustCompile(`^\s*──\s+.+?\s+\(L(\d+)(?:-L(\d+))?\)\s+in\s+(.+?)\s*(?:\[[A-Za-z]\d+\])?\s+──\s*$`)
	numberedSymbolInRe    = regexp.MustCompile(`^\s*.+?\s+\(L(\d+)(?:-L(\d+))?\)\s+in\s+(.+?)\s*(?:\[[A-Za-z]\d+\])?\s*$`)
	numberedSymbolLineRe  = regexp.MustCompile(`\s+\(L(\d+)(?:-L(\d+))?\)\s*(?:\[[A-Za-z]\d+\])?\s*$`)
	readLineRe            = regexp.MustCompile(`^\s*(\d+):\s?(.*)$`)
	searchLineRe          = regexp.MustCompile(`^\s*(?:\[[^\]]+\]\s*)?>?\s*(\d+)\s*│\s?(.*)$`)
	recommendedReadItemRe = regexp.MustCompile(`^\s*-\s+(.+?)\s*$`)
)

var numberedSymbolPathKindTokens = []string{
	" function ",
	" method ",
	" type ",
	" interface ",
	" const ",
	" var ",
}

func parseReadHeaderLine(line string) (path string, startLine int, endLine int, ok bool) {
	match := readHeaderRe.FindStringSubmatch(line)
	if match == nil {
		return "", 0, 0, false
	}
	return splitPathRangeSuffix(stripLocatorSuffix(match[1]))
}

func parseSearchHeaderLine(line string) (string, bool) {
	match := searchHeaderRe.FindStringSubmatch(line)
	if match == nil {
		return "", false
	}
	path, _, _, ok := splitPathRangeSuffix(stripLocatorSuffix(match[1]))
	return path, ok
}

func parseSymbolHeaderLine(line string) (path string, startLine int, endLine int, ok bool) {
	match := symbolHeaderRe.FindStringSubmatch(line)
	if match == nil {
		return "", 0, 0, false
	}
	startLine, _ = strconv.Atoi(match[1])
	endLine = startLine
	if match[2] != "" {
		endLine, _ = strconv.Atoi(match[2])
	}
	path, _, _, ok = splitPathRangeSuffix(stripLocatorSuffix(match[3]))
	return path, startLine, endLine, ok
}

func parseNumberedSymbolCandidateLine(line string) (numberedSymbolCandidate, bool) {
	rest, ok := splitNumberedListLinePrefix(line)
	if !ok {
		return numberedSymbolCandidate{}, false
	}
	if candidate, ok := parseNumberedPathFirstSymbolCandidate(rest); ok {
		return candidate, true
	}
	return parseNumberedInClauseSymbolCandidate(rest)
}

func splitNumberedListLinePrefix(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	dotIdx := strings.Index(line, ".")
	if dotIdx <= 0 {
		return "", false
	}
	for _, r := range line[:dotIdx] {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	rest := strings.TrimSpace(line[dotIdx+1:])
	if rest == "" {
		return "", false
	}
	return rest, true
}

func parseNumberedPathFirstSymbolCandidate(rest string) (numberedSymbolCandidate, bool) {
	path, ok := parseNumberedSymbolPathPrefix(rest)
	if !ok {
		return numberedSymbolCandidate{}, false
	}
	startLine, endLine, ok := parseNumberedSymbolLineRange(rest)
	if !ok {
		return numberedSymbolCandidate{}, false
	}
	return numberedSymbolCandidate{
		path:          path,
		startLine:     startLine,
		endLine:       endLine,
		repoPreferred: true,
	}, true
}

func parseNumberedInClauseSymbolCandidate(rest string) (numberedSymbolCandidate, bool) {
	match := numberedSymbolInRe.FindStringSubmatch(rest)
	if match == nil {
		return numberedSymbolCandidate{}, false
	}
	startLine, endLine, ok := parseLineRangeMatches(match[1], match[2])
	if !ok {
		return numberedSymbolCandidate{}, false
	}
	path, _, _, pathOK := splitPathRangeSuffix(stripLocatorSuffix(match[3]))
	if !pathOK {
		return numberedSymbolCandidate{}, false
	}
	return numberedSymbolCandidate{
		path:      path,
		startLine: startLine,
		endLine:   endLine,
	}, true
}

func parseNumberedSymbolPathPrefix(rest string) (string, bool) {
	for _, token := range numberedSymbolPathKindTokens {
		if idx := strings.Index(rest, token); idx > 0 {
			path := strings.TrimSpace(rest[:idx])
			if path != "" {
				return path, true
			}
		}
	}
	return "", false
}

func parseNumberedSymbolLineRange(rest string) (startLine int, endLine int, ok bool) {
	match := numberedSymbolLineRe.FindStringSubmatch(rest)
	if match == nil {
		return 0, 0, false
	}
	return parseLineRangeMatches(match[1], match[2])
}

func parseLineRangeMatches(start, end string) (startLine int, endLine int, ok bool) {
	startLine, _ = strconv.Atoi(start)
	endLine = startLine
	if end != "" {
		endLine, _ = strconv.Atoi(end)
	}
	return startLine, endLine, startLine > 0 && endLine > 0
}

func isAmbiguousSymbolListHeader(line string) bool {
	return strings.HasPrefix(line, "Multiple definitions found ") ||
		strings.HasPrefix(line, "Multiple symbols matched ")
}

func parseFormattedEvidenceLine(line string) (lineNo int, excerpt string, ok bool) {
	if match := searchLineRe.FindStringSubmatch(line); match != nil {
		lineNo, _ = strconv.Atoi(match[1])
		return lineNo, strings.TrimSpace(match[2]), lineNo > 0
	}
	if match := readLineRe.FindStringSubmatch(line); match != nil {
		lineNo, _ = strconv.Atoi(match[1])
		return lineNo, strings.TrimSpace(match[2]), lineNo > 0
	}
	return 0, "", false
}

func splitPathRangeSuffix(raw string) (path string, startLine int, endLine int, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, 0, false
	}
	if idx := strings.LastIndex(raw, ":"); idx > 0 && idx < len(raw)-1 {
		suffix := raw[idx+1:]
		if start, end, rangeOK := parseLineRangeSuffix(suffix); rangeOK {
			return strings.TrimSpace(raw[:idx]), start, end, true
		}
	}
	return raw, 0, 0, true
}

func parseLineRangeSuffix(suffix string) (startLine int, endLine int, ok bool) {
	if strings.Contains(suffix, "-") {
		parts := strings.SplitN(suffix, "-", 2)
		if len(parts) != 2 || !isDigits(parts[0]) || !isDigits(parts[1]) {
			return 0, 0, false
		}
		startLine, _ = strconv.Atoi(parts[0])
		endLine, _ = strconv.Atoi(parts[1])
		if startLine <= 0 || endLine <= 0 {
			return 0, 0, false
		}
		return startLine, endLine, true
	}
	if !isDigits(suffix) {
		return 0, 0, false
	}
	startLine, _ = strconv.Atoi(suffix)
	if startLine <= 0 {
		return 0, 0, false
	}
	return startLine, startLine, true
}

func stripLocatorSuffix(s string) string {
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	if strings.HasPrefix(last, "[") && strings.HasSuffix(last, "]") {
		return strings.Join(fields[:len(fields)-1], " ")
	}
	return s
}

func parseRecommendedReadItem(item string) (path string, reason string, ok bool) {
	location, ok := parseRenderedPathLocationItem(item)
	if ok {
		reason = "recommended read"
		if location.detail != "" {
			reason = location.detail
		}
		return location.path, reason, true
	}
	item = stripLocatorSuffix(strings.TrimSpace(item))
	if item == "" {
		return "", "", false
	}
	reason = "recommended read"
	if before, after, found := strings.Cut(item, "|"); found {
		item = strings.TrimSpace(before)
		if trimmed := strings.TrimSpace(after); trimmed != "" {
			reason = trimmed
		}
	}
	return item, reason, true
}

func parseRenderedPathLocationBullet(line string) (renderedPathLocation, bool) {
	match := recommendedReadItemRe.FindStringSubmatch(line)
	if match == nil {
		return renderedPathLocation{}, false
	}
	return parseRenderedPathLocationItem(match[1])
}

func parseRenderedPathLocationItem(item string) (renderedPathLocation, bool) {
	item = stripLocatorSuffix(strings.TrimSpace(item))
	if item == "" {
		return renderedPathLocation{}, false
	}
	detail := ""
	if before, after, found := strings.Cut(item, "|"); found {
		item = strings.TrimSpace(before)
		detail = strings.TrimSpace(after)
	}
	path, startLine, endLine, ok := splitRenderedPathLocation(item)
	if !ok {
		return renderedPathLocation{}, false
	}
	return renderedPathLocation{
		path:      path,
		startLine: startLine,
		endLine:   endLine,
		detail:    detail,
	}, true
}

func splitRenderedPathLocation(raw string) (path string, startLine int, endLine int, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, 0, false
	}
	parts := strings.Split(raw, ":")
	for i := 1; i < len(parts); i++ {
		start, end, suffixOK := parseRenderedLocationSuffix(parts[i])
		if !suffixOK {
			continue
		}
		path = strings.TrimSpace(strings.Join(parts[:i], ":"))
		if path == "" {
			return "", 0, 0, false
		}
		return path, start, end, true
	}
	return "", 0, 0, false
}

func parseRenderedLocationSuffix(s string) (startLine int, endLine int, ok bool) {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return 0, 0, false
	}
	startLine, endLine, ok = parseLineRangeSuffix(fields[0])
	if !ok {
		return 0, 0, false
	}
	if len(fields) == 1 || fields[1] == "in" {
		return startLine, endLine, true
	}
	return 0, 0, false
}

func locationEvidenceExcerpt(location renderedPathLocation, fallback string) string {
	if location.detail != "" {
		return location.detail
	}
	return fallback
}

func parsePathLineFacts(repoRoot, invocationCWD, output string) []string {
	var paths []string
	for _, line := range strings.Split(output, "\n") {
		for _, token := range strings.Fields(line) {
			path, ok := parsePathLineToken(token)
			if !ok {
				continue
			}
			normalized, pathOK := normalizeLedgerPath(repoRoot, invocationCWD, path)
			if !pathOK {
				continue
			}
			paths = appendUniqueString(paths, normalized, maxRecordedFiles)
		}
	}
	return paths
}

func parsePathLineToken(token string) (string, bool) {
	token = strings.Trim(token, " \t\r\n,;()")
	token = strings.TrimRight(token, ":")
	if token == "" || strings.Contains(token, "://") {
		return "", false
	}
	parts := strings.Split(token, ":")
	for i := 1; i < len(parts); i++ {
		if !isDigits(parts[i]) {
			continue
		}
		path := strings.Join(parts[:i], ":")
		if looksLikePathLineCandidate(path) {
			return path, true
		}
	}
	return "", false
}

func looksLikePathLineCandidate(path string) bool {
	if path == "" {
		return false
	}
	if strings.Contains(path, "/") || strings.Contains(path, `\`) {
		return true
	}
	base := path
	if idx := strings.LastIndexAny(base, `/\`); idx >= 0 {
		base = base[idx+1:]
	}
	return strings.Contains(base, ".") || base == "Makefile" || base == "Dockerfile"
}
