package modelinput

import "unicode/utf8"

type reviewProbeResultPromptOutputLimiter struct {
	remainingBytes int
}

func newReviewProbeResultPromptOutputLimiter() *reviewProbeResultPromptOutputLimiter {
	return &reviewProbeResultPromptOutputLimiter{remainingBytes: maxReviewProbeResultPromptTotalOutputBytes}
}

func (l *reviewProbeResultPromptOutputLimiter) limit(output string) (string, bool) {
	if output == "" {
		return "", false
	}
	if l == nil {
		return output, false
	}
	if l.remainingBytes <= 0 {
		return reviewProbeResultPromptOutputOmittedMarker, true
	}

	limit := min(maxReviewProbeResultPromptCommandOutputBytes, l.remainingBytes)
	if len(output) <= limit {
		l.remainingBytes -= len(output)
		return output, false
	}

	truncated := truncateReviewProbeResultPromptOutput(output, limit)
	l.remainingBytes -= len(truncated)
	return truncated + reviewProbeResultPromptOutputTruncatedMarker, true
}

func truncateReviewProbeResultPromptOutput(output string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(output) <= limit {
		return output
	}

	truncated := output[:limit]
	for !utf8.ValidString(truncated) {
		_, size := utf8.DecodeLastRuneInString(truncated)
		if size <= 0 || size > len(truncated) {
			return ""
		}
		truncated = truncated[:len(truncated)-size]
	}
	return truncated
}
