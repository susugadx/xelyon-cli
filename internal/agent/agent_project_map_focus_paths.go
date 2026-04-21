package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
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

func projectMapPriorityPathsFromInput(cwd, rootPath string, candidates []string, limit int) []string {
	if limit <= 0 {
		return nil
	}

	capHint := len(candidates)
	if capHint > limit {
		capHint = limit
	}
	normalized := make([]string, 0, capHint)
	for _, candidate := range candidates {
		path, ok := resolveProjectMapInputCandidate(cwd, rootPath, candidate)
		if !ok {
			continue
		}
		normalized = append(normalized, path)
		if len(normalized) >= limit {
			break
		}
	}
	return normalized
}

func resolveProjectMapInputCandidate(cwd, rootPath, candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" || rootPath == "" {
		return "", false
	}
	if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
		return "", false
	}
	if filepath.IsAbs(candidate) {
		absPath := filepath.Clean(candidate)
		if !projectMapPathExists(absPath) {
			return "", false
		}
		return canonicalizeProjectMapPriorityPath(rootPath, absPath)
	}
	if isWindowsAbsoluteProjectMapPath(candidate) {
		absPath := filepath.Clean(windowsAbsoluteProjectMapPathToLocal(candidate))
		if !projectMapPathExists(absPath) {
			return "", false
		}
		return canonicalizeProjectMapPriorityPath(rootPath, absPath)
	}

	sessionAbs := filepath.Clean(filepath.Join(cwd, filepath.FromSlash(candidate)))
	rootAbs := filepath.Clean(filepath.Join(rootPath, filepath.FromSlash(candidate)))

	sessionExists := projectMapPathExists(sessionAbs)
	rootExists := projectMapPathExists(rootAbs)

	switch {
	case rootExists && (looksRepoRelativeProjectMapPath(candidate) || !sessionExists):
		return canonicalizeProjectMapPriorityPath(rootPath, rootAbs)
	case sessionExists:
		return canonicalizeProjectMapPriorityPath(rootPath, sessionAbs)
	case rootExists:
		return canonicalizeProjectMapPriorityPath(rootPath, rootAbs)
	default:
		return "", false
	}
}

func canonicalizeProjectMapPriorityPath(rootPath, absPath string) (string, bool) {
	if rootPath == "" || absPath == "" {
		return "", false
	}

	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		return "", false
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", false
	}

	relPath, err := filepath.Rel(rootAbs, absPath)
	if err != nil {
		return "", false
	}
	if relPath == "." {
		return "", false
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", false
	}

	return filepath.ToSlash(filepath.Clean(relPath)), true
}

func projectMapPathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func looksRepoRelativeProjectMapPath(candidate string) bool {
	candidate = filepath.ToSlash(strings.TrimSpace(candidate))
	if candidate == "" {
		return false
	}
	if strings.HasPrefix(candidate, "./") || strings.HasPrefix(candidate, "../") {
		return false
	}
	return strings.Contains(candidate, "/")
}

func isWindowsAbsoluteProjectMapPath(candidate string) bool {
	if len(candidate) < 4 {
		return false
	}
	if (candidate[0] < 'A' || candidate[0] > 'Z') && (candidate[0] < 'a' || candidate[0] > 'z') {
		return false
	}
	return candidate[1] == ':' && candidate[2] == '/'
}

func windowsAbsoluteProjectMapPathToLocal(candidate string) string {
	if !isWindowsAbsoluteProjectMapPath(candidate) {
		return candidate
	}
	return candidate[2:]
}

func renderProjectMapFocusOverlay(paths []string) string {
	paths = dedupeProjectMapPriorityPaths(paths)
	if len(paths) == 0 {
		return ""
	}
	if len(paths) > projectMapFocusMaxPaths {
		paths = paths[:projectMapFocusMaxPaths]
	}

	var b strings.Builder
	b.WriteString("Focus files for current task:\n")
	for _, path := range paths {
		b.WriteString("- ")
		b.WriteString(path)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func composeProjectMapPromptSection(baseSection, focusSection string) string {
	baseSection = strings.TrimRight(baseSection, "\n")
	focusSection = strings.TrimRight(focusSection, "\n")

	switch {
	case baseSection == "":
		if focusSection == "" {
			return ""
		}
		return "## Project Map\n\n" + focusSection
	case focusSection == "":
		return baseSection
	default:
		return baseSection + "\n\n" + focusSection
	}
}

func countProjectMapFocusLines(section string) int {
	if strings.TrimSpace(section) == "" {
		return 0
	}
	count := 0
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "- ") {
			count++
		}
	}
	return count
}

func dedupeProjectMapPriorityPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	deduped := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		deduped = append(deduped, path)
	}
	return deduped
}

func appendProjectMapSection(systemPrompt, section string) string {
	if strings.TrimSpace(section) == "" {
		return systemPrompt
	}

	// Project Map is the most volatile part of the system prompt.
	// Put it behind a cache boundary so Claude can reuse the stable prefix
	// even when the map changes after edits or repo-state updates.
	if !strings.Contains(systemPrompt, api.SystemPromptCacheBoundary) {
		return systemPrompt + api.SystemPromptCacheBoundary + section
	}

	return systemPrompt + "\n\n" + section

}
