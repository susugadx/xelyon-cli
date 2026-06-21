package report

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func canonicalizeReviewProbeSummaryMutationOutcome(summary ReviewProbeSummary) ReviewProbeSummary {
	if !isReviewProbeSummaryMutationOutcome(summary) {
		return summary
	}
	summary.Status = domain.ReviewProbeMutatedWorktree
	summary.MutatedWorktree = true
	return summary
}

// CanonicalizeReviewProbeSummaryMutationOutcome は mutation outcome の内部表現を揃える。
func CanonicalizeReviewProbeSummaryMutationOutcome(summary ReviewProbeSummary) ReviewProbeSummary {
	return canonicalizeReviewProbeSummaryMutationOutcome(summary)
}

// CanonicalizeReviewProbeSummaryMutationOutcomes は summary slice 内の mutation outcome を揃える。
func CanonicalizeReviewProbeSummaryMutationOutcomes(summaries []ReviewProbeSummary) {
	for i := range summaries {
		summaries[i] = canonicalizeReviewProbeSummaryMutationOutcome(summaries[i])
	}
}

func isReviewProbeSummaryMutationOutcome(summary ReviewProbeSummary) bool {
	return summary.Status == domain.ReviewProbeMutatedWorktree || summary.MutatedWorktree
}

// IsReviewProbeSummaryMutationOutcome は probe summary が worktree mutation outcome かを返す。
func IsReviewProbeSummaryMutationOutcome(summary ReviewProbeSummary) bool {
	return isReviewProbeSummaryMutationOutcome(summary)
}

// CopyReviewProbeSummaries は trusted probe summary を report 注入用に deep copy する。
func CopyReviewProbeSummaries(summaries []ReviewProbeSummary) []ReviewProbeSummary {
	if len(summaries) == 0 {
		return nil
	}

	copied := make([]ReviewProbeSummary, len(summaries))
	for i, summary := range summaries {
		copied[i] = summary
		copied[i].MutatedFiles = copyReviewProbeSummaryStringSlice(summary.MutatedFiles)
		copied[i].Commands = copyReviewProbeCommandSummaries(summary.Commands)
	}
	return copied
}

// NormalizeReviewReportForTrustedProbeOutcomes は blocked/timeout probe を含む report の検証状態を正規化する。
func NormalizeReviewReportForTrustedProbeOutcomes(report ReviewReport) ReviewReport {
	if !hasBlockedReviewProbeSummary(report.ProbeSummaries) {
		return report
	}

	switch report.Verdict {
	case ReviewVerdictClean:
		return report
	case ReviewVerdictHasFindings:
		if report.OverallVerificationStatus == ReviewVerificationVerified {
			report.OverallVerificationStatus = ReviewVerificationPartiallyVerified
		}
	case ReviewVerdictBlocked:
		if report.OverallVerificationStatus == ReviewVerificationVerified || report.OverallVerificationStatus == ReviewVerificationNotApplicable {
			report.OverallVerificationStatus = ReviewVerificationBlockedOrInconclusive
		}
	}
	return report
}

func hasBlockedReviewProbeSummary(summaries []ReviewProbeSummary) bool {
	for _, summary := range summaries {
		if isReviewProbeSummaryMutationOutcome(summary) {
			return true
		}
		switch summary.Status {
		case domain.ReviewProbeBlocked, domain.ReviewProbeTimedOut:
			return true
		}
	}
	return false
}

func copyReviewProbeCommandSummaries(summaries []ReviewProbeCommandSummary) []ReviewProbeCommandSummary {
	if summaries == nil {
		return nil
	}

	copied := make([]ReviewProbeCommandSummary, len(summaries))
	for i, summary := range summaries {
		copied[i] = summary
		copied[i].Args = copyReviewProbeSummaryStringSlice(summary.Args)
	}
	return copied
}

func copyReviewProbeSummaryStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}

func validateReviewProbePlanID(field, candidate string) (string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", fmt.Errorf("%s must be non-empty", field)
	}
	if strings.TrimSpace(candidate) != candidate {
		return "", fmt.Errorf("%s must be canonical ID without leading/trailing whitespace: got %q", field, candidate)
	}
	if containsAnyWhitespace(candidate) {
		return "", fmt.Errorf("%s must not include whitespace: got %q", field, candidate)
	}
	for _, r := range candidate {
		if !isReviewProbePlanIDRune(r) {
			return "", fmt.Errorf("%s must contain only ASCII letters, digits, hyphen, or underscore: got %q", field, candidate)
		}
	}
	return candidate, nil
}

func isReviewProbePlanIDRune(r rune) bool {
	return ('a' <= r && r <= 'z') ||
		('A' <= r && r <= 'Z') ||
		('0' <= r && r <= '9') ||
		r == '-' ||
		r == '_'
}
