package turnsupport

import "strings"

// RetryState は normal turn の retry ループで空回り検出を行う状態。
//
// stalled（空回り）は「停止の絶対条件」ではなく「方針変更ヒント」として扱う。
// ErrorFingerprint は雑な近似であり、false positive / false negative の両方があり得る。
// そのため stalled 検出時もまず retry 指示を強化し、即座に hard stop はしない。
type RetryState struct {
	count       int
	lastErrorFP string
	sameCount   int
	stalledRuns int
}

const stalledRetryThreshold = 3
const stalledHardThreshold = 2

// StalledLevel は retry の空回り検出の深刻度。
type StalledLevel int

const (
	// StalledNone は空回りなしで通常 retry を続けられる状態。
	StalledNone StalledLevel = iota
	// StalledSoft は同一失敗が続いており、retry 指示を強化すべき状態。
	StalledSoft
	// StalledHard は同一失敗がさらに続いており、caller が外部介入方針を選ぶ状態。
	StalledHard
)

// RecordFailure はエラーを記録し、空回りの深刻度を返す。
func (s *RetryState) RecordFailure(errorOutput string) StalledLevel {
	if s == nil {
		return StalledNone
	}
	fp := ErrorFingerprint(errorOutput)
	if fp == s.lastErrorFP {
		s.sameCount++
	} else {
		s.lastErrorFP = fp
		s.sameCount = 1
		s.stalledRuns = 0
	}
	s.count++

	if s.sameCount < stalledRetryThreshold {
		return StalledNone
	}
	s.stalledRuns++
	if s.stalledRuns <= stalledHardThreshold {
		return StalledSoft
	}
	return StalledHard
}

// Reset は成功時やユーザー手動 retry 時に状態をリセットする。
func (s *RetryState) Reset() {
	if s == nil {
		return
	}
	s.lastErrorFP = ""
	s.sameCount = 0
	s.stalledRuns = 0
	s.count = 0
}

// Count は累積 retry 回数を返す。
func (s *RetryState) Count() int {
	if s == nil {
		return 0
	}
	return s.count
}

// SameCount は同一 fingerprint の連続回数を返す。
func (s *RetryState) SameCount() int {
	if s == nil {
		return 0
	}
	return s.sameCount
}

// ErrorFingerprint はエラー出力から空回り検出用の近似 fingerprint を返す。
//
// 厳密なエラー同一性判定ではなく、「同じ根本原因のエラーが繰り返されている可能性」の
// ヒントとして使う。軽い正規化（trim, ANSI 除去, 空白圧縮）後の先頭 200 文字で比較する。
func ErrorFingerprint(s string) string {
	s = strings.TrimSpace(s)
	s = normalizeErrorText(s)
	return truncateRunes(s, 200)
}

func truncateRunes(s string, n int) string {
	count := 0
	for i := range s {
		if count >= n {
			return s[:i]
		}
		count++
	}
	return s
}

func normalizeErrorText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] >= 0x20 && s[j] <= 0x3F {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j - 1
			continue
		}
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		} else {
			b.WriteByte(c)
			prevSpace = false
		}
	}
	return b.String()
}
