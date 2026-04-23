package tools

import "strings"

type markdownFenceCandidate struct {
	start  int
	end    int
	closed bool
}

type markdownFenceScanner struct {
	text       string
	searchFrom int
}

func newMarkdownFenceScanner(text string) *markdownFenceScanner {
	return &markdownFenceScanner{text: text}
}

func (s *markdownFenceScanner) Next() (markdownFenceCandidate, bool) {
	if s.searchFrom >= len(s.text) {
		return markdownFenceCandidate{}, false
	}

	start := strings.Index(s.text[s.searchFrom:], "```")
	if start == -1 {
		return markdownFenceCandidate{}, false
	}
	start += s.searchFrom

	endSearch := start + 3
	newline := strings.Index(s.text[endSearch:], "\n")
	if newline != -1 {
		endSearch += newline + 1
	}

	end := strings.Index(s.text[endSearch:], "```")
	if end == -1 {
		s.searchFrom = len(s.text)
		return markdownFenceCandidate{
			start:  start,
			end:    len(s.text),
			closed: false,
		}, true
	}

	end += endSearch + 3
	s.searchFrom = end
	return markdownFenceCandidate{
		start:  start,
		end:    end,
		closed: true,
	}, true
}
