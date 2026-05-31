package evidence

import (
	"fmt"
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

		file := readReviewEvidenceRegularFile(reviewEvidenceRegularFileReadInput{
			repoRoot:     repoRoot,
			absPath:      absPath,
			relPath:      relPath,
			maxBytes:     limits.MaxRuleFileBytes,
			allowSymlink: true,
		})
		switch file.status {
		case reviewEvidenceRegularFileReadMissing:
			continue
		case reviewEvidenceRegularFileReadStatFailed:
			return nil, fmt.Errorf("failed to stat rule file %q: %w", relPath, file.err)
		case reviewEvidenceRegularFileReadInvalidPath:
			return nil, file.err
		case reviewEvidenceRegularFileReadDir, reviewEvidenceRegularFileReadNonRegular, reviewEvidenceRegularFileReadSymlink:
			continue
		case reviewEvidenceRegularFileReadFailed:
			return nil, fmt.Errorf("failed to read rule file %q: %w", relPath, file.err)
		case reviewEvidenceRegularFileReadOK:
		default:
			continue
		}
		files = append(files, ReviewRuleFileEvidence{
			Path:      relPath,
			Content:   string(file.data),
			Truncated: file.truncated,
			SizeBytes: file.sizeBytes,
		})
	}
	return files, nil
}
