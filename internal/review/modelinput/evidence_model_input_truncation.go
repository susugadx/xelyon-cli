package modelinput

import reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"

func buildReviewEvidenceLimitsInput(limits reviewevidence.ReviewEvidenceLimits) ReviewEvidenceLimitsInput {
	return ReviewEvidenceLimitsInput{
		MaxCommandOutputBytes:      limits.MaxCommandOutputBytes,
		MaxUntrackedFileBytes:      limits.MaxUntrackedFileBytes,
		MaxRuleFileBytes:           limits.MaxRuleFileBytes,
		MaxTotalUntrackedBytes:     limits.MaxTotalUntrackedBytes,
		MaxUntrackedFiles:          limits.MaxUntrackedFiles,
		MaxContextFileBytes:        limits.MaxContextFileBytes,
		MaxTotalContextBytes:       limits.MaxTotalContextBytes,
		MaxContextFiles:            limits.MaxContextFiles,
		MaxRelatedSearchTerms:      limits.MaxRelatedSearchTerms,
		MaxRelatedSearchFiles:      limits.MaxRelatedSearchFiles,
		MaxTotalRelatedSearchBytes: limits.MaxTotalRelatedSearchBytes,
		MaxRelatedSearchFileBytes:  limits.MaxRelatedSearchFileBytes,
		MaxRelatedSearchHits:       limits.MaxRelatedSearchHits,
		MaxSearchSnippetBytes:      limits.MaxSearchSnippetBytes,
		CommandTimeoutMS:           limits.CommandTimeout.Milliseconds(),
	}
}

func buildReviewEvidenceTruncationFlagsInput(repoRoot string, bundle reviewevidence.ReviewEvidenceBundle) ReviewEvidenceTruncationFlagsInput {
	return ReviewEvidenceTruncationFlagsInput{
		StatusShort:        bundle.StatusShortTruncated,
		Diffs:              buildReviewEvidenceDiffTruncationInputs(bundle.Diffs),
		UntrackedList:      bundle.UntrackedListTruncated,
		RelatedCandidates:  bundle.RelatedCandidateListTruncated,
		RelatedSearch:      bundle.RelatedSearchTruncated,
		WebSearchEvidence:  bundle.WebSearchEvidence.Truncated,
		UntrackedSnapshots: bundle.UntrackedSnapshotsTruncated,
		UntrackedFiles: buildReviewEvidencePathTruncationInputs(repoRoot, bundle.UntrackedFiles, func(file reviewevidence.ReviewUntrackedFile) (string, bool) {
			return file.Path, file.Truncated
		}),
		RuleFiles: buildReviewEvidencePathTruncationInputs(repoRoot, bundle.RuleFiles, func(file reviewevidence.ReviewRuleFileEvidence) (string, bool) {
			return file.Path, file.Truncated
		}),
		ChangedFileContext: buildReviewEvidencePathTruncationInputs(repoRoot, bundle.ChangedFileContext, func(file reviewevidence.ReviewContextFileEvidence) (string, bool) {
			return file.Path, file.Truncated
		}),
		RelatedContextFiles: buildReviewEvidencePathTruncationInputs(repoRoot, bundle.RelatedContextFiles, func(file reviewevidence.ReviewContextFileEvidence) (string, bool) {
			return file.Path, file.Truncated
		}),
	}
}

func buildReviewEvidenceDiffTruncationInputs(diffs []reviewevidence.ReviewDiffEvidence) []ReviewEvidenceDiffTruncationInput {
	result := make([]ReviewEvidenceDiffTruncationInput, 0, len(diffs))
	for _, diff := range diffs {
		result = append(result, ReviewEvidenceDiffTruncationInput{
			Source:     diff.Source,
			Stat:       diff.StatTruncated,
			NameStatus: diff.NameStatusTruncated,
			Diff:       diff.DiffTruncated,
		})
	}
	return result
}

func buildReviewEvidencePathTruncationInputs[T any](repoRoot string, files []T, mapper func(T) (string, bool)) []ReviewEvidencePathTruncationInput {
	result := make([]ReviewEvidencePathTruncationInput, 0, len(files))
	for _, file := range files {
		path, truncated := mapper(file)
		result = append(result, ReviewEvidencePathTruncationInput{
			Path:      reviewevidence.FormatReviewEvidencePathDisplay(repoRoot, path),
			Truncated: truncated,
		})
	}
	return result
}
