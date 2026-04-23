package tools

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
	scanner := newMarkdownFenceScanner(text)

	for {
		fence, ok := scanner.Next()
		if !ok {
			break
		}
		if !fence.closed {
			if policy.unclosedFence == markdownUnclosedFencePolicyToEOF {
				// 閉じていない場合は残り全部をコードブロックとみなす
				ranges = append(ranges, [2]int{fence.start, fence.end})
			}
			break
		}
		ranges = append(ranges, [2]int{fence.start, fence.end})
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
