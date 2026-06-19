package evidence

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

// BuildReviewEvidenceModelInput は ReviewEvidenceBundle を LLM 入力 DTO に変換する。
// この変換は LLM 入力 schema の owner であり、evidence の収集方針は扱わない。
func BuildReviewEvidenceModelInput(bundle ReviewEvidenceBundle) ReviewEvidenceModelInput {
	repoRoot := bundle.RepoRoot

	return ReviewEvidenceModelInput{
		TargetKind: bundle.TargetKind,
		RepoRoot:   reviewEvidenceRepoRootPathDisplay,
		CWDDisplay: formatReviewEvidencePathDisplay(repoRoot, bundle.CWD),
		GitStatusShort: ReviewEvidenceTextBlock{
			Content:   bundle.StatusShort,
			Truncated: bundle.StatusShortTruncated,
		},
		ChangeInventory:     buildReviewEvidenceChangeInventoryInput(repoRoot, bundle.Inventory),
		ChangedFiles:        buildReviewEvidenceChangedFileInputs(repoRoot, bundle.ChangedFiles),
		ChangedFileContext:  buildReviewEvidenceContextFileInputs(repoRoot, bundle.ChangedFileContext),
		RelatedContextFiles: buildReviewEvidenceContextFileInputs(repoRoot, bundle.RelatedContextFiles),
		RelatedSearchHits:   buildReviewEvidenceRelatedSearchHitInputs(repoRoot, bundle.RelatedSearchHits),
		GenericImpact:       buildReviewEvidenceGenericImpactInput(repoRoot, bundle.GenericImpactCandidates),
		WebSearchEvidence:   bundle.WebSearchEvidence,
		ExternalSupport:     externaldoc.SummarizeExternalSupport(bundle.WebSearchEvidence),
		RuleFiles:           buildReviewEvidenceRuleFileInputs(repoRoot, bundle.RuleFiles),
		Diffs:               buildReviewEvidenceDiffInputs(bundle.Diffs),
		UntrackedFiles:      buildReviewEvidenceUntrackedFileInputs(repoRoot, bundle.UntrackedFiles),
		Limits:              buildReviewEvidenceLimitsInput(bundle.Limits),
		TruncationFlags:     buildReviewEvidenceTruncationFlagsInput(repoRoot, bundle),
	}
}

func buildReviewEvidenceChangeInventoryInput(repoRoot string, inventory ReviewChangeInventory) ReviewEvidenceChangeInventoryInput {
	return ReviewEvidenceChangeInventoryInput{
		Generated:    formatReviewEvidencePathDisplays(repoRoot, inventory.Generated),
		Tests:        formatReviewEvidencePathDisplays(repoRoot, inventory.Tests),
		Docs:         formatReviewEvidencePathDisplays(repoRoot, inventory.Docs),
		Config:       formatReviewEvidencePathDisplays(repoRoot, inventory.Config),
		Production:   formatReviewEvidencePathDisplays(repoRoot, inventory.Production),
		NewFiles:     formatReviewEvidencePathDisplays(repoRoot, inventory.NewFiles),
		DeletedFiles: formatReviewEvidencePathDisplays(repoRoot, inventory.DeletedFiles),
		RenamedFiles: formatReviewEvidencePathDisplays(repoRoot, inventory.RenamedFiles),
		Untracked:    formatReviewEvidencePathDisplays(repoRoot, inventory.Untracked),
	}
}

func buildReviewEvidenceChangedFileInputs(repoRoot string, files []ReviewChangedFile) []ReviewEvidenceChangedFileInput {
	result := make([]ReviewEvidenceChangedFileInput, 0, len(files))
	for _, file := range files {
		result = append(result, ReviewEvidenceChangedFileInput{
			Path:     formatReviewEvidencePathDisplay(repoRoot, file.Path),
			OldPath:  formatReviewEvidenceOptionalPathDisplay(repoRoot, file.OldPath),
			Status:   file.Status,
			Staged:   file.Staged,
			Unstaged: file.Unstaged,
		})
	}
	return result
}

func buildReviewEvidenceContextFileInputs(repoRoot string, files []ReviewContextFileEvidence) []ReviewEvidenceContextFileInput {
	result := make([]ReviewEvidenceContextFileInput, 0, len(files))
	for _, file := range files {
		result = append(result, ReviewEvidenceContextFileInput{
			Path:       formatReviewEvidencePathDisplay(repoRoot, file.Path),
			Role:       file.Role,
			Content:    file.Content,
			Truncated:  file.Truncated,
			Skipped:    file.Skipped,
			SkipReason: file.SkipReason,
			SizeBytes:  file.SizeBytes,
			ReadBytes:  file.ReadBytes,
		})
	}
	return result
}

