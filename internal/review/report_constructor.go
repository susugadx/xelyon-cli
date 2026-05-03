package review

import "time"

// NewReviewReportSkeleton は schema v1 の最小 report 枠を構築する。
func NewReviewReportSkeleton(req ReviewRequest, generatedAt time.Time) ReviewReport {
	return ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV1,
		TargetKind:                req.TargetKind,
		CustomInstructions:        req.CustomInstructions,
		GeneratedAt:               generatedAt,
		OverallVerificationStatus: ReviewVerificationUnverified,
		RootCauseGroups:           make([]ReviewRootCauseGroup, 0),
		ProbeSummaries:            make([]ReviewProbeSummary, 0),
		CheckedSurfaces:           make([]ReviewSurfaceCoverage, 0),
		UnverifiedSurfaces:        make([]ReviewSurfaceCoverage, 0),
		ResidualRisks:             make([]ReviewResidualRisk, 0),
	}
}
