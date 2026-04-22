package tools

import "strings"

type markdownUnclosedFencePolicy int

const (
	markdownUnclosedFencePolicyToEOF markdownUnclosedFencePolicy = iota + 1
	markdownUnclosedFencePolicyIgnore
)

type markdownCodeBlockPolicy struct {
	unclosedFence markdownUnclosedFencePolicy
}

func defaultMarkdownCodeBlockPolicy() markdownCodeBlockPolicy {
	// 既存契約: 未クローズ fence は末尾まで code block として扱う。
	return markdownCodeBlockPolicy{
		unclosedFence: markdownUnclosedFencePolicyToEOF,
	}
}

// findCodeBlockRanges はMarkdownコードブロックの範囲を返す。
func findCodeBlockRanges(text string) [][2]int {
	return findCodeBlockRangesWithPolicy(text, defaultMarkdownCodeBlockPolicy())
}

func findCodeBlockRangesWithPolicy(text string, policy markdownCodeBlockPolicy) [][2]int {
	var ranges [][2]int
	idx := 0

	for idx < len(text) {
		// ``` の開始を探す
		start := strings.Index(text[idx:], "```")
		if start == -1 {
			break
		}
		start += idx

		// 対応する ``` の終了を探す（開始の次の行から）
		endSearch := start + 3
		// 言語指定がある場合は改行まで読み飛ばす
		newline := strings.Index(text[endSearch:], "\n")
		if newline != -1 {
			endSearch += newline + 1
		}

		end := strings.Index(text[endSearch:], "```")
		if end == -1 {
			if policy.unclosedFence == markdownUnclosedFencePolicyToEOF {
				// 閉じていない場合は残り全部をコードブロックとみなす
				ranges = append(ranges, [2]int{start, len(text)})
			}
			break
		}
		end += endSearch + 3

		ranges = append(ranges, [2]int{start, end})
		idx = end
	}

	return ranges
}

// isInCodeBlock は指定位置がコードブロック内かどうかを返す。
func isInCodeBlock(pos int, ranges [][2]int) bool {
	for _, r := range ranges {
		if pos >= r[0] && pos < r[1] {
			return true
		}
	}
	return false
}