func buildReviewEvidenceRelatedSearchHitInputs(repoRoot string, hits []ReviewRelatedSearchHit) []ReviewEvidenceRelatedSearchHitInput {
	result := make([]ReviewEvidenceRelatedSearchHitInput, 0, len(hits))
	for _, hit := range hits {
		result = append(result, ReviewEvidenceRelatedSearchHitInput{
			Path:    formatReviewEvidencePathDisplay(repoRoot, hit.Path),
			Line:    hit.Line,
			Snippet: hit.Snippet,
			Reason:  hit.Reason,
		})
	}
	return result
}

func buildReviewEvidenceGenericImpactInput(repoRoot string, generic ReviewGenericImpactCandidates) ReviewEvidenceGenericImpactInput {
	candidates := make([]ReviewEvidenceGenericImpactCandidateInput, 0, len(generic.Candidates))
	for _, candidate := range generic.Candidates {
		candidates = append(candidates, ReviewEvidenceGenericImpactCandidateInput{
			Path:    formatReviewEvidencePathDisplay(repoRoot, candidate.Path),
			Role:    candidate.Role,
			Reason:  redactReviewEvidenceRepoRootInText(repoRoot, candidate.Reason),
			Token:   candidate.Token,
			Line:    candidate.Line,
			Snippet: redactReviewEvidenceRepoRootInText(repoRoot, candidate.Snippet),
		})
	}
	return ReviewEvidenceGenericImpactInput{
		Tokens:     append([]string{}, generic.Tokens...),
		Candidates: candidates,
		Truncated:  generic.Truncated,
	}
}

func redactReviewEvidenceRepoRootInText(repoRoot, text string) string {
	displayRepoRoot, ok := normalizeReviewEvidencePathDisplayRepoRoot(repoRoot)
	if !ok || text == "" {
		return text
	}
	redacted := text
	for _, variant := range ReviewEvidencePathReplacementVariants(displayRepoRoot) {
		redacted = strings.ReplaceAll(redacted, variant, reviewEvidenceRepoRootPathDisplay)
	}
	return redacted
}

func buildReviewEvidenceRuleFileInputs(repoRoot string, files []ReviewRuleFileEvidence) []ReviewEvidenceRuleFileInput {
	result := make([]ReviewEvidenceRuleFileInput, 0, len(files))
	for _, file := range files {
		result = append(result, ReviewEvidenceRuleFileInput{
			Path:      formatReviewEvidencePathDisplay(repoRoot, file.Path),
			Content:   file.Content,
			Truncated: file.Truncated,
			SizeBytes: file.SizeBytes,
		})
	}
	return result
}

func buildReviewEvidenceDiffInputs(diffs []ReviewDiffEvidence) []ReviewEvidenceDiffInput {
	result := make([]ReviewEvidenceDiffInput, 0, len(diffs))
	for _, diff := range diffs {
		result = append(result, ReviewEvidenceDiffInput{
			Source: diff.Source,
			Stat: ReviewEvidenceTextBlock{
				Content:   diff.Stat,
				Truncated: diff.StatTruncated,
			},
			NameStatus: ReviewEvidenceTextBlock{
				Content:   diff.NameStatus,
				Truncated: diff.NameStatusTruncated,
			},
			Diff: ReviewEvidenceTextBlock{
				Content:   diff.Diff,
				Truncated: diff.DiffTruncated,
			},
		})
	}
	return result
}

func buildReviewEvidenceUntrackedFileInputs(repoRoot string, files []ReviewUntrackedFile) []ReviewEvidenceUntrackedFileInput {
	result := make([]ReviewEvidenceUntrackedFileInput, 0, len(files))
	for _, file := range files {
		result = append(result, ReviewEvidenceUntrackedFileInput{
			Path:       formatReviewEvidencePathDisplay(repoRoot, file.Path),
			Symlink:    file.Symlink,
			LinkTarget: formatReviewEvidenceSymlinkTargetDisplay(repoRoot, file),
			Snapshot:   file.Snapshot,
			Binary:     file.Binary,
			Truncated:  file.Truncated,
			SizeBytes:  file.SizeBytes,
			ReadBytes:  file.ReadBytes,
		})
	}
	return result
}
