package evidence

import "time"

const (
	defaultReviewEvidenceMaxCommandOutputBytes      = 1024 * 1024
	defaultReviewEvidenceMaxUntrackedFileBytes      = 64 * 1024
	defaultReviewEvidenceMaxRuleFileBytes           = 64 * 1024
	defaultReviewEvidenceMaxTotalUntrackedBytes     = 256 * 1024
	defaultReviewEvidenceMaxUntrackedFiles          = 100
	defaultReviewEvidenceMaxContextFileBytes        = 32 * 1024
	defaultReviewEvidenceMaxTotalContextBytes       = 160 * 1024
	defaultReviewEvidenceMaxContextFiles            = 24
	defaultReviewEvidenceMaxRelatedSearchTerms      = 12
	defaultReviewEvidenceMaxRelatedSearchFiles      = 200
	defaultReviewEvidenceMaxTotalRelatedSearchBytes = 512 * 1024
	defaultReviewEvidenceMaxRelatedSearchFileBytes  = 64 * 1024
	defaultReviewEvidenceMaxRelatedSearchHits       = 40
	defaultReviewEvidenceMaxSearchSnippetBytes      = 240
	defaultReviewEvidenceCommandTimeout             = 30 * time.Second
)

// DefaultReviewEvidenceLimits は EvidenceBuilder の既定 resource budget を返す。
func DefaultReviewEvidenceLimits() ReviewEvidenceLimits {
	return ReviewEvidenceLimits{
		MaxCommandOutputBytes:      defaultReviewEvidenceMaxCommandOutputBytes,
		MaxUntrackedFileBytes:      defaultReviewEvidenceMaxUntrackedFileBytes,
		MaxRuleFileBytes:           defaultReviewEvidenceMaxRuleFileBytes,
		MaxTotalUntrackedBytes:     defaultReviewEvidenceMaxTotalUntrackedBytes,
		MaxUntrackedFiles:          defaultReviewEvidenceMaxUntrackedFiles,
		MaxContextFileBytes:        defaultReviewEvidenceMaxContextFileBytes,
		MaxTotalContextBytes:       defaultReviewEvidenceMaxTotalContextBytes,
		MaxContextFiles:            defaultReviewEvidenceMaxContextFiles,
		MaxRelatedSearchTerms:      defaultReviewEvidenceMaxRelatedSearchTerms,
		MaxRelatedSearchFiles:      defaultReviewEvidenceMaxRelatedSearchFiles,
		MaxTotalRelatedSearchBytes: defaultReviewEvidenceMaxTotalRelatedSearchBytes,
		MaxRelatedSearchFileBytes:  defaultReviewEvidenceMaxRelatedSearchFileBytes,
		MaxRelatedSearchHits:       defaultReviewEvidenceMaxRelatedSearchHits,
		MaxSearchSnippetBytes:      defaultReviewEvidenceMaxSearchSnippetBytes,
		CommandTimeout:             defaultReviewEvidenceCommandTimeout,
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
	if limits.MaxContextFileBytes <= 0 {
		limits.MaxContextFileBytes = defaults.MaxContextFileBytes
	}
	if limits.MaxTotalContextBytes <= 0 {
		limits.MaxTotalContextBytes = defaults.MaxTotalContextBytes
	}
	if limits.MaxContextFiles <= 0 {
		limits.MaxContextFiles = defaults.MaxContextFiles
	}
	if limits.MaxRelatedSearchTerms <= 0 {
		limits.MaxRelatedSearchTerms = defaults.MaxRelatedSearchTerms
	}
	if limits.MaxRelatedSearchFiles <= 0 {
		limits.MaxRelatedSearchFiles = defaults.MaxRelatedSearchFiles
	}
	if limits.MaxTotalRelatedSearchBytes <= 0 {
		limits.MaxTotalRelatedSearchBytes = defaults.MaxTotalRelatedSearchBytes
	}
	if limits.MaxRelatedSearchFileBytes <= 0 {
		limits.MaxRelatedSearchFileBytes = defaults.MaxRelatedSearchFileBytes
	}
	if limits.MaxRelatedSearchHits <= 0 {
		limits.MaxRelatedSearchHits = defaults.MaxRelatedSearchHits
	}
	if limits.MaxSearchSnippetBytes <= 0 {
		limits.MaxSearchSnippetBytes = defaults.MaxSearchSnippetBytes
	}
	if limits.CommandTimeout <= 0 {
		limits.CommandTimeout = defaults.CommandTimeout
	}
	return limits
}

func truncateReviewEvidenceStringPrefix(value string, maxBytes int64) (string, bool) {
	if maxBytes <= 0 {
		return "", value != ""
	}
	if int64(len(value)) <= maxBytes {
		return value, false
	}
	return value[:int(maxBytes)], true
}

func minReviewEvidenceInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minReviewEvidenceInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
