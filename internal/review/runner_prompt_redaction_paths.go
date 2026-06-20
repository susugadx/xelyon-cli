package review

import (
	"path/filepath"
	"strings"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

func normalizeReviewRunnerPromptPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.Clean(filepath.FromSlash(path))
}

func reviewRunnerPromptSlashPath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), `\`, "/")
}

func reviewRunnerPromptIsolatedProbeRoot(path string) (string, bool) {
	cleaned := normalizeReviewRunnerPromptPath(path)
	if cleaned == "" {
		return "", false
	}

	parts := strings.Split(reviewRunnerPromptSlashPath(cleaned), "/")
	for i, part := range parts {
		for _, prefix := range reviewprobe.ReviewProbeIsolatedTempRootPrefixes() {
			if strings.HasPrefix(part, prefix) {
				return filepath.FromSlash(strings.Join(parts[:i+1], "/")), true
			}
		}
	}
	return "", false
}

func reviewRunnerPromptIsolatedProbeRootsInText(text string) []string {
	if text == "" {
		return nil
	}

	seen := map[string]struct{}{}
	var roots []string
	for _, prefix := range reviewprobe.ReviewProbeIsolatedTempRootPrefixes() {
		for searchStart := 0; searchStart < len(text); {
			prefixOffset := strings.Index(text[searchStart:], prefix)
			if prefixOffset < 0 {
				break
			}

			prefixStart := searchStart + prefixOffset
			candidate := text[reviewRunnerPromptFreeTextPathStart(text, prefixStart):reviewRunnerPromptFreeTextPathEnd(text, prefixStart+len(prefix))]
			candidate = strings.TrimRight(candidate, `:,.);]}"'`)
			root, ok := reviewRunnerPromptIsolatedProbeRoot(candidate)
			if ok {
				key := reviewRunnerPromptSlashPath(root)
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					roots = append(roots, root)
				}
			}
			searchStart = prefixStart + len(prefix)
		}
	}
	return roots
}

func reviewRunnerPromptFreeTextPathStart(text string, prefixStart int) int {
	start := prefixStart
	for start > 0 && isReviewRunnerPromptFreeTextPathByte(text[start-1]) {
		start--
	}
	return start
}

func reviewRunnerPromptFreeTextPathEnd(text string, prefixEnd int) int {
	end := prefixEnd
	for end < len(text) && isReviewRunnerPromptFreeTextPathByte(text[end]) {
		end++
	}
	return end
}

func isReviewRunnerPromptAbsolutePath(path string) bool {
	return filepath.IsAbs(path) || reviewevidence.IsReviewEvidenceWindowsAbsolutePath(path)
}

func isReviewRunnerPromptPathTokenByte(b byte) bool {
	return isReviewASCIIAlpha(b) || ('0' <= b && b <= '9') || b == '_' || b == '-' || b == '.'
}

func isReviewRunnerPromptFreeTextPathByte(b byte) bool {
	return isReviewRunnerPromptPathTokenByte(b) || b == '/' || b == '\\' || b == ':'
}
