package evidence

import (
	"path/filepath"
	"strings"
)

// ReviewEvidencePathReplacementVariants は path redaction で照合する slash/native 表現を返す。
func ReviewEvidencePathReplacementVariants(path string) []string {
	cleaned := normalizeReviewEvidenceReplacementPath(path)
	if cleaned == "" {
		return nil
	}

	slashPath := reviewEvidenceSlashPath(cleaned)
	variants := []string{slashPath}
	if isReviewEvidenceWindowsAbsolutePath(cleaned) || isReviewEvidenceWindowsAbsolutePath(slashPath) {
		nativePath := strings.ReplaceAll(slashPath, "/", `\`)
		if nativePath != slashPath {
			variants = append(variants, nativePath)
		}
	}
	return variants
}

func normalizeReviewEvidenceReplacementPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.Clean(filepath.FromSlash(path))
}

func reviewEvidenceSlashPath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), `\`, "/")
}
