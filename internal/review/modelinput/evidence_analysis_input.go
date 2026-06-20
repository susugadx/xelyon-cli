package modelinput

import (
	reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

// BuildReviewPressureSignalInputs は model input から deterministic pressure signal を作る。
func BuildReviewPressureSignalInputs(input ReviewEvidenceModelInput) []reviewanalysis.PressureSignal {
	return reviewanalysis.BuildPressureSignals(BuildReviewAnalysisEvidenceInput(input), reviewanalysis.PressureSignalOptions{
		KnownRuleFilePaths: reviewevidence.KnownReviewRuleFilePaths(),
	})
}

// BuildReviewAnalysisEvidenceInput は ReviewEvidenceModelInput を analysis package 用 DTO へ変換する。
func BuildReviewAnalysisEvidenceInput(input ReviewEvidenceModelInput) reviewanalysis.EvidenceInput {
	return reviewanalysis.EvidenceInput{
		ChangeInventory:     buildReviewAnalysisChangeInventory(input.ChangeInventory),
		ChangedFiles:        buildReviewAnalysisChangedFiles(input.ChangedFiles),
		ChangedFileContext:  buildReviewAnalysisContextFiles(input.ChangedFileContext),
		RelatedContextFiles: buildReviewAnalysisContextFiles(input.RelatedContextFiles),
		RelatedSearchHits:   buildReviewAnalysisRelatedSearchHits(input.RelatedSearchHits),
		GenericImpact:       buildReviewAnalysisGenericImpact(input.GenericImpact),
		WebSearchEvidence:   buildReviewAnalysisWebSearchEvidence(input.WebSearchEvidence),
		Diffs:               buildReviewAnalysisDiffs(input.Diffs),
		UntrackedFiles:      buildReviewAnalysisUntrackedFiles(input.UntrackedFiles),
		TruncationFlags:     buildReviewAnalysisTruncationFlags(input.TruncationFlags),
	}
}

func buildReviewAnalysisChangedFiles(files []ReviewEvidenceChangedFileInput) []reviewanalysis.ChangedFile {
	result := make([]reviewanalysis.ChangedFile, 0, len(files))
	for _, file := range files {
		result = append(result, reviewanalysis.ChangedFile{
			Path:    file.Path,
			OldPath: file.OldPath,
		})
	}
	return result
}

func buildReviewAnalysisContextFiles(files []ReviewEvidenceContextFileInput) []reviewanalysis.ContextFile {
	result := make([]reviewanalysis.ContextFile, 0, len(files))
	for _, file := range files {
		result = append(result, reviewanalysis.ContextFile{
			Path:       file.Path,
			Role:       file.Role,
			Truncated:  file.Truncated,
			Skipped:    file.Skipped,
			SkipReason: file.SkipReason,
		})
	}
	return result
}

func buildReviewAnalysisRelatedSearchHits(hits []ReviewEvidenceRelatedSearchHitInput) []reviewanalysis.RelatedSearchHit {
	result := make([]reviewanalysis.RelatedSearchHit, 0, len(hits))
	for _, hit := range hits {
		result = append(result, reviewanalysis.RelatedSearchHit{
			Path: hit.Path,
		})
	}
	return result
}

func buildReviewAnalysisGenericImpact(generic ReviewEvidenceGenericImpactInput) reviewanalysis.GenericImpact {
	candidates := make([]reviewanalysis.GenericImpactCandidate, 0, len(generic.Candidates))
	for _, candidate := range generic.Candidates {
		candidates = append(candidates, reviewanalysis.GenericImpactCandidate{
			Path:  candidate.Path,
			Role:  candidate.Role,
			Token: candidate.Token,
		})
	}
	return reviewanalysis.GenericImpact{
		Tokens:     append([]string(nil), generic.Tokens...),
		Candidates: candidates,
		Truncated:  generic.Truncated,
	}
}

func buildReviewAnalysisChangeInventory(inventory ReviewEvidenceChangeInventoryInput) reviewanalysis.ChangeInventory {
	return reviewanalysis.ChangeInventory{
		Generated:    append([]string(nil), inventory.Generated...),
		Tests:        append([]string(nil), inventory.Tests...),
		Docs:         append([]string(nil), inventory.Docs...),
		Config:       append([]string(nil), inventory.Config...),
		Production:   append([]string(nil), inventory.Production...),
		NewFiles:     append([]string(nil), inventory.NewFiles...),
		DeletedFiles: append([]string(nil), inventory.DeletedFiles...),
		RenamedFiles: append([]string(nil), inventory.RenamedFiles...),
		Untracked:    append([]string(nil), inventory.Untracked...),
	}
}

func buildReviewAnalysisWebSearchEvidence(evidence externaldoc.WebSearchEvidence) externaldoc.WebSearchEvidence {
	result := evidence
	result.Queries = append([]externaldoc.WebSearchEvidenceQuery(nil), evidence.Queries...)
	result.ExternalDocs = append([]externaldoc.Evidence(nil), evidence.ExternalDocs...)
	return result
}

func buildReviewAnalysisDiffs(diffs []ReviewEvidenceDiffInput) []reviewanalysis.Diff {
	result := make([]reviewanalysis.Diff, 0, len(diffs))
	for _, diff := range diffs {
		result = append(result, reviewanalysis.Diff{
			Stat:       buildReviewAnalysisTextBlock(diff.Stat),
			NameStatus: buildReviewAnalysisTextBlock(diff.NameStatus),
			Diff:       buildReviewAnalysisTextBlock(diff.Diff),
		})
	}
	return result
}

func buildReviewAnalysisTextBlock(block ReviewEvidenceTextBlock) reviewanalysis.TextBlock {
	return reviewanalysis.TextBlock{
		Content:   block.Content,
		Truncated: block.Truncated,
	}
}

func buildReviewAnalysisUntrackedFiles(files []ReviewEvidenceUntrackedFileInput) []reviewanalysis.UntrackedFile {
	result := make([]reviewanalysis.UntrackedFile, 0, len(files))
	for _, file := range files {
		result = append(result, reviewanalysis.UntrackedFile{
			Path: file.Path,
		})
	}
	return result
}

func buildReviewAnalysisTruncationFlags(flags ReviewEvidenceTruncationFlagsInput) reviewanalysis.TruncationFlags {
	return reviewanalysis.TruncationFlags{
		StatusShort:         flags.StatusShort,
		Diffs:               buildReviewAnalysisDiffTruncations(flags.Diffs),
		UntrackedList:       flags.UntrackedList,
		RelatedCandidates:   flags.RelatedCandidates,
		RelatedSearch:       flags.RelatedSearch,
		WebSearchEvidence:   flags.WebSearchEvidence,
		UntrackedSnapshots:  flags.UntrackedSnapshots,
		UntrackedFiles:      buildReviewAnalysisPathTruncations(flags.UntrackedFiles),
		RuleFiles:           buildReviewAnalysisPathTruncations(flags.RuleFiles),
		ChangedFileContext:  buildReviewAnalysisPathTruncations(flags.ChangedFileContext),
		RelatedContextFiles: buildReviewAnalysisPathTruncations(flags.RelatedContextFiles),
	}
}

func buildReviewAnalysisDiffTruncations(flags []ReviewEvidenceDiffTruncationInput) []reviewanalysis.DiffTruncation {
	result := make([]reviewanalysis.DiffTruncation, 0, len(flags))
	for _, flag := range flags {
		result = append(result, reviewanalysis.DiffTruncation{
			Source:     flag.Source,
			Stat:       flag.Stat,
			NameStatus: flag.NameStatus,
			Diff:       flag.Diff,
		})
	}
	return result
}

func buildReviewAnalysisPathTruncations(flags []ReviewEvidencePathTruncationInput) []reviewanalysis.PathTruncation {
	result := make([]reviewanalysis.PathTruncation, 0, len(flags))
	for _, flag := range flags {
		result = append(result, reviewanalysis.PathTruncation{
			Path:      flag.Path,
			Truncated: flag.Truncated,
		})
	}
	return result
}
