package review

import reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"

// ValidateReviewProbePlanAgainstEvidence は Pass1 probe plan が evidence bundle の
// material path と evidence pressure を扱っていることを検証する。
func ValidateReviewProbePlanAgainstEvidence(plan ReviewProbePlan, bundle ReviewEvidenceBundle) error {
	return reviewanalysis.ValidateProbePlanAgainstEvidence(plan, buildReviewAnalysisEvidenceInput(BuildReviewEvidenceModelInput(bundle)))
}
