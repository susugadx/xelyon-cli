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

type projectMapInputMatch struct {
	candidate string
	start     int
	end       int
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
