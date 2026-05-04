package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandruntime"
)

const maxDroppedAttachments = maxComposerAttachments

type droppedPathParseKind int

const (
	droppedPathParseNotPath droppedPathParseKind = iota
	droppedPathParseReady
	droppedPathParseLimit
)

type droppedPathParseResult struct {
	kind  droppedPathParseKind
	paths []string
}

func parseDroppedPaths(content string) droppedPathParseResult {
	lines := droppedPathLines(content)
	if len(lines) == 0 {
		return droppedPathParseResult{kind: droppedPathParseNotPath}
	}

	candidates, kind := collectDroppedPathCandidates(lines)
	if kind != droppedPathParseReady {
		return droppedPathParseResult{kind: kind}
	}

	if len(candidates) > maxDroppedAttachments {
		return droppedPathParseResult{kind: droppedPathParseLimit}
	}

	dedup := dedupeDroppedPathCandidates(candidates)
	if len(dedup) == 0 {
		return droppedPathParseResult{kind: droppedPathParseNotPath}
	}
	return droppedPathParseResult{
		kind:  droppedPathParseReady,
		paths: dedup,
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

func collectDroppedPathCandidates(lines []string) ([]string, droppedPathParseKind) {
	candidates := make([]string, 0, len(lines))
	for _, line := range lines {
		parsedLine, kind := parseDroppedPathLine(line)
		if kind != droppedPathParseReady {
			return nil, kind
		}
		candidates = append(candidates, parsedLine...)
	}
	if len(candidates) == 0 {
		return nil, droppedPathParseNotPath
	}
	return candidates, droppedPathParseReady
}

func parseDroppedPathLine(line string) ([]string, droppedPathParseKind) {
	tokens := parsePastedPathTokens(line)
	if len(tokens) == 0 {
		return nil, droppedPathParseNotPath
	}

	normalizedPaths, ok := resolveDroppedPathTokens(tokens)
	if !ok {
		return nil, droppedPathParseNotPath
	}
	return normalizedPaths, droppedPathParseReady
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
