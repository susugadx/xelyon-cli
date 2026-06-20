package promptreduction

import (
	"crypto/sha256"
	"encoding/hex"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

func cloneReviewEvidenceBundleForPromptCompact(bundle reviewevidence.ReviewEvidenceBundle) reviewevidence.ReviewEvidenceBundle {
	clone := bundle
	clone.ChangedFiles = append([]reviewevidence.ReviewChangedFile(nil), bundle.ChangedFiles...)
	clone.ChangedFileContext = append([]reviewevidence.ReviewContextFileEvidence(nil), bundle.ChangedFileContext...)
	clone.RelatedContextFiles = append([]reviewevidence.ReviewContextFileEvidence(nil), bundle.RelatedContextFiles...)
	clone.RelatedSearchHits = append([]reviewevidence.ReviewRelatedSearchHit(nil), bundle.RelatedSearchHits...)
	clone.GenericImpactCandidatePaths = append([]string(nil), bundle.GenericImpactCandidatePaths...)
	clone.GenericImpactCandidates.Tokens = append([]string(nil), bundle.GenericImpactCandidates.Tokens...)
	clone.GenericImpactCandidates.Candidates = append([]reviewevidence.ReviewGenericImpactCandidate(nil), bundle.GenericImpactCandidates.Candidates...)
	clone.UntrackedFiles = append([]reviewevidence.ReviewUntrackedFile(nil), bundle.UntrackedFiles...)
	clone.RuleFiles = append([]reviewevidence.ReviewRuleFileEvidence(nil), bundle.RuleFiles...)
	clone.Diffs = append([]reviewevidence.ReviewDiffEvidence(nil), bundle.Diffs...)
	clone.Inventory = cloneReviewChangeInventory(bundle.Inventory)
	clone.WebSearchEvidence = cloneReviewWebSearchEvidenceForPromptCompact(bundle.WebSearchEvidence)
	return clone
}

func cloneReviewChangeInventory(inventory reviewevidence.ReviewChangeInventory) reviewevidence.ReviewChangeInventory {
	return reviewevidence.ReviewChangeInventory{
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

func cloneReviewWebSearchEvidenceForPromptCompact(evidence externaldoc.WebSearchEvidence) externaldoc.WebSearchEvidence {
	clone := evidence
	clone.Queries = append([]externaldoc.WebSearchEvidenceQuery(nil), evidence.Queries...)
	for i := range clone.Queries {
		clone.Queries[i].Results = append([]externaldoc.WebSearchEvidenceResult(nil), evidence.Queries[i].Results...)
	}
	clone.ExternalDocs = append([]externaldoc.Evidence(nil), evidence.ExternalDocs...)
	for i := range clone.ExternalDocs {
		clone.ExternalDocs[i].Snippets = append([]externaldoc.SnippetEvidence(nil), evidence.ExternalDocs[i].Snippets...)
	}
	return clone
}

// ReviewPromptShortHash は provider-facing placeholder 用の短い content hash を返す。
func ReviewPromptShortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])[:16]
}
