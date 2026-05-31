package report

import (
	"fmt"
	"strings"
)

func canonicalizeReviewProbeSummaryMutationOutcome(summary ReviewProbeSummary) ReviewProbeSummary {
	if !isReviewProbeSummaryMutationOutcome(summary) {
		return summary
	}
	summary.Status = ReviewProbeMutatedWorktree
	summary.MutatedWorktree = true
	return summary
}

func isReviewProbeSummaryMutationOutcome(summary ReviewProbeSummary) bool {
	return summary.Status == ReviewProbeMutatedWorktree || summary.MutatedWorktree
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
