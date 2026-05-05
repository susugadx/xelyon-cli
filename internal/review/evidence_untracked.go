package review

import (
	"bytes"
	"fmt"
	"os"
)

type reviewUntrackedEvidence struct {
	Files              []ReviewUntrackedFile
	SnapshotsTruncated bool
}

func buildReviewUntrackedFileEvidence(repoRoot string, paths []string, limits ReviewEvidenceLimits) (reviewUntrackedEvidence, error) {
	evidence := reviewUntrackedEvidence{
		Files: make([]ReviewUntrackedFile, 0, minReviewEvidenceInt(len(paths), limits.MaxUntrackedFiles)),
	}
	var totalRead int64

	for i, path := range paths {
		if len(evidence.Files) >= limits.MaxUntrackedFiles || totalRead >= limits.MaxTotalUntrackedBytes {
			evidence.SnapshotsTruncated = i < len(paths)
			break
		}

		absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, path)
		if err != nil {
			return reviewUntrackedEvidence{}, err
		}

		lstatInfo, err := os.Lstat(absPath)
		if err != nil {
			return reviewUntrackedEvidence{}, fmt.Errorf("failed to stat untracked path %q: %w", relPath, err)
		}
		if lstatInfo.IsDir() {
			continue
		}

		remainingTotal := limits.MaxTotalUntrackedBytes - totalRead
		maxBytes := minReviewEvidenceInt64(limits.MaxUntrackedFileBytes, remainingTotal)
		if maxBytes <= 0 {
			evidence.SnapshotsTruncated = i < len(paths)
			break
		}

		if lstatInfo.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(absPath)
			if err != nil {
				return reviewUntrackedEvidence{}, fmt.Errorf("failed to read untracked symlink %q: %w", relPath, err)
			}
			snapshot, truncated := truncateReviewEvidenceStringPrefix(linkTarget, maxBytes)
			totalBudgetLimited := maxBytes == remainingTotal
			if totalBudgetLimited && int64(len(linkTarget)) > maxBytes {
				evidence.SnapshotsTruncated = true
			}
			totalRead += int64(len(snapshot))

			evidence.Files = append(evidence.Files, ReviewUntrackedFile{
				Path:       relPath,
				Symlink:    true,
				LinkTarget: snapshot,
				Truncated:  truncated,
				SizeBytes:  int64(len(linkTarget)),
				ReadBytes:  int64(len(snapshot)),
			})
			continue
		}
		if !lstatInfo.Mode().IsRegular() {
			continue
		}
		if err := validateReviewEvidenceExistingPath(repoRoot, absPath, relPath); err != nil {
			return reviewUntrackedEvidence{}, err
		}

		statInfo, err := os.Stat(absPath)
		if err != nil {
			return reviewUntrackedEvidence{}, fmt.Errorf("failed to stat untracked file %q: %w", relPath, err)
		}
		if statInfo.IsDir() || !statInfo.Mode().IsRegular() {
			continue
		}

		data, truncated, err := readReviewEvidenceFilePrefix(absPath, maxBytes)
		if err != nil {
			return reviewUntrackedEvidence{}, fmt.Errorf("failed to read untracked file %q: %w", relPath, err)
		}
		totalBudgetLimited := maxBytes == remainingTotal
		truncated = truncated || statInfo.Size() > maxBytes
		if totalBudgetLimited && statInfo.Size() > maxBytes {
			evidence.SnapshotsTruncated = true
		}
		totalRead += int64(len(data))

		binary := bytes.IndexByte(data, 0) >= 0
		snapshot := ""
		if !binary {
			snapshot = string(data)
		}
		evidence.Files = append(evidence.Files, ReviewUntrackedFile{
			Path:      relPath,
			Snapshot:  snapshot,
			Binary:    binary,
			Truncated: truncated,
			SizeBytes: statInfo.Size(),
			ReadBytes: int64(len(data)),
		})
	}

	return evidence, nil
}
