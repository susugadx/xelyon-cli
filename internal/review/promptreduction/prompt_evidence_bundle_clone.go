package promptreduction

import (
	"crypto/sha256"
	"encoding/hex"
)

func cloneReviewEvidenceBundleForPromptCompact(bundle ReviewEvidenceBundle) ReviewEvidenceBundle {
	clone := bundle
	clone.ChangedFiles = append([]ReviewChangedFile(nil), bundle.ChangedFiles...)
	clone.ChangedFileContext = append([]ReviewContextFileEvidence(nil), bundle.ChangedFileContext...)
	clone.RelatedContextFiles = append([]ReviewContextFileEvidence(nil), bundle.RelatedContextFiles...)
	clone.RelatedSearchHits = append([]ReviewRelatedSearchHit(nil), bundle.RelatedSearchHits...)
	clone.GenericImpactCandidatePaths = append([]string(nil), bundle.GenericImpactCandidatePaths...)
	clone.GenericImpactCandidates.Tokens = append([]string(nil), bundle.GenericImpactCandidates.Tokens...)
	clone.GenericImpactCandidates.Candidates = append([]ReviewGenericImpactCandidate(nil), bundle.GenericImpactCandidates.Candidates...)
	clone.UntrackedFiles = append([]ReviewUntrackedFile(nil), bundle.UntrackedFiles...)
	clone.RuleFiles = append([]ReviewRuleFileEvidence(nil), bundle.RuleFiles...)
	clone.Diffs = append([]ReviewDiffEvidence(nil), bundle.Diffs...)
	clone.Inventory = cloneReviewChangeInventory(bundle.Inventory)
	clone.WebSearchEvidence = cloneReviewWebSearchEvidenceForPromptCompact(bundle.WebSearchEvidence)
	return clone
}

func cloneReviewChangeInventory(inventory ReviewChangeInventory) ReviewChangeInventory {
	return ReviewChangeInventory{
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

func cloneReviewWebSearchEvidenceForPromptCompact(evidence ReviewWebSearchEvidence) ReviewWebSearchEvidence {
	clone := evidence
	clone.Queries = append([]ReviewWebSearchEvidenceQuery(nil), evidence.Queries...)
	for i := range clone.Queries {
		clone.Queries[i].Results = append([]ReviewWebSearchEvidenceResult(nil), evidence.Queries[i].Results...)
	}
	clone.ExternalDocs = append([]ReviewExternalDocEvidence(nil), evidence.ExternalDocs...)
	for i := range clone.ExternalDocs {
		clone.ExternalDocs[i].Snippets = append([]ReviewExternalDocSnippetEvidence(nil), evidence.ExternalDocs[i].Snippets...)
	}
	return clone
}

// ReviewPromptShortHash は provider-facing placeholder 用の短い content hash を返す。
func ReviewPromptShortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])[:16]
}
