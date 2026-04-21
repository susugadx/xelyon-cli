package agent

import (
	"regexp"
	"strings"
)

var (
	// 引用付き path は slash を含むものを優先的に取る。
	projectMapQuotedPathPattern = regexp.MustCompile("[\"'`]([^\"'`]+(?:[\\\\/][^\"'`]+)+)[\"'`]")
	// slash を含まない quoted filename は dedicated pattern で扱う。
	// 'design spec.md' のような空白付き filename は他パターンではまたげない。
	projectMapQuotedFilenamePattern = regexp.MustCompile(`["']([^"']+\.[a-zA-Z0-9]{1,10})["']`)
	projectMapInputPathPatterns     = []*regexp.Regexp{
		projectMapQuotedPathPattern,
		regexp.MustCompile(`\b([A-Za-z]:[\\/][^\s"'` + "`" + `]+)\b`),
		regexp.MustCompile(`\b((?:[\w.-]+[\\/])+[\w./\\-]*)\b`),
		projectMapQuotedFilenamePattern,
		regexp.MustCompile(`\b((?:[\w.-]+/)*[\w.-]+\.[a-zA-Z0-9]{1,10})\b`),
		regexp.MustCompile(`(/[^\s"']+)`),
	}
)

const projectMapFocusMaxPaths = 5

type projectMapInputMatch struct {
	candidate string
	start     int
	end       int
}

func extractProjectMapFocusPaths(cwd, rootPath, input string, limit int) []string {
	if limit <= 0 {
		return nil
	}

	paths := dedupeProjectMapPriorityPaths(projectMapPriorityPathsFromInput(cwd, rootPath, extractProjectMapPathsFromInput(input), limit))
	if len(paths) == 0 {
		return nil
	}
	return paths
}

func extractProjectMapPathsFromInput(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	pathSet := make(map[string]struct{})
	var accepted []projectMapInputMatch
	var paths []string
	for _, pattern := range projectMapInputPathPatterns {
		matches := pattern.FindAllStringSubmatchIndex(input, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}
			start, end := match[2], match[3]
			if shouldSkipProjectMapInputMatch(accepted, start, end) {
				continue
			}
			candidate := cleanProjectMapInputPathCandidate(input[start:end])
			if candidate == "" {
				continue
			}
			if _, ok := pathSet[candidate]; ok {
				continue
			}
			pathSet[candidate] = struct{}{}
			accepted = append(accepted, projectMapInputMatch{
				candidate: candidate,
				start:     start,
				end:       end,
			})
			paths = append(paths, candidate)
		}
	}
	return filterProjectMapInputCandidates(paths)
}

func shouldSkipProjectMapInputMatch(accepted []projectMapInputMatch, start, end int) bool {
	for _, existing := range accepted {
		if start >= existing.start && end <= existing.end {
			return true
		}
	}
	return false
}

func cleanProjectMapInputPathCandidate(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	candidate = strings.Trim(candidate, ".,:;!?()[]{}<>")
	candidate = strings.Trim(candidate, "\"'`")
	candidate = strings.ReplaceAll(candidate, "\\", "/")
	if candidate == "" {
		return ""
	}
	if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
		return ""
	}
	if !strings.Contains(candidate, "/") && !strings.Contains(candidate, ".") {
		return ""
	}
	return candidate
}

func filterProjectMapInputCandidates(candidates []string) []string {
	if len(candidates) == 0 {
		return nil
	}

	type normalizedCandidate struct {
		original   string
		normalized string
	}

	normalized := make([]normalizedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		cleaned := cleanProjectMapInputPathCandidate(candidate)
		if cleaned == "" {
			continue
		}
		normalized = append(normalized, normalizedCandidate{
			original:   cleaned,
			normalized: strings.ToLower(cleaned),
		})
	}

	filtered := make([]string, 0, len(normalized))
	for i, candidate := range normalized {
		if candidate.original == "" {
			continue
		}
		if strings.Contains(candidate.original, "/") {
			filtered = append(filtered, candidate.original)
			continue
		}

		shadowed := false
		for j, other := range normalized {
			if i == j {
				continue
			}
			if !strings.Contains(other.original, "/") {
				continue
			}
			if strings.HasSuffix(other.normalized, "/"+candidate.normalized) {
				shadowed = true
				break
			}
		}
		if !shadowed {
			filtered = append(filtered, candidate.original)
		}
	}

	return dedupeProjectMapPriorityPaths(filtered)
}
