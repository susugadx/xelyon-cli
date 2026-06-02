package review

import (
	reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
)

func buildReviewAnalysisEvidenceInput(input ReviewEvidenceModelInput) reviewanalysis.EvidenceInput {
	return reviewevidence.BuildReviewAnalysisEvidenceInput(input)
}
