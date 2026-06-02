package evidence

import (
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

		remainingTotal := limits.MaxTotalUntrackedBytes - totalRead
		maxBytes := minReviewEvidenceInt64(limits.MaxUntrackedFileBytes, remainingTotal)
		if maxBytes <= 0 {
			evidence.SnapshotsTruncated = i < len(paths)
			break
		}

		file := readReviewEvidenceRegularFile(reviewEvidenceRegularFileReadInput{
			repoRoot: repoRoot,
			absPath:  absPath,
			relPath:  relPath,
			maxBytes: maxBytes,
		})
		switch file.status {
		case reviewEvidenceRegularFileReadMissing, reviewEvidenceRegularFileReadStatFailed:
			if file.stage == reviewEvidenceRegularFileReadStageStat {
				return reviewUntrackedEvidence{}, fmt.Errorf("failed to stat untracked file %q: %w", relPath, file.err)
			}
			return reviewUntrackedEvidence{}, fmt.Errorf("failed to stat untracked path %q: %w", relPath, file.err)
		case reviewEvidenceRegularFileReadInvalidPath:
			return reviewUntrackedEvidence{}, file.err
		case reviewEvidenceRegularFileReadDir, reviewEvidenceRegularFileReadNonRegular:
			continue
		case reviewEvidenceRegularFileReadSymlink:
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
		case reviewEvidenceRegularFileReadFailed:
			return reviewUntrackedEvidence{}, fmt.Errorf("failed to read untracked file %q: %w", relPath, file.err)
		case reviewEvidenceRegularFileReadOK:
		default:
			continue
		}

		totalBudgetLimited := maxBytes == remainingTotal
		if totalBudgetLimited && file.sizeBytes > maxBytes {
			evidence.SnapshotsTruncated = true
		}
		totalRead += file.readBytes

		snapshot := ""
		if !file.binary {
			snapshot = string(file.data)
		}
		evidence.Files = append(evidence.Files, ReviewUntrackedFile{
			Path:      relPath,
			Snapshot:  snapshot,
			Binary:    file.binary,
			Truncated: file.truncated,
			SizeBytes: file.sizeBytes,
			ReadBytes: file.readBytes,
		})
	}

	return evidence, nil
}
