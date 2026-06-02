package probe

import reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"

// ReviewGroupSeverity は probe plan candidate risk の重要度を表す。
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

// ReviewEvidenceRef は probe plan の pre-probe evidence 参照を表す。
type ReviewEvidenceRef = reviewreport.ReviewEvidenceRef

// ReviewProbeSummary は ReviewProbeResult から report 用に切り出した要約。
type ReviewProbeSummary = reviewreport.ReviewProbeSummary

// ReviewProbeCommandSummary は report 用の probe command 要約。
type ReviewProbeCommandSummary = reviewreport.ReviewProbeCommandSummary

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

func validateEvidenceRef(field string, ref ReviewEvidenceRef, probeSummariesByID map[string]reviewreport.ReviewProbeSummary) error {
	return reviewreport.ValidateEvidenceRef(field, ref, probeSummariesByID)
}

func isKnownReviewGroupSeverity(severity ReviewGroupSeverity) bool {
	return reviewreport.IsKnownReviewGroupSeverity(severity)
}

func isKnownReviewEvidenceKind(kind string) bool {
	return reviewreport.IsKnownReviewEvidenceKind(kind)
}
