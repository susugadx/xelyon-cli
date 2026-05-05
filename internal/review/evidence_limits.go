package review

import "time"

const (
	defaultReviewEvidenceMaxCommandOutputBytes  = 1024 * 1024
	defaultReviewEvidenceMaxUntrackedFileBytes  = 64 * 1024
	defaultReviewEvidenceMaxRuleFileBytes       = 64 * 1024
	defaultReviewEvidenceMaxTotalUntrackedBytes = 256 * 1024
	defaultReviewEvidenceMaxUntrackedFiles      = 100
	defaultReviewEvidenceCommandTimeout         = 30 * time.Second
)

// DefaultReviewEvidenceLimits は EvidenceBuilder の既定 resource budget を返す。
func DefaultReviewEvidenceLimits() ReviewEvidenceLimits {
	return ReviewEvidenceLimits{
		MaxCommandOutputBytes:  defaultReviewEvidenceMaxCommandOutputBytes,
		MaxUntrackedFileBytes:  defaultReviewEvidenceMaxUntrackedFileBytes,
		MaxRuleFileBytes:       defaultReviewEvidenceMaxRuleFileBytes,
		MaxTotalUntrackedBytes: defaultReviewEvidenceMaxTotalUntrackedBytes,
		MaxUntrackedFiles:      defaultReviewEvidenceMaxUntrackedFiles,
		CommandTimeout:         defaultReviewEvidenceCommandTimeout,
	}
}

func normalizeReviewEvidenceLimits(limits ReviewEvidenceLimits) ReviewEvidenceLimits {
	defaults := DefaultReviewEvidenceLimits()
	if limits.MaxCommandOutputBytes <= 0 {
		limits.MaxCommandOutputBytes = defaults.MaxCommandOutputBytes
	}
	if limits.MaxUntrackedFileBytes <= 0 {
		limits.MaxUntrackedFileBytes = defaults.MaxUntrackedFileBytes
	}
	if limits.MaxRuleFileBytes <= 0 {
		limits.MaxRuleFileBytes = defaults.MaxRuleFileBytes
	}
	if limits.MaxTotalUntrackedBytes <= 0 {
		limits.MaxTotalUntrackedBytes = defaults.MaxTotalUntrackedBytes
	}
	if limits.MaxUntrackedFiles <= 0 {
		limits.MaxUntrackedFiles = defaults.MaxUntrackedFiles
	}
	if limits.CommandTimeout <= 0 {
		limits.CommandTimeout = defaults.CommandTimeout
	}
	return limits
}
