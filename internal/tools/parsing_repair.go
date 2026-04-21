package tools

import (
	"fmt"
	"strings"
)

// repairJSONStringValues は JSON 文字列値内の生制御文字をエスケープする。
// LLM が FC rescue テキストで出力する malformed JSON を修復する。
// 正常な JSON はそのまま返す。既にエスケープ済みの \\n, \\t, \\" 等は二重エスケープしない。
//
// 修復対象:
//   - 生改行 (0x0A) → \n
//   - 生キャリッジリターン (0x0D) → \r
//   - 生タブ (0x09) → \t
//   - その他の制御文字 (0x00-0x1F) → \uXXXX
func repairJSONStringValues(jsonStr string) string {
	var buf strings.Builder
	buf.Grow(len(jsonStr) + 64)

	inString := false
	escaped := false

	for i := 0; i < len(jsonStr); i++ {
		ch := jsonStr[i]

		if escaped {
			buf.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			buf.WriteByte(ch)
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			buf.WriteByte(ch)
			continue
		}

		if inString {
			switch ch {
			case '\n':
				buf.WriteString(`\n`)
			case '\r':
				buf.WriteString(`\r`)
			case '\t':
				buf.WriteString(`\t`)
			default:
				if ch < 0x20 {
					fmt.Fprintf(&buf, `\u%04x`, ch)
				} else {
					buf.WriteByte(ch)
				}
			}
		} else {
			buf.WriteByte(ch)
		}
	}

	return buf.String()
}
