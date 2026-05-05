package review

import (
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
)

func resolveReviewEvidenceDirs(repoRoot, cwd string) (string, string, error) {
	// repoRoot は Git 実行と path validation の基準、cwd は診断用に保持する起動位置。
	resolvedRepoRoot, err := resolveReviewEvidenceDir(repoRoot, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve repo root: %w", err)
	}
	resolvedCWD, err := resolveReviewEvidenceDir(cwd, resolvedRepoRoot)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve cwd: %w", err)
	}
	canonicalRepoRoot, err := canonicalReviewEvidenceRepoRoot(resolvedRepoRoot)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve repo root symlinks: %w", err)
	}
	return canonicalRepoRoot, resolvedCWD, nil
}

func canonicalReviewEvidenceRepoRoot(repoRoot string) (string, error) {
	evaluated, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return "", err
	}
	return filepath.Clean(evaluated), nil
}

func resolveReviewEvidenceDir(candidate, fallback string) (string, error) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		trimmed = fallback
	}
	if trimmed == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		trimmed = cwd
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func resolveReviewEvidenceRepoPathLexically(repoRoot, candidate string) (string, string, error) {
	relPath, err := normalizeReviewEvidenceRelativePath(candidate)
	if err != nil {
		return "", "", err
	}
	absPath := filepath.Clean(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
	if err := validateReviewEvidencePathWithinRepoRoot(repoRoot, absPath, relPath); err != nil {
		return "", "", err
	}
	return absPath, relPath, nil
}

func validateReviewEvidenceRelativePaths(repoRoot string, paths []string, label string) error {
	for _, path := range paths {
		absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, path)
		if err != nil {
			return fmt.Errorf("invalid %s %q: %w", label, path, err)
		}
		if err := validateReviewEvidencePathWithinRepoRoot(repoRoot, absPath, relPath); err != nil {
			return fmt.Errorf("invalid %s %q: %w", label, path, err)
		}
	}
	return nil
}

func normalizeReviewEvidenceRelativePath(candidate string) (string, error) {
	if candidate == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(candidate) || pathpkg.IsAbs(candidate) {
		return "", fmt.Errorf("absolute path is not allowed")
	}

	cleaned := filepath.Clean(filepath.FromSlash(candidate))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("path is empty")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository root")
	}
	return filepath.ToSlash(cleaned), nil
}

func validateReviewEvidencePathWithinRepoRoot(repoRoot, absPath, label string) error {
	if !filepath.IsAbs(absPath) {
		return fmt.Errorf("%s is not absolute after resolution", label)
	}
	inside, err := isPathWithinRepoRoot(repoRoot, filepath.Clean(absPath))
	if err != nil {
		return fmt.Errorf("failed to validate %s: %w", label, err)
	}
	if !inside {
		return fmt.Errorf("%s is outside repository root", label)
	}
	return nil
}

func validateReviewEvidenceExistingPath(repoRoot, absPath, label string) error {
	evaluated, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", label, err)
	}
	inside, err := isPathWithinRepoRoot(repoRoot, filepath.Clean(evaluated))
	if err != nil {
		return fmt.Errorf("failed to validate %s symlink target: %w", label, err)
	}
	if !inside {
		return fmt.Errorf("%s resolves outside repository root", label)
	}
	return nil
}
