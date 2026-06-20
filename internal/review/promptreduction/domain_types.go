package promptreduction

import (
	reviewdomain "github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

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

// TargetCurrentChanges は現在の作業ツリー差分を review 対象にする。
const TargetCurrentChanges = reviewdomain.TargetCurrentChanges

// ReviewEvidenceBundle は prompt reduction が読み取る evidence bundle を表す。
type ReviewEvidenceBundle = reviewevidence.ReviewEvidenceBundle

// ReviewChangedFile は prompt reduction が状態要約に使う変更ファイル情報を表す。
type ReviewChangedFile = reviewevidence.ReviewChangedFile

// ReviewContextFileEvidence は prompt reduction が保持する context file evidence を表す。
type ReviewContextFileEvidence = reviewevidence.ReviewContextFileEvidence

// ReviewRelatedSearchHit は prompt reduction が保持する related search hit を表す。
type ReviewRelatedSearchHit = reviewevidence.ReviewRelatedSearchHit

// ReviewGenericImpactCandidate は prompt reduction が保持する generic impact candidate を表す。
type ReviewGenericImpactCandidate = reviewevidence.ReviewGenericImpactCandidate

// ReviewUntrackedFile は prompt reduction が保持する untracked file evidence を表す。
type ReviewUntrackedFile = reviewevidence.ReviewUntrackedFile

// ReviewRuleFileEvidence は prompt reduction が保持する rule file evidence を表す。
type ReviewRuleFileEvidence = reviewevidence.ReviewRuleFileEvidence

// ReviewDiffEvidence は prompt reduction が保持する diff evidence を表す。
type ReviewDiffEvidence = reviewevidence.ReviewDiffEvidence

// ReviewChangeInventory は prompt reduction が保持する change inventory を表す。
type ReviewChangeInventory = reviewevidence.ReviewChangeInventory

// ReviewWebSearchEvidence は prompt reduction が保持する web search evidence を表す。
type ReviewWebSearchEvidence = externaldoc.WebSearchEvidence

// ReviewWebSearchEvidenceQuery は prompt reduction が保持する web search query evidence を表す。
type ReviewWebSearchEvidenceQuery = externaldoc.WebSearchEvidenceQuery

// ReviewWebSearchEvidenceResult は prompt reduction が保持する web search result evidence を表す。
type ReviewWebSearchEvidenceResult = externaldoc.WebSearchEvidenceResult

// ReviewExternalDocEvidence は prompt reduction が保持する external doc evidence を表す。
type ReviewExternalDocEvidence = externaldoc.Evidence

// ReviewExternalDocSnippetEvidence は prompt reduction が保持する external doc snippet evidence を表す。
type ReviewExternalDocSnippetEvidence = externaldoc.SnippetEvidence

// ReviewExternalDocSourceCredibility は prompt reduction が扱う external doc credibility を表す。
type ReviewExternalDocSourceCredibility = externaldoc.SourceCredibility

// ReviewExternalDocSourceCredibilityOfficialCandidate は official candidate credibility を表す。
const ReviewExternalDocSourceCredibilityOfficialCandidate = externaldoc.SourceCredibilityOfficialCandidate

// ReviewProbePlan は prompt reduction が参照する probe plan schema を表す。
type ReviewProbePlan = reviewprobeplan.ReviewProbePlan

// ReviewProbeCandidateRiskStatus は prompt reduction が保持する probe candidate risk status を表す。
type ReviewProbeCandidateRiskStatus = reviewprobeplan.ReviewProbeCandidateRiskStatus

const (
	// ReviewProbeCandidateRiskNeedsProbe は probe が必要な candidate risk を表す。
	ReviewProbeCandidateRiskNeedsProbe = reviewprobeplan.ReviewProbeCandidateRiskNeedsProbe
	// ReviewProbeCandidateRiskUnverified は未検証の candidate risk を表す。
	ReviewProbeCandidateRiskUnverified = reviewprobeplan.ReviewProbeCandidateRiskUnverified
)

// ReviewReport は prompt reduction が参照する review report schema を表す。
type ReviewReport = reviewreport.ReviewReport

// ReviewProbeSummary は prompt reduction が参照する probe summary を表す。
type ReviewProbeSummary = reviewreport.ReviewProbeSummary

// ReviewSaturationCheck は prompt reduction が参照する saturation check schema を表す。
type ReviewSaturationCheck = reviewreport.ReviewSaturationCheck

// ReviewEvidenceRef は prompt reduction が保持する report evidence ref を表す。
type ReviewEvidenceRef = reviewreport.ReviewEvidenceRef

const (
	// ReviewEvidenceKindExternalDoc は external doc evidence ref kind を表す。
	ReviewEvidenceKindExternalDoc = reviewreport.ReviewEvidenceKindExternalDoc
	// ReviewReportCandidateRiskUnverified は未検証 candidate risk を表す。
	ReviewReportCandidateRiskUnverified = reviewreport.ReviewReportCandidateRiskUnverified
	// ReviewReportCandidateRiskResidualRisk は residual risk candidate を表す。
	ReviewReportCandidateRiskResidualRisk = reviewreport.ReviewReportCandidateRiskResidualRisk
	// ReviewReportCandidateRiskDismissed は dismissed candidate risk を表す。
	ReviewReportCandidateRiskDismissed = reviewreport.ReviewReportCandidateRiskDismissed
	// ReviewReportImpactSurfaceChecked は checked impact surface を表す。
	ReviewReportImpactSurfaceChecked = reviewreport.ReviewReportImpactSurfaceChecked
	// ReviewEvidenceKindProbe は probe evidence ref kind を表す。
	ReviewEvidenceKindProbe = reviewreport.ReviewEvidenceKindProbe
	// ReviewEvidenceKindProbeCommand は probe command evidence ref kind を表す。
	ReviewEvidenceKindProbeCommand = reviewreport.ReviewEvidenceKindProbeCommand
)

// ReviewCommandIndex は command index の明示値を返す。
func ReviewCommandIndex(i int) *int {
	return reviewreport.ReviewCommandIndex(i)
}
