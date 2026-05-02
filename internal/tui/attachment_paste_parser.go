package tui

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandruntime"
)

const maxDroppedAttachments = maxComposerAttachments

type droppedPathParseKind int

const (
	droppedPathParseNotPath droppedPathParseKind = iota
	droppedPathParseReady
	droppedPathParseLimit
	droppedPathParseInvalid
)

type droppedPathParseResult struct {
	kind  droppedPathParseKind
	paths []string
}

type pastedPathTokenParseResult struct {
	tokens  []string
	invalid bool
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
	tokenResult := parsePastedPathTokens(line)
	if tokenResult.invalid {
		return nil, droppedPathParseInvalid
	}
	tokens := tokenResult.tokens
	if len(tokens) == 0 {
		return nil, droppedPathParseNotPath
	}

	normalizedPaths := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !looksLikePastedPathCandidate(token) {
			if len(tokens) == 1 {
				if resolved, ok := resolveSingleBareRelativeFilePath(token); ok {
					normalizedPaths = append(normalizedPaths, resolved)
					continue
				}
			}
			return nil, droppedPathParseNotPath
		}
		normalized := normalizePastedPathToken(token)
		if !normalized.isOK() {
			return nil, droppedPathParseNotPath
		}
		normalizedPaths = append(normalizedPaths, normalized.path)
	}
	return normalizedPaths, droppedPathParseReady
}

func resolveSingleBareRelativeFilePath(token string) (string, bool) {
	trimmed := trimPathQuotes(strings.TrimSpace(token))
	if trimmed == "" || trimmed == "." || trimmed == ".." {
		return "", false
	}
	if strings.ContainsAny(trimmed, `/\`) || strings.Contains(trimmed, ".") {
		return "", false
	}

	normalized := normalizePastedPathToken(trimmed)
	if !normalized.isOK() {
		return "", false
	}

	info, err := os.Stat(normalized.path)
	if err != nil || info.IsDir() {
		return "", false
	}
	return normalized.path, true
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

func parsePastedPathTokens(line string) pastedPathTokenParseResult {
	if strings.Contains(line, `\ `) {
		return pastedPathTokenParseResult{tokens: []string{line}}
	}
	tokens, status := commandruntime.SplitStrict(line)
	if !status.IsOK() {
		return pastedPathTokenParseResult{invalid: true}
	}
	if len(tokens) > 0 {
		return pastedPathTokenParseResult{tokens: tokens}
	}
	return pastedPathTokenParseResult{tokens: []string{line}}
}
