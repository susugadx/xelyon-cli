package probe

import (
	"fmt"
	"strings"
)

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

func validateReviewProbePlanRequiredText(field, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		return fmt.Errorf("%s must be non-empty", field)
	}
	return nil
}

func validateReviewProbePlanPurpose(field, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		return fmt.Errorf("%s must be non-empty", field)
	}
	if strings.TrimSpace(candidate) != candidate {
		return fmt.Errorf("%s must be canonical purpose without leading/trailing whitespace: got %q", field, candidate)
	}
	if len([]byte(candidate)) > MaxReviewProbePlanPurposeBytes {
		return fmt.Errorf("%s must be at most %d bytes", field, MaxReviewProbePlanPurposeBytes)
	}
	return nil
}

func isKnownReviewProbeImpactSurfaceCategory(category ReviewProbeImpactSurfaceCategory) bool {
	for _, known := range reviewProbeImpactSurfaceCategories {
		if category == known {
			return true
		}
	}
	return false
}

func isReviewProbePlanPreProbeEvidenceKind(kind string) bool {
	return IsReviewProbePlanPreProbeEvidenceKind(kind)
}

// IsReviewProbePlanPreProbeEvidenceKind は kind が probe 実行前 evidence 参照かを返す。
func IsReviewProbePlanPreProbeEvidenceKind(kind string) bool {
	if !isKnownReviewEvidenceKind(kind) {
		return false
	}
	switch kind {
	case ReviewEvidenceKindProbe, ReviewEvidenceKindProbeCommand:
		return false
	default:
		return true
	}
}

func isKnownReviewProbeImpactSurfaceStatus(status ReviewProbeImpactSurfaceStatus) bool {
	for _, known := range reviewProbeImpactSurfaceStatuses {
		if status == known {
			return true
		}
	}
	return false
}

func isKnownReviewProbeCandidateRiskStatus(status ReviewProbeCandidateRiskStatus) bool {
	for _, known := range reviewProbeCandidateRiskStatuses {
		if status == known {
			return true
		}
	}
	return false
}
