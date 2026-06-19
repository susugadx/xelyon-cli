package review

import (
	"context"
)

func (r *ReviewRunner) collectPostPass1WebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle, plan ReviewProbePlan, evidenceMarkdown string, redactor reviewRunnerPromptRedactor) (ReviewEvidenceBundle, string, reviewRunnerPromptRedactor, reviewCoverageAuditContext) {
	if !bundle.WebSearchEvidence.Enabled {
		return bundle, evidenceMarkdown, redactor, reviewCoverageAuditContext{}
	}
	provider, ok := r.evidenceBuilder.(ReviewPostPass1WebSearchEvidenceProvider)
	if !ok {
		return bundle, evidenceMarkdown, redactor, buildReviewCoverageAuditContext(bundle.WebSearchEvidence, bundle)
	}
	before := bundle.WebSearchEvidence
	bundle.WebSearchEvidence = provider.CollectPostPass1WebSearchEvidence(ctx, bundle, plan)
	evidenceMarkdown = RenderReviewEvidenceMarkdown(bundle)
	redactor = newReviewRunnerPromptRedactor(bundle, nil)
	r.saveReviewRunTextArtifact("evidence_post_pass1.md", evidenceMarkdown, redactor)
	r.saveReviewRunJSONArtifact("web_search_evidence_post_pass1.json", bundle.WebSearchEvidence, redactor)
	return bundle, evidenceMarkdown, redactor, buildReviewCoverageAuditContext(before, bundle)
}
