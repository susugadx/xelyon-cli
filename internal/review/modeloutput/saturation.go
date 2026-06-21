package modeloutput

import (
	"fmt"

	reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

// SaturationCheckModelOutputInput は model の saturation check JSON 出力を確定する入力。
type SaturationCheckModelOutputInput struct {
	Content         string
	Plan            reviewprobeplan.ReviewProbePlan
	FinalizedReport reviewreport.ReviewReport
	ExternalDocs    []externaldoc.Evidence
}

// FinalizeSaturationCheckModelOutput は model の saturation check JSON を strict decode し、external_doc refs を照合する。
func FinalizeSaturationCheckModelOutput(input SaturationCheckModelOutputInput) (reviewreport.ReviewSaturationCheck, error) {
	check, err := reviewreport.DecodeReviewSaturationCheckJSON(
		[]byte(input.Content),
		reviewanalysis.PlanScopeFromProbePlan(input.Plan),
		input.FinalizedReport,
	)
	if err != nil {
		return reviewreport.ReviewSaturationCheck{}, fmt.Errorf("review runner decode saturation check: %w", err)
	}
	if err := reviewanalysis.ValidateSaturationExternalDocRefs(check, input.ExternalDocs); err != nil {
		return reviewreport.ReviewSaturationCheck{}, fmt.Errorf("review runner decode saturation check: %w", err)
	}
	return check, nil
}
