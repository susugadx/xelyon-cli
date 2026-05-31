package finalcheck

import "strings"

const noProgressThreshold = 2

// RetryState は final_checks.commands の失敗再試行が実際に進捗しているかだけを判定する。
type RetryState struct {
	lastFailureFingerprint string
	lastChangeFingerprint  string
	repeatCount            int
}

// Reset は final check retry の進捗判定状態をリセットする。
func (s *RetryState) Reset() {
	if s == nil {
		return
	}
	s.lastFailureFingerprint = ""
	s.lastChangeFingerprint = ""
	s.repeatCount = 0
}

// RecordFailure は失敗結果と進捗 fingerprint を記録し、進捗なしの連続失敗なら true を返す。
func (s *RetryState) RecordFailure(result RunResult, progressFingerprint string) bool {
	if s == nil {
		return false
	}

	failureFingerprint := strings.TrimSpace(result.FailureFingerprint)
	if failureFingerprint == "" {
		s.Reset()
		return false
	}

	progressFingerprint = strings.TrimSpace(progressFingerprint)
	if progressFingerprint == "" {
		s.lastFailureFingerprint = failureFingerprint
		s.lastChangeFingerprint = ""
		s.repeatCount = 1
		return false
	}

	if failureFingerprint == s.lastFailureFingerprint && progressFingerprint == s.lastChangeFingerprint {
		s.repeatCount++
	} else {
		s.lastFailureFingerprint = failureFingerprint
		s.lastChangeFingerprint = progressFingerprint
		s.repeatCount = 1
	}

	return s.repeatCount >= noProgressThreshold
}
