package review

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
)

func TestCompactReviewWebSearchDiscoveryEvidenceKeepsUnsafeExternalSupport(t *testing.T) {
	longSearchSnippet := strings.Repeat("RAW_DISCOVERY_SNIPPET_MUST_STAY_WITH_UNSAFE_EXTERNAL_SUPPORT ", 400)
	tests := []struct {
		name     string
		evidence externaldoc.WebSearchEvidence
	}{
		{
			name: "unknown source credibility",
			evidence: reviewWebSearchDiscoveryCompactEvidenceForTest(longSearchSnippet, []externaldoc.Evidence{
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-unknown", externaldoc.SourceCredibilityUnknown, false, "unknown source snippet"),
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-official", externaldoc.SourceCredibilityOfficialCandidate, false, "official source snippet"),
			}),
		},
		{
			name: "truncated external doc",
			evidence: reviewWebSearchDiscoveryCompactEvidenceForTest(longSearchSnippet, []externaldoc.Evidence{
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-truncated", externaldoc.SourceCredibilityOfficialCandidate, true, "truncated source snippet"),
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-official", externaldoc.SourceCredibilityOfficialCandidate, false, "official source snippet"),
			}),
		},
		{
			name: "official confirmation false",
			evidence: reviewWebSearchDiscoveryCompactEvidenceForTest(longSearchSnippet, []externaldoc.Evidence{
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-single", externaldoc.SourceCredibilityOfficialCandidate, false, "single official snippet"),
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle := newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")
			bundle.WebSearchEvidence = tt.evidence
			compacted, savedBytes, savedTokens, ok := reviewpromptreduction.CompactReviewWebSearchDiscoveryEvidence(bundle)
			if ok || savedBytes != 0 || savedTokens != 0 || compacted.WebSearchEvidence.Enabled {
				t.Fatalf("reviewpromptreduction.CompactReviewWebSearchDiscoveryEvidence() = (%#v, %d, %d, %t), want conservative keep", compacted.WebSearchEvidence, savedBytes, savedTokens, ok)
			}
		})
	}
}
