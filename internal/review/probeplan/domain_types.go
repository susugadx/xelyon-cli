package probeplan

import (
	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

// TargetKind は review 対象の種類を表す。
type TargetKind = domain.TargetKind

const (
	// TargetCurrentChanges は現在の作業ツリー差分を review 対象にする。
	TargetCurrentChanges = domain.TargetCurrentChanges
)

// ReviewProbeMode は probe plan schema 上の probe 実行 mode を表す。
type ReviewProbeMode = domain.ReviewProbeMode

const (
	// ReviewProbeHostReadOnly は元 repo を read-only bind した process sandbox で実行する。
	ReviewProbeHostReadOnly = domain.ReviewProbeHostReadOnly
	// ReviewProbeScratchOnly は repo 外 scratch だけを書き込み可能にした process sandbox で実行する。
	ReviewProbeScratchOnly = domain.ReviewProbeScratchOnly
	// ReviewProbeRepoSandbox は元 repo の現在状態を一時 worktree へコピーし、copy 側だけを bind して実行する。
	ReviewProbeRepoSandbox = domain.ReviewProbeRepoSandbox
)

func isKnownReviewProbeMode(mode ReviewProbeMode) bool {
	return domain.IsKnownReviewProbeMode(mode)
}

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
