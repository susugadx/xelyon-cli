package review

import (
	"path/filepath"
	"strings"
)

const reviewEvidenceOutsideRepoPathDisplay = "<outside-repo>"

func formatReviewEvidencePathDisplay(repoRoot, candidate string) string {
	canonicalRepoRoot, ok := canonicalReviewEvidencePathDisplayRepoRoot(repoRoot)
	if !ok {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	return formatReviewEvidencePathDisplayWithCanonicalRepoRoot(canonicalRepoRoot, candidate)
}

func formatReviewEvidenceOptionalPathDisplay(repoRoot, path string) string {
	if path == "" {
		return ""
	}
	return formatReviewEvidencePathDisplay(repoRoot, path)
}

func formatReviewEvidencePathDisplays(repoRoot string, paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		result = append(result, formatReviewEvidencePathDisplay(repoRoot, path))
	}
	return result
}

func canonicalReviewEvidencePathDisplayRepoRoot(repoRoot string) (string, bool) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", false
	}
	resolvedRepoRoot, err := resolveReviewEvidenceDir(repoRoot, "")
	if err != nil {
		return "", false
	}
	canonicalRepoRoot, err := canonicalReviewEvidenceRepoRoot(resolvedRepoRoot)
	if err != nil {
		return "", false
	}
	return canonicalRepoRoot, true
}

func formatReviewEvidencePathDisplayWithCanonicalRepoRoot(canonicalRepoRoot, candidate string) string {
	if isReviewEvidenceWindowsAbsolutePath(candidate) && !filepath.IsAbs(candidate) {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	if !filepath.IsAbs(candidate) {
		display, ok := formatReviewEvidenceLexicalRelativePathDisplay(candidate)
		if !ok {
			return reviewEvidenceOutsideRepoPathDisplay
		}
		return display
	}

	display, ok := formatReviewEvidenceAbsolutePathDisplay(canonicalRepoRoot, candidate)
	if !ok {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	return display
}

func formatReviewEvidenceLexicalRelativePathDisplay(candidate string) (string, bool) {
	if candidate == "" {
		return "", false
	}

	cleaned := filepath.Clean(filepath.FromSlash(candidate))
	if cleaned == "." {
		return ".", true
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(cleaned), true
}

func formatReviewEvidenceAbsolutePathDisplay(repoRoot, candidate string) (string, bool) {
	cleaned := filepath.Clean(candidate)
	if display, ok := formatReviewEvidenceRepoRelativePathDisplay(repoRoot, cleaned); ok {
		return display, true
	}

	evaluated, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return "", false
	}
	return formatReviewEvidenceRepoRelativePathDisplay(repoRoot, filepath.Clean(evaluated))
}

func formatReviewEvidenceRepoRelativePathDisplay(repoRoot, candidate string) (string, bool) {
	inside, err := isPathWithinRepoRoot(repoRoot, candidate)
	if err != nil || !inside {
		return "", false
	}

	rel, err := filepath.Rel(repoRoot, candidate)
	if err != nil {
		return "", false
	}
	if rel == "." || rel == "" {
		return ".", true
	}
	return filepath.ToSlash(rel), true
}

func isReviewEvidenceWindowsAbsolutePath(candidate string) bool {
	if len(candidate) >= 3 && isReviewEvidenceASCIILetter(candidate[0]) && candidate[1] == ':' && (candidate[2] == '\\' || candidate[2] == '/') {
		return true
	}
	return strings.HasPrefix(candidate, `\`)
}

func isReviewEvidenceASCIILetter(value byte) bool {
	return ('a' <= value && value <= 'z') || ('A' <= value && value <= 'Z')
}
