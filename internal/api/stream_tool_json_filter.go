package api

import "strings"

// toolJSONPatterns はツールJSON開始パターン。
var toolJSONPatterns = []string{
	`{"tool"`,
	`{ "tool"`,
	`{"id"`,
	`{ "id"`,
	`{"name"`,  // DeepSeek が OpenAI 互換形式で出力するパターン
	`{ "name"`, // 同上（スペース付き）
}

// matchesPatternPrefix はチャンク末尾がツールJSONパターンのプレフィックスに
// 一致する長さを返す。一致しない場合は 0。
// チャンク分割対応: 次チャンクと結合してからパターン判定するため。
func matchesPatternPrefix(content string) int {
	for _, pattern := range toolJSONPatterns {
		for prefixLen := len(pattern) - 1; prefixLen >= 1; prefixLen-- {
			prefix := pattern[:prefixLen]
			if strings.HasSuffix(content, prefix) {
				return prefixLen
			}
		}
	}
	return 0
}

// filterToolJSON はストリーミング中のツールJSONを検知して非表示にする。
//
// 設計（簡略化版）:
// - inToolJSON が true の間は全て非表示
// - strings.Index でパターンを検出（チャンク単位でシンプルに判定）
// - inString: JSON文字列リテラル内かどうかを追跡（文字列内の{}を無視するため）
// - escaped: JSON文字列内のエスケープシーケンス追跡用（\\ や \" を正しく処理する）
func filterToolJSON(content string, inToolJSON *bool, jsonDepth *int, inString *bool, escaped *bool) string {
	var result strings.Builder
	remaining := content

	for len(remaining) > 0 {
		if *inToolJSON {
			// ツールJSON内: 終了位置を探しながら非表示
			endIdx := -1
			for i, ch := range remaining {
				if *inString {
					if *escaped {
						// escape sequence consumed
						*escaped = false
					} else if ch == '\\' {
						*escaped = true
					} else if ch == '"' {
						*inString = false
					}
				} else {
					if ch == '"' {
						*inString = true
						*escaped = false
					} else if ch == '{' {
						*jsonDepth++
					} else if ch == '}' {
						*jsonDepth--
						if *jsonDepth == 0 {
							*inToolJSON = false
							*inString = false
							*escaped = false
							endIdx = i + 1 // '}' の次の位置
							break
						}
					}
				}
			}

			if endIdx == -1 {
				// JSON終了なし → 残り全て非表示
				return result.String()
			}
			// JSON終了 → 残りを継続処理
			remaining = remaining[endIdx:]
			continue
		}

		// パターン検出
		foundIdx := -1
		patternLen := 0
		for _, pattern := range toolJSONPatterns {
			if idx := strings.Index(remaining, pattern); idx != -1 {
				if foundIdx == -1 || idx < foundIdx {
					foundIdx = idx
					patternLen = len(pattern)
				}
			}
		}

		if foundIdx == -1 {
			// パターンなし → 残り全て表示
			result.WriteString(remaining)
			return result.String()
		}

		// パターン前の部分を出力
		result.WriteString(remaining[:foundIdx])
		// パターン自体をスキップし、inToolJSON状態に移行する
		remaining = remaining[foundIdx+patternLen:]

		// パターン以降を処理開始
		*inToolJSON = true
		*jsonDepth = 1 // パターンに含まれる最初の '{' をカウントした状態から開始
		*inString = false
		*escaped = false
	}

	return result.String()
}
