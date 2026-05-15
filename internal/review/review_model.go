package review

import "context"

// ReviewModelPhase は ReviewModel への model 呼び出し用途を表す。
type ReviewModelPhase string

const (
	// ReviewModelPhaseProbePlan は Pass1 の probe plan 生成を表す。
	ReviewModelPhaseProbePlan ReviewModelPhase = "probe_plan"
	// ReviewModelPhaseReport は Pass2 の final report 生成を表す。
	ReviewModelPhaseReport ReviewModelPhase = "report"
	// ReviewModelPhaseSaturationCheck は final report 後の漏れ検査を表す。
	ReviewModelPhaseSaturationCheck ReviewModelPhase = "saturation_check"
	// ReviewModelPhaseReportRevision は saturation 指摘に基づく report 再生成を表す。
	ReviewModelPhaseReportRevision ReviewModelPhase = "report_revision"
)

// ReviewModel は ReviewRunner が使う最小の model 呼び出し契約を表す。
//
// provider、agent、TUI には依存しない。Prompt は次フェーズの ReviewRunner
// または prompt builder が構築済みの完全な model input で、Content は raw model
// response text として decoder に渡す。
type ReviewModel interface {
	CompleteReview(ctx context.Context, req ReviewModelRequest) (ReviewModelResponse, error)
}

// ReviewModelRequest は ReviewModel に渡す 1 回分の入力を表す。
type ReviewModelRequest struct {
	Phase  ReviewModelPhase
	Prompt string
}

// ReviewModelResponse は ReviewModel から返る raw response を表す。
type ReviewModelResponse struct {
	Content string
}
