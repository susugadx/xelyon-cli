package jsast

import "strings"

func testLSPRangeForToken(line, token string) (int, int) {
	idx := strings.Index(line, token)
	if idx < 0 {
		panic("token not found: " + token)
	}
	start := testLSPCharacterWidth(line[:idx]) + 1
	return start, start + testLSPCharacterWidth(token)
}

func testLSPCharacterWidth(text string) int {
	width := 0
	for _, r := range text {
		if r > 0xffff {
			width += 2
		} else {
			width++
		}
	}
	return width
}
