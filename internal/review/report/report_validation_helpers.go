package report

import (
	"fmt"
	"strings"
)

func validateEvidencePath(field, candidate string) error {
	return validateReviewCanonicalRelativePath(field, candidate, reviewRelativePathValidationPolicy{
		pathKind:         "repo-relative path",
		rejectWhitespace: false,
	})
}

func validateRequiredProbeID(field, candidate string) (string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", fmt.Errorf("%s must be non-empty", field)
	}
	return validateOptionalProbeID(field, candidate)
}

func validateRequiredReportID(field, candidate string) (string, error) {
	if candidate == "" {
		return "", fmt.Errorf("%s must be non-empty", field)
	}
	return validateOptionalReportID(field, candidate)
}

func validateOptionalReportID(field, candidate string) (string, error) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		if candidate != "" {
			return "", fmt.Errorf("%s must be canonical report ID without whitespace: got %q", field, candidate)
		}
		return "", nil
	}
	if trimmed != candidate {
		return "", fmt.Errorf("%s must be canonical report ID without leading/trailing whitespace: got %q", field, candidate)
	}
	if containsAnyWhitespace(candidate) {
		return "", fmt.Errorf("%s must not include whitespace: got %q", field, candidate)
	}
	return candidate, nil
}

func validateOptionalProbeID(field, candidate string) (string, error) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		if candidate != "" {
			return "", fmt.Errorf("%s must be canonical probe_id without leading/trailing whitespace: got %q", field, candidate)
		}
		return "", nil
	}
	if trimmed != candidate {
		return "", fmt.Errorf("%s must be canonical probe_id without leading/trailing whitespace: got %q", field, candidate)
	}
	return candidate, nil
}

func validateOptionalEvidencePath(field, candidate string) (string, error) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		if candidate != "" {
			return "", fmt.Errorf("%s must be canonical repo-relative path without leading/trailing whitespace: got %q", field, candidate)
		}
		return "", nil
	}
	if trimmed != candidate {
		return "", fmt.Errorf("%s must be canonical repo-relative path without leading/trailing whitespace: got %q", field, candidate)
	}
	return candidate, nil
}

func isKnownReviewVerdict(verdict ReviewVerdict) bool {
	switch verdict {
	case ReviewVerdictClean, ReviewVerdictHasFindings, ReviewVerdictBlocked:
		return true
	default:
		return false
	}
}

func isKnownReviewVerificationStatus(status ReviewVerificationStatus) bool {
	switch status {
	case ReviewVerificationVerified,
		ReviewVerificationPartiallyVerified,
		ReviewVerificationUnverified,
		ReviewVerificationNotApplicable,
		ReviewVerificationBlockedOrInconclusive:
		return true
	default:
		return false
	}
}

// IsKnownReviewGroupSeverity は既知の group severity かを返す。
func IsKnownReviewGroupSeverity(severity ReviewGroupSeverity) bool {
	return isKnownReviewGroupSeverity(severity)
}

func isKnownReviewGroupSeverity(severity ReviewGroupSeverity) bool {
	for _, known := range reviewGroupSeverities {
		if severity == known {
			return true
		}
	}
	return false
}

// IsKnownReviewEvidenceKind は既知の evidence kind かを返す。
func IsKnownReviewEvidenceKind(kind string) bool {
	return isKnownReviewEvidenceKind(kind)
}

func isKnownReviewEvidenceKind(kind string) bool {
	for _, known := range reviewEvidenceKinds {
		if kind == known {
			return true
		}
	}
	return false
}
