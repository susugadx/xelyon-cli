package tui

import "strings"

func appendContinuationPrefix(currentContent *strings.Builder, currentWidth *int, subRowIdx int, prefix string, prefixWidth int) {
	if subRowIdx <= 0 || prefix == "" {
		return
	}
	currentContent.WriteString(prefix)
	*currentWidth += prefixWidth
}

func continuationPrefixWidth(subRowIdx int, prefixWidth int) int {
	if subRowIdx > 0 {
		return prefixWidth
	}
	return 0
}

func getContinuationPrefix(line string) string {
	var prefix strings.Builder
	for _, r := range line {
		if r == ' ' {
			prefix.WriteRune(r)
			continue
		}
		break
	}
	return prefix.String()
}
