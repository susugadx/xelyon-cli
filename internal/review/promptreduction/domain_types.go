package promptreduction

// ReviewModelPhase は prompt reduction が記録する review model phase。
type ReviewModelPhase string

const (
	// ReviewModelPhaseReport は Pass2 の final report 生成を表す。
	ReviewModelPhaseReport ReviewModelPhase = "report"
	// ReviewModelPhaseSaturationCheck は final report 後の漏れ検査を表す。
	ReviewModelPhaseSaturationCheck ReviewModelPhase = "saturation_check"
	// ReviewModelPhaseReportRevision は saturation 指摘に基づく report 再生成を表す。
	ReviewModelPhaseReportRevision ReviewModelPhase = "report_revision"
)
