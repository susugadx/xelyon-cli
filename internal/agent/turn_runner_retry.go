package agent

import "strings"

// retryState は retry ループの空回り検出を行う内部状態。
//
// stalled（空回り）は「停止の絶対条件」ではなく「方針変更ヒント」として扱う。
// errorFingerprint は雑な近似であり、false positive / false negative の両方があり得る。
// そのため stalled 検出時もまず retry 指示を強化し、即座に hard stop はしない。
type retryState struct {
	count       int    // 累積リトライ回数（表示用）
	lastErrorFP string // 直前のエラー fingerprint（空回り近似用）
	sameCount   int    // 同一 fingerprint の連続回数
	stalledRuns int    // stalled 検出後に続行した回数（soft→hard エスカレーション用）
}

// stalledRetryThreshold は同一 fingerprint が連続何回で stalled hint とみなすか。
const stalledRetryThreshold = 3

// stalledHardThreshold は stalled 検出後にさらに何回失敗したら hard escalation するか。
// hard escalation 時は caller が外部介入方針（AI への委譲など）を選ぶ。
const stalledHardThreshold = 2

// stalledLevel は空回り検出の深刻度。
type stalledLevel int

const (
	stalledNone stalledLevel = iota // 空回りなし → 通常リトライ
	stalledSoft                     // 空回りヒント → retry 指示を強化して続行
	stalledHard                     // 空回り確定 → 外部介入
)

// recordFailure はエラーを記録し、空回りの深刻度を返す。
//   - stalledNone: 新しいエラーまたは閾値未到達 → 通常リトライ
//   - stalledSoft: 同一 fingerprint が閾値に達した → retry 指示を強化して続行
//   - stalledHard: soft 後もさらに同一エラーが続いた → 外部介入
func (s *retryState) recordFailure(errorOutput string) stalledLevel {
	fp := errorFingerprint(errorOutput)
	if fp == s.lastErrorFP {
		s.sameCount++
	} else {
		s.lastErrorFP = fp
		s.sameCount = 1
		s.stalledRuns = 0
	}
	s.count++

	if s.sameCount < stalledRetryThreshold {
		return stalledNone
	}
	s.stalledRuns++
	if s.stalledRuns <= stalledHardThreshold {
		return stalledSoft
	}
	return stalledHard
}

// reset は成功時やユーザー手動リトライ時に状態をリセットする。
func (s *retryState) reset() {
	s.lastErrorFP = ""
	s.sameCount = 0
	s.stalledRuns = 0
	s.count = 0
}

// errorFingerprint はエラー出力から空回り検出用の雑な近似 fingerprint を返す。
//
// 厳密なエラー同一性判定ではなく、「同じ根本原因のエラーが繰り返されている可能性」の
// ヒントとして使う。軽い正規化（trim, ANSI 除去, 空白圧縮）後の先頭 200 文字で比較する。
// false positive（別エラーを同一視）/ false negative（同一エラーを別扱い）の両方があり得るため、
// この fingerprint だけで hard stop の判断はしない。
func errorFingerprint(s string) string {
	s = strings.TrimSpace(s)
	s = normalizeErrorText(s)
	return truncateRunes(s, 200)
}

// truncateRunes は s を最大 n ルーンで切り詰める（rune 境界で安全に切断）。
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

// normalizeErrorText は ANSI エスケープ除去 + 連続空白圧縮を行う。
func normalizeErrorText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for i := 0; i < len(s); i++ {
		// ANSI CSI シーケンス: \x1b[ ... 終端文字 まで読み飛ばす
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
