package evidence

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/pathpolicy"
)

const (
	// OutsideRepoPathDisplay は repo root 外の path を LLM 入力へ出すときの固定表現。
	OutsideRepoPathDisplay = "<outside-repo>"

	reviewEvidenceOutsideRepoPathDisplay = OutsideRepoPathDisplay
)

// FormatReviewEvidencePathDisplay は repo root 基準の安全な path 表示へ正規化する。
func FormatReviewEvidencePathDisplay(repoRoot, candidate string) string {
	return formatReviewEvidencePathDisplay(repoRoot, candidate)
}

// IsReviewEvidenceWindowsAbsolutePath は Windows absolute path 表現かを返す。
func IsReviewEvidenceWindowsAbsolutePath(candidate string) bool {
	return isReviewEvidenceWindowsAbsolutePath(candidate)
}

func formatReviewEvidencePathDisplay(repoRoot, candidate string) string {
	displayRepoRoot, ok := normalizeReviewEvidencePathDisplayRepoRoot(repoRoot)
	if !ok {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	return formatReviewEvidencePathDisplayWithRepoRoot(displayRepoRoot, candidate)
}

func normalizeReviewEvidencePathDisplayRepoRoot(repoRoot string) (string, bool) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", false
	}
	if isReviewEvidenceWindowsAbsolutePath(repoRoot) && !filepath.IsAbs(repoRoot) {
		return "", false
	}
	cleaned := filepath.Clean(filepath.FromSlash(repoRoot))
	return cleaned, cleaned != "."
}

func formatReviewEvidencePathDisplayWithRepoRoot(repoRoot, candidate string) string {
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

	display, ok := formatReviewEvidenceAbsolutePathDisplay(repoRoot, candidate)
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
	return "", false
}

func formatReviewEvidenceRepoRelativePathDisplay(repoRoot, candidate string) (string, bool) {
	inside, err := pathpolicy.IsWithinRoot(repoRoot, candidate)
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
	if len(candidate) >= 3 && isReviewASCIIAlpha(candidate[0]) && candidate[1] == ':' && (candidate[2] == '\\' || candidate[2] == '/') {
		return true
	}
	return strings.HasPrefix(candidate, `\`)
}

func isReviewASCIIAlpha(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}
