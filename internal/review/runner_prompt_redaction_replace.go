package review

import "strings"

func replaceReviewRunnerPromptPath(text, path, display string) string {
	if text == "" || path == "" {
		return text
	}

	var b strings.Builder
	last := 0
	for {
		index := strings.Index(text[last:], path)
		if index < 0 {
			b.WriteString(text[last:])
			return b.String()
		}

		start := last + index
		end := start + len(path)
		if !isReviewRunnerPromptPathBoundary(text, start, end) {
			b.WriteString(text[last:end])
			last = end
			continue
		}

		b.WriteString(text[last:start])
		b.WriteString(display)
		last = end
	}
}

func isReviewRunnerPromptPathBoundary(text string, start, end int) bool {
	beforeOK := start == 0 || !isReviewRunnerPromptPathTokenByte(text[start-1])
	afterOK := end == len(text) || text[end] == '/' || text[end] == '\\' || !isReviewRunnerPromptPathTokenByte(text[end])
	return beforeOK && afterOK
}
