package review

import (
	"path/filepath"
	"strings"
)

const reviewEvidenceOutsideRepoPathDisplay = "<outside-repo>"

func formatReviewEvidencePathDisplay(repoRoot, candidate string) string {
	if strings.TrimSpace(repoRoot) == "" {
		return reviewEvidenceOutsideRepoPathDisplay
	}

	resolvedRepoRoot, err := resolveReviewEvidenceDir(repoRoot, "")
	if err != nil {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	canonicalRepoRoot, err := canonicalReviewEvidenceRepoRoot(resolvedRepoRoot)
	if err != nil {
		return reviewEvidenceOutsideRepoPathDisplay
	}

	if isReviewEvidenceWindowsAbsolutePath(candidate) && !filepath.IsAbs(candidate) {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	if !filepath.IsAbs(candidate) {
		display, ok := formatReviewEvidenceRelativePathDisplay(candidate)
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

func formatReviewEvidenceRelativePathDisplay(candidate string) (string, bool) {
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
	if display, ok := formatReviewEvidencePathInsideRepoDisplay(repoRoot, cleaned); ok {
		return display, true
	}

	evaluated, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return "", false
	}
	return formatReviewEvidencePathInsideRepoDisplay(repoRoot, filepath.Clean(evaluated))
}

func formatReviewEvidencePathInsideRepoDisplay(repoRoot, candidate string) (string, bool) {
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
