package commandruntime

import "unicode"

// shouldStartQuoteSegment は語中 quote を quote セグメントとして開始するかを判定する。
// token 先頭は常に quote セグメントとして開始し、語中は対応する閉じ quote がある場合のみ開始する。
func shouldStartQuoteSegment(currentTokenLen int, runes []rune, openIndex int, quote rune) bool {
	if currentTokenLen == 0 {
		return true
	}
	if quote == '\'' && shouldTreatApostropheAsLiteralInCurrentToken(runes, openIndex) {
		return false
	}
	closeIndex := findClosingQuoteIndex(runes, openIndex+1, quote)
	if closeIndex < 0 {
		return false
	}
	return true
}

func findClosingQuoteIndex(runes []rune, start int, quote rune) int {
	for i := start; i < len(runes); i++ {
		if runes[i] != quote {
			continue
		}
		if i > start && runes[i-1] == '\\' {
			continue
		}
		return i
	}
	return -1
}

// shouldTreatApostropheAsLiteralInCurrentToken は現在 token 範囲だけで apostrophe literal を判定する。
// don't / can't / it's など短い英語短縮形を quote 開始ではなく文字として扱う。
func shouldTreatApostropheAsLiteralInCurrentToken(runes []rune, openIndex int) bool {
	if !hasLetterNeighborsAroundIndex(runes, openIndex) {
		return false
	}
	suffix := tokenFragmentAfterIndex(runes, openIndex)
	return isShortAlphabeticFragment(suffix, 2)
}

func nextWhitespaceIndex(runes []rune, start int) int {
	for i := start; i < len(runes); i++ {
		if unicode.IsSpace(runes[i]) {
			return i
		}
	}
	return len(runes)
}

func lettersOnly(runes []rune) bool {
	if len(runes) == 0 {
		return false
	}
	for _, r := range runes {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func hasLetterNeighborsAroundIndex(runes []rune, index int) bool {
	if index <= 0 || index+1 >= len(runes) {
		return false
	}
	return unicode.IsLetter(runes[index-1]) && unicode.IsLetter(runes[index+1])
}

func tokenFragmentAfterIndex(runes []rune, index int) []rune {
	start := index + 1
	end := nextWhitespaceIndex(runes, start)
	return runes[start:end]
}

func isShortAlphabeticFragment(runes []rune, maxLen int) bool {
	if len(runes) == 0 || len(runes) > maxLen {
		return false
	}
	return lettersOnly(runes)
}
