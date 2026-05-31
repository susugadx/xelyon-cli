package review

import (
	reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

const (
	// ReviewReportSchemaVersionV1 は旧 `/review` report schema v1 の識別子。
	ReviewReportSchemaVersionV1 = reviewreport.ReviewReportSchemaVersionV1
	// ReviewReportSchemaVersionV2 は `/review` report schema v2 の識別子。
	ReviewReportSchemaVersionV2 = reviewreport.ReviewReportSchemaVersionV2
	// ReviewReportSkeletonBlockedSummary は skeleton report の blocked reason 既定文。
	ReviewReportSkeletonBlockedSummary = reviewreport.ReviewReportSkeletonBlockedSummary
)

// ReviewVerdict は review report の最終判定。
type ReviewVerdict = reviewreport.ReviewVerdict

const (
	// ReviewVerdictClean は finding がない最終判定を表す。
	ReviewVerdictClean = reviewreport.ReviewVerdictClean
	// ReviewVerdictHasFindings は finding がある最終判定を表す。
	ReviewVerdictHasFindings = reviewreport.ReviewVerdictHasFindings
	// ReviewVerdictBlocked は review が完了できなかった最終判定を表す。
	ReviewVerdictBlocked = reviewreport.ReviewVerdictBlocked
)

// ReviewVerificationStatus は検証状態の明示表現。
type ReviewVerificationStatus = reviewreport.ReviewVerificationStatus

const (
	// ReviewVerificationVerified は検証済み状態を表す。
	ReviewVerificationVerified = reviewreport.ReviewVerificationVerified
	// ReviewVerificationPartiallyVerified は一部検証済み状態を表す。
	ReviewVerificationPartiallyVerified = reviewreport.ReviewVerificationPartiallyVerified
	// ReviewVerificationUnverified は未検証状態を表す。
	ReviewVerificationUnverified = reviewreport.ReviewVerificationUnverified
	// ReviewVerificationNotApplicable は検証対象外状態を表す。
	ReviewVerificationNotApplicable = reviewreport.ReviewVerificationNotApplicable
	// ReviewVerificationBlockedOrInconclusive は検証不能または不確定状態を表す。
	ReviewVerificationBlockedOrInconclusive = reviewreport.ReviewVerificationBlockedOrInconclusive
)

// ReviewGroupSeverity は root-cause group 単位の重要度。
type ReviewGroupSeverity = reviewreport.ReviewGroupSeverity

const (
	// ReviewGroupSeverityCritical は critical severity を表す。
	ReviewGroupSeverityCritical = reviewreport.ReviewGroupSeverityCritical
	// ReviewGroupSeverityHigh は high severity を表す。
	ReviewGroupSeverityHigh = reviewreport.ReviewGroupSeverityHigh
	// ReviewGroupSeverityMedium は medium severity を表す。
	ReviewGroupSeverityMedium = reviewreport.ReviewGroupSeverityMedium
	// ReviewGroupSeverityLow は low severity を表す。
	ReviewGroupSeverityLow = reviewreport.ReviewGroupSeverityLow
	// ReviewGroupSeverityInfo は info severity を表す。
	ReviewGroupSeverityInfo = reviewreport.ReviewGroupSeverityInfo
)

// ReviewReport は `/review` の最終 report schema を表す。
type ReviewReport = reviewreport.ReviewReport

// ReviewReportComputedSummary は runner が validation 後に算出する派生 count を表す。
type ReviewReportComputedSummary = reviewreport.ReviewReportComputedSummary

// ReviewRootCauseGroup は同一根本原因に紐づく finding 群をまとめる。
type ReviewRootCauseGroup = reviewreport.ReviewRootCauseGroup

// ReviewFinding は root-cause group 内の個別所見を表す。
type ReviewFinding = reviewreport.ReviewFinding

// ReviewEvidenceRef は finding/surface/risk を支える根拠参照を表す。
type ReviewEvidenceRef = reviewreport.ReviewEvidenceRef

const (
	// ReviewEvidenceKindProbeCommand は probe command evidence ref を表す。
	ReviewEvidenceKindProbeCommand = reviewreport.ReviewEvidenceKindProbeCommand
	// ReviewEvidenceKindProbe は probe evidence ref を表す。
	ReviewEvidenceKindProbe = reviewreport.ReviewEvidenceKindProbe
	// ReviewEvidenceKindFile は file evidence ref を表す。
	ReviewEvidenceKindFile = reviewreport.ReviewEvidenceKindFile
	// ReviewEvidenceKindDiff は diff evidence ref を表す。
	ReviewEvidenceKindDiff = reviewreport.ReviewEvidenceKindDiff
	// ReviewEvidenceKindGitStatus は git status evidence ref を表す。
	ReviewEvidenceKindGitStatus = reviewreport.ReviewEvidenceKindGitStatus
	// ReviewEvidenceKindRuleFile は rule file evidence ref を表す。
	ReviewEvidenceKindRuleFile = reviewreport.ReviewEvidenceKindRuleFile
	// ReviewEvidenceKindExternalDoc は external doc evidence ref を表す。
	ReviewEvidenceKindExternalDoc = reviewreport.ReviewEvidenceKindExternalDoc
)

// ReviewCommandIndex は command index の明示値を返す。
func ReviewCommandIndex(i int) *int {
	return reviewreport.ReviewCommandIndex(i)
}

// ReviewSurfaceCoverage は確認済み/未確認の surface を構造化して保持する。
type ReviewSurfaceCoverage = reviewreport.ReviewSurfaceCoverage

// ReviewReportScopeCoverage は Pass1 で列挙した scope を Pass2 がどう処理したかを表す。
type ReviewReportScopeCoverage = reviewreport.ReviewReportScopeCoverage

// ReviewReportImpactSurfaceStatus は Pass2 での impact surface 処理結果。
type ReviewReportImpactSurfaceStatus = reviewreport.ReviewReportImpactSurfaceStatus

const (
	// ReviewReportImpactSurfaceChecked は checked surface coverage を表す。
	ReviewReportImpactSurfaceChecked = reviewreport.ReviewReportImpactSurfaceChecked
	// ReviewReportImpactSurfaceFinding は finding surface coverage を表す。
	ReviewReportImpactSurfaceFinding = reviewreport.ReviewReportImpactSurfaceFinding
	// ReviewReportImpactSurfaceUnverified は unverified surface coverage を表す。
	ReviewReportImpactSurfaceUnverified = reviewreport.ReviewReportImpactSurfaceUnverified
	// ReviewReportImpactSurfaceResidualRisk は residual-risk surface coverage を表す。
	ReviewReportImpactSurfaceResidualRisk = reviewreport.ReviewReportImpactSurfaceResidualRisk
)

// ReviewReportCandidateRiskStatus は Pass2 での candidate risk 処理結果。
type ReviewReportCandidateRiskStatus = reviewreport.ReviewReportCandidateRiskStatus

const (
	// ReviewReportCandidateRiskDismissed は dismissed candidate risk coverage を表す。
	ReviewReportCandidateRiskDismissed = reviewreport.ReviewReportCandidateRiskDismissed
	// ReviewReportCandidateRiskFinding は finding candidate risk coverage を表す。
	ReviewReportCandidateRiskFinding = reviewreport.ReviewReportCandidateRiskFinding
	// ReviewReportCandidateRiskUnverified は unverified candidate risk coverage を表す。
	ReviewReportCandidateRiskUnverified = reviewreport.ReviewReportCandidateRiskUnverified
	// ReviewReportCandidateRiskResidualRisk は residual-risk candidate risk coverage を表す。
	ReviewReportCandidateRiskResidualRisk = reviewreport.ReviewReportCandidateRiskResidualRisk
)

// ReviewReportImpactSurfaceCoverage は Pass1 impact surface 1 件の Pass2 処理結果。
type ReviewReportImpactSurfaceCoverage = reviewreport.ReviewReportImpactSurfaceCoverage

// ReviewReportCandidateRiskCoverage は Pass1 candidate risk 1 件の Pass2 処理結果。
type ReviewReportCandidateRiskCoverage = reviewreport.ReviewReportCandidateRiskCoverage

// ReviewReportPassFindingCoverage は Pass1 scope 外で Pass2 が新たに見つけた finding 接続。
type ReviewReportPassFindingCoverage = reviewreport.ReviewReportPassFindingCoverage

// ReviewResidualRisk は未解消リスクを表す。
type ReviewResidualRisk = reviewreport.ReviewResidualRisk

// ReviewProbeSummary は ReviewProbeResult から report 用に切り出した要約。
type ReviewProbeSummary = reviewreport.ReviewProbeSummary

// ReviewProbeCommandSummary は report 用の probe command 要約。
type ReviewProbeCommandSummary = reviewreport.ReviewProbeCommandSummary

const (
	// ReviewSaturationCheckSchemaVersionV1 は runner 内部の final report saturation check schema。
	ReviewSaturationCheckSchemaVersionV1 = reviewreport.ReviewSaturationCheckSchemaVersionV1
)

// ReviewSaturationStatus は final report が Pass1 scope を十分に処理したかを表す。
type ReviewSaturationStatus = reviewreport.ReviewSaturationStatus

const (
	// ReviewSaturationStatusSaturated は final report が Pass1 scope を十分に処理した状態を表す。
	ReviewSaturationStatusSaturated = reviewreport.ReviewSaturationStatusSaturated
	// ReviewSaturationStatusNeedsRevision は final report の再生成が必要な状態を表す。
	ReviewSaturationStatusNeedsRevision = reviewreport.ReviewSaturationStatusNeedsRevision
	// ReviewSaturationStatusBlocked は saturation check が block された状態を表す。
	ReviewSaturationStatusBlocked = reviewreport.ReviewSaturationStatusBlocked
)

// ReviewSaturationCheck は runner 内部で使う final report 漏れ検査 DTO。
type ReviewSaturationCheck = reviewreport.ReviewSaturationCheck

// ReviewSaturationAdditionalFindingCandidate は revision に渡す追加 finding 候補。
type ReviewSaturationAdditionalFindingCandidate = reviewreport.ReviewSaturationAdditionalFindingCandidate

// DecodeReviewReportJSON は strict JSON として ReviewReport を decode して検証する。
func DecodeReviewReportJSON(data []byte) (ReviewReport, error) {
	return reviewreport.DecodeReviewReportJSON(data)
}

func decodeReviewReportModelStrictJSON(data []byte) (ReviewReport, error) {
	return reviewreport.DecodeReviewReportModelStrictJSON(data)
}

// ValidateReviewReport は schema v2 の review report 契約を検証する。
func ValidateReviewReport(report ReviewReport) error {
	return reviewreport.ValidateReviewReport(report)
}

// ValidateReviewReportAgainstProbePlan は Pass2 report が Pass1 scope と trusted probe outcome を閉じていることを検証する。
func ValidateReviewReportAgainstProbePlan(report ReviewReport, plan ReviewProbePlan, trustedProbeSummaries []ReviewProbeSummary) error {
	return reviewreport.ValidateReviewReportAgainstPlanScope(report, reviewanalysis.PlanScopeFromProbePlan(plan), trustedProbeSummaries)
}

// ComputeReviewReportComputedSummary は validated report と runner trusted probe summaries から
// final report 用の派生 count を算出する。
func ComputeReviewReportComputedSummary(report ReviewReport, probeSummaries []ReviewProbeSummary) ReviewReportComputedSummary {
	return reviewreport.ComputeReviewReportComputedSummary(report, probeSummaries)
}

// DecodeReviewSaturationCheckJSON は strict JSON として saturation check を decode して検証する。
func DecodeReviewSaturationCheckJSON(data []byte, plan ReviewProbePlan, finalizedReport ReviewReport) (ReviewSaturationCheck, error) {
	return reviewreport.DecodeReviewSaturationCheckJSON(data, reviewanalysis.PlanScopeFromProbePlan(plan), finalizedReport)
}

// ValidateReviewSaturationCheck は runner 内部の final report saturation check 契約を検証する。
func ValidateReviewSaturationCheck(check ReviewSaturationCheck, plan ReviewProbePlan, finalizedReport ReviewReport) error {
	return reviewreport.ValidateReviewSaturationCheck(check, reviewanalysis.PlanScopeFromProbePlan(plan), finalizedReport)
}
