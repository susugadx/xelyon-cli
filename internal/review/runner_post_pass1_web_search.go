package review

import (
	"context"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
)

func (r *ReviewRunner) collectPostPass1WebSearchEvidence(ctx context.Context, bundle reviewevidence.ReviewEvidenceBundle, plan reviewprobeplan.ReviewProbePlan, evidenceMarkdown string, redactor reviewRunnerPromptRedactor) (reviewevidence.ReviewEvidenceBundle, string, reviewRunnerPromptRedactor, reviewCoverageAuditContext) {
	if !bundle.WebSearchEvidence.Enabled {
		return bundle, evidenceMarkdown, redactor, reviewCoverageAuditContext{}
	}
	provider, ok := r.evidenceBuilder.(reviewevidence.ReviewPostPass1WebSearchEvidenceProvider)
	if !ok {
		return bundle, evidenceMarkdown, redactor, buildReviewCoverageAuditContext(bundle.WebSearchEvidence, bundle)
	}
	before := bundle.WebSearchEvidence
	bundle.WebSearchEvidence = provider.CollectPostPass1WebSearchEvidence(ctx, bundle, plan)
	evidenceMarkdown = reviewevidence.RenderReviewEvidenceMarkdown(bundle)
	redactor = newReviewRunnerPromptRedactor(bundle, nil)
	r.saveReviewRunTextArtifact("evidence_post_pass1.md", evidenceMarkdown, redactor)
	r.saveReviewRunJSONArtifact("web_search_evidence_post_pass1.json", bundle.WebSearchEvidence, redactor)
	return bundle, evidenceMarkdown, redactor, buildReviewCoverageAuditContext(before, bundle)
}
