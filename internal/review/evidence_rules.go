package review

import (
	"fmt"
	"os"
)

var reviewEvidenceRuleFilePaths = []string{
	"AGENTS.md",
	"CLAUDE.md",
	"xelyon.yaml",
	"Makefile",
	".codex/config.toml",
}

func buildReviewRuleFileEvidence(repoRoot string, limits ReviewEvidenceLimits) ([]ReviewRuleFileEvidence, error) {
	files := make([]ReviewRuleFileEvidence, 0, len(reviewEvidenceRuleFilePaths))
	for _, path := range reviewEvidenceRuleFilePaths {
		absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, path)
		if err != nil {
			return nil, err
		}

		lstatInfo, err := os.Lstat(absPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to stat rule file %q: %w", relPath, err)
		}
		if lstatInfo.IsDir() {
			continue
		}
		if err := validateReviewEvidenceExistingPath(repoRoot, absPath, relPath); err != nil {
			return nil, err
		}

		statInfo, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat rule file %q: %w", relPath, err)
		}
		if statInfo.IsDir() {
			continue
		}
		if !statInfo.Mode().IsRegular() {
			continue
		}

		data, truncated, err := readReviewEvidenceFilePrefix(absPath, limits.MaxRuleFileBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to read rule file %q: %w", relPath, err)
		}
		truncated = truncated || statInfo.Size() > limits.MaxRuleFileBytes
		files = append(files, ReviewRuleFileEvidence{
			Path:      relPath,
			Content:   string(data),
			Truncated: truncated,
			SizeBytes: statInfo.Size(),
		})
	}
	return files, nil
}
