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

	displayRepoRoot, ok := normalizeReviewEvidencePathDisplayRepoRoot(repoRoot)
	if !ok {
		return reviewEvidenceOutsideRepoPathDisplay
	}

	if isReviewEvidenceWindowsAbsolutePath(linkTarget) && !filepath.IsAbs(linkTarget) {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	if filepath.IsAbs(linkTarget) {
		if display, ok := formatReviewEvidenceAbsolutePathDisplay(displayRepoRoot, linkTarget); ok {
			return display
		}
		return reviewEvidenceOutsideRepoPathDisplay
	}

	symlinkDisplayPath := formatReviewEvidencePathDisplayWithRepoRoot(displayRepoRoot, file.Path)
	if symlinkDisplayPath == reviewEvidenceOutsideRepoPathDisplay {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	symlinkParent := filepath.Dir(filepath.FromSlash(symlinkDisplayPath))
	resolvedTarget := filepath.Clean(filepath.Join(displayRepoRoot, symlinkParent, filepath.FromSlash(linkTarget)))
	if display, ok := formatReviewEvidenceRepoRelativePathDisplay(displayRepoRoot, resolvedTarget); ok {
		return display
	}
	return reviewEvidenceOutsideRepoPathDisplay
}
