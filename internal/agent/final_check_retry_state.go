package agent

import (
	"strings"
)

const finalCheckNoProgressThreshold = 2

// finalCheckRetryState は final_checks.commands の失敗再試行が
// 実際に進捗しているかだけを判定する。
type finalCheckRetryState struct {
	lastFailureFingerprint string
	lastChangeFingerprint  string
	repeatCount            int
}

func (s *finalCheckRetryState) reset() {
	if s == nil {
		return
	}
	s.lastFailureFingerprint = ""
	s.lastChangeFingerprint = ""
	s.repeatCount = 0
}

func (s *finalCheckRetryState) recordFailure(result finalCheckRunResult, changeFingerprint string) bool {
	if s == nil {
		return false
	}

	failureFingerprint := strings.TrimSpace(result.failureFingerprint)
	if failureFingerprint == "" {
		s.reset()
		return false
	}

	changeFingerprint = strings.TrimSpace(changeFingerprint)
	if changeFingerprint == "" {
		s.lastFailureFingerprint = failureFingerprint
		s.lastChangeFingerprint = ""
		s.repeatCount = 1
		return false
	}

	if failureFingerprint == s.lastFailureFingerprint && changeFingerprint == s.lastChangeFingerprint {
		s.repeatCount++
	} else {
		s.lastFailureFingerprint = failureFingerprint
		s.lastChangeFingerprint = changeFingerprint
		s.repeatCount = 1
	}

	return s.repeatCount >= finalCheckNoProgressThreshold
}
