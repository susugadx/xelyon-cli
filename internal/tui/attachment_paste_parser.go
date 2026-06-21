package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"
)

const maxDroppedAttachments = tuiattachments.MaxComposerAttachments

func parseDroppedPaths(content string) tuiattachments.DroppedPathParseResult {
	lines := droppedPathLines(content)
	if len(lines) == 0 {
		return tuiattachments.DroppedPathParseResult{Kind: tuiattachments.DroppedPathParseNotPath}
	}

	candidates, kind := collectDroppedPathCandidates(lines)
	if kind != tuiattachments.DroppedPathParseReady {
		return tuiattachments.DroppedPathParseResult{Kind: kind}
	}

	if len(candidates) > maxDroppedAttachments {
		return tuiattachments.DroppedPathParseResult{Kind: tuiattachments.DroppedPathParseLimit}
	}

	dedup := dedupeDroppedPathCandidates(candidates)
	if len(dedup) == 0 {
		return tuiattachments.DroppedPathParseResult{Kind: tuiattachments.DroppedPathParseNotPath}
	}
	return tuiattachments.DroppedPathParseResult{
		Kind:  tuiattachments.DroppedPathParseReady,
		Paths: dedup,
	}
}

func droppedPathLines(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}

	rawLines := strings.Split(strings.ReplaceAll(trimmed, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func collectDroppedPathCandidates(lines []string) ([]string, tuiattachments.DroppedPathParseKind) {
	candidates := make([]string, 0, len(lines))
	for _, line := range lines {
		parsedLine, kind := parseDroppedPathLine(line)
		if kind != tuiattachments.DroppedPathParseReady {
			return nil, kind
		}
		candidates = append(candidates, parsedLine...)
	}
	if len(candidates) == 0 {
		return nil, tuiattachments.DroppedPathParseNotPath
	}
	return candidates, tuiattachments.DroppedPathParseReady
}

func parseDroppedPathLine(line string) ([]string, tuiattachments.DroppedPathParseKind) {
	tokens := parsePastedPathTokens(line)
	if len(tokens) == 0 {
		return nil, tuiattachments.DroppedPathParseNotPath
	}

	normalizedPaths, ok := resolveDroppedPathTokens(tokens)
	if !ok {
		return nil, tuiattachments.DroppedPathParseNotPath
	}
	return normalizedPaths, tuiattachments.DroppedPathParseReady
}

func resolveDroppedPathTokens(tokens []string) ([]string, bool) {
	normalizedPaths := make([]string, 0, len(tokens))
	ctx := pathCandidateContext{allowSingleBareRelative: len(tokens) == 1}
	for _, token := range tokens {
		resolved, ok := resolvePathCandidateToken(token, ctx)
		if !ok {
			return nil, false
		}
		normalizedPaths = append(normalizedPaths, resolved)
	}
	return normalizedPaths, true
}

func dedupeDroppedPathCandidates(candidates []string) []string {
	dedup := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, path := range candidates {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		dedup = append(dedup, path)
	}
	return dedup
}

func parsePastedPathTokens(line string) []string {
	if strings.Contains(line, `\ `) {
		return []string{line}
	}
	tokens, status := commandruntime.SplitStrict(line)
	if !status.IsOK() {
		return []string{line}
	}
	if len(tokens) > 0 {
		return tokens
	}
	return []string{line}
}
