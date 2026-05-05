package review

import "path/filepath"

const reviewEvidenceTruncatedLinkTargetDisplay = "<truncated-link-target>"

func formatReviewEvidenceSymlinkTargetDisplay(repoRoot string, file ReviewUntrackedFile) string {
	if !file.Symlink {
		return ""
	}
	linkTarget := file.LinkTarget
	if linkTarget == "" {
		return ""
	}
	if file.Truncated {
		return reviewEvidenceTruncatedLinkTargetDisplay
	}

	canonicalRepoRoot, ok := canonicalReviewEvidencePathDisplayRepoRoot(repoRoot)
	if !ok {
		return reviewEvidenceOutsideRepoPathDisplay
	}

	if isReviewEvidenceWindowsAbsolutePath(linkTarget) && !filepath.IsAbs(linkTarget) {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	if filepath.IsAbs(linkTarget) {
		if display, ok := formatReviewEvidenceAbsolutePathDisplay(canonicalRepoRoot, linkTarget); ok {
			return display
		}
		return reviewEvidenceOutsideRepoPathDisplay
	}

	symlinkDisplayPath := formatReviewEvidencePathDisplayWithCanonicalRepoRoot(canonicalRepoRoot, file.Path)
	if symlinkDisplayPath == reviewEvidenceOutsideRepoPathDisplay {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	symlinkParent := filepath.Dir(filepath.FromSlash(symlinkDisplayPath))
	resolvedTarget := filepath.Clean(filepath.Join(canonicalRepoRoot, symlinkParent, filepath.FromSlash(linkTarget)))
	if display, ok := formatReviewEvidenceRepoRelativePathDisplay(canonicalRepoRoot, resolvedTarget); ok {
		return display
	}
	return reviewEvidenceOutsideRepoPathDisplay
}
