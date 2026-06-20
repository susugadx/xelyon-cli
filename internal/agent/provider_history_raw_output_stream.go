package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	providerHistoryRawOutputContextMinBufferBytes = 32 * 1024
	providerHistoryRawOutputContextMaxBufferBytes = 256 * 1024
	providerHistoryRawOutputContextMaxLineBytes   = 32 * 1024
	providerHistoryRawOutputContextLineRadius     = 4
)

type providerHistoryRawOutputContextScanner struct {
	hints      []string
	bodyBudget int
	bufferCap  int
	buffer     strings.Builder
	overflow   bool
	remainder  string
	openMatch  bool
	prevLines  []string
	matched    bool
	matchTerm  string
	matchLine  int
	matchStart int
	matchLines []string
	afterLines int
	totalLines int
	finalized  bool
}

func newProviderHistoryRawOutputContextScanner(hints []string, bodyBudgetTokens int) *providerHistoryRawOutputContextScanner {
	bufferCap := bodyBudgetTokens * 8
	if bufferCap < providerHistoryRawOutputContextMinBufferBytes {
		bufferCap = providerHistoryRawOutputContextMinBufferBytes
	}
	if bufferCap > providerHistoryRawOutputContextMaxBufferBytes {
		bufferCap = providerHistoryRawOutputContextMaxBufferBytes
	}
	return &providerHistoryRawOutputContextScanner{
		hints:      append([]string(nil), hints...),
		bodyBudget: bodyBudgetTokens,
		bufferCap:  bufferCap,
	}
}

func (s *providerHistoryRawOutputContextScanner) Scan(chunk []byte) error {
	if len(chunk) == 0 {
		return nil
	}
	if !s.overflow {
		if s.buffer.Len()+len(chunk) <= s.bufferCap {
			s.buffer.Write(chunk)
		} else {
			s.overflow = true
			s.buffer.Reset()
		}
	}
	s.scanLines(string(chunk))
	return nil
}

func (s *providerHistoryRawOutputContextScanner) Body() (string, string) {
	s.finalize()
	if !s.overflow {
		body := strings.TrimSpace(s.buffer.String())
		if body == "" {
			return "", providerHistoryRawOutputRequiredRefsMissingReason
		}
		return body, ""
	}
	if !s.matched {
		return "", providerHistoryRawOutputActiveContextCoverageInsufficientReason
	}
	body := s.renderMatchedExcerpt()
	if strings.TrimSpace(body) == "" {
		return "", providerHistoryRawOutputActiveContextCoverageInsufficientReason
	}
	return body, ""
}

func (s *providerHistoryRawOutputContextScanner) renderMatchedExcerpt() string {
	if len(s.matchLines) == 0 {
		return ""
	}
	budget := s.bodyBudget
	if budget <= 0 {
		budget = 128
	}
	matchOffset := s.matchLine - s.matchStart
	if matchOffset < 0 || matchOffset >= len(s.matchLines) {
		return ""
	}
	maxRadius := providerHistoryRawOutputMinInt(
		providerHistoryRawOutputContextLineRadius,
		providerHistoryRawOutputMaxInt(matchOffset, len(s.matchLines)-matchOffset-1),
	)
	for radius := maxRadius; radius >= 0; radius-- {
		startOffset := providerHistoryRawOutputMaxInt(0, matchOffset-radius)
		endOffset := providerHistoryRawOutputMinInt(len(s.matchLines), matchOffset+radius+1)
		selected := append([]string(nil), s.matchLines[startOffset:endOffset]...)
		if radius == 0 {
			selected[0] = providerHistoryRawOutputTrimMatchedLine(selected[0], s.matchTerm, budget)
		}
		start := s.matchStart + startOffset
		end := s.matchStart + endOffset
		excerpt := providerHistoryRawOutputRenderMatchedExcerpt(selected, s.matchTerm, s.matchLine, s.totalLines, start, end)
		if token.EstimateTokenCount(excerpt) <= budget {
			return excerpt
		}
	}
	line := providerHistoryRawOutputTrimMatchedLine(s.matchLines[matchOffset], s.matchTerm, budget/2)
	return providerHistoryRawOutputRenderMatchedExcerpt([]string{line}, s.matchTerm, s.matchLine, s.totalLines, s.matchLine, s.matchLine+1)
}

func (s *providerHistoryRawOutputContextScanner) scanLines(chunk string) {
	if chunk == "" {
		return
	}
	combined := s.remainder + chunk
	for {
		idx := strings.IndexByte(combined, '\n')
		if idx < 0 {
			s.storeLineRemainder(combined)
			return
		}
		s.scanCompleteLine(combined[:idx])
		combined = combined[idx+1:]
	}
}

func (s *providerHistoryRawOutputContextScanner) finalize() {
	if s.finalized {
		return
	}
	s.finalized = true
	if s.remainder != "" {
		s.scanCompleteLine(s.remainder)
		s.remainder = ""
	}
}

func (s *providerHistoryRawOutputContextScanner) storeLineRemainder(line string) {
	if len(line) <= providerHistoryRawOutputContextMaxLineBytes {
		s.remainder = line
		return
	}
	s.scanOpenLineFragment(line)
	s.remainder = providerHistoryRawOutputTail(line, providerHistoryRawOutputContextMaxLineBytes)
}

func (s *providerHistoryRawOutputContextScanner) scanOpenLineFragment(line string) {
	if s.matched {
		return
	}
	if term := providerHistoryRawOutputLineMatchedTerm(line, s.hints); term != "" {
		fragment := providerHistoryRawOutputLineAroundTerm(line, term, providerHistoryRawOutputContextMaxLineBytes)
		s.recordMatchedLine(s.totalLines, fragment, term)
		s.openMatch = true
	}
}

func (s *providerHistoryRawOutputContextScanner) scanCompleteLine(line string) {
	if s.openMatch {
		s.totalLines++
		s.openMatch = false
		return
	}
	if len(line) > providerHistoryRawOutputContextMaxLineBytes {
		s.scanOpenLineFragment(line)
		line = providerHistoryRawOutputTail(line, providerHistoryRawOutputContextMaxLineBytes)
		if s.openMatch {
			s.totalLines++
			s.openMatch = false
			return
		}
	}
	s.scanLine(line)
}

func (s *providerHistoryRawOutputContextScanner) scanLine(line string) {
	lineIndex := s.totalLines
	s.totalLines++
	if s.matched {
		if s.afterLines > 0 {
			s.matchLines = append(s.matchLines, line)
			s.afterLines--
		}
		return
	}
	if term := providerHistoryRawOutputLineMatchedTerm(line, s.hints); term != "" {
		s.recordMatchedLine(lineIndex, line, term)
		return
	}
	s.prevLines = append(s.prevLines, line)
	if len(s.prevLines) > providerHistoryRawOutputContextLineRadius {
		s.prevLines = s.prevLines[len(s.prevLines)-providerHistoryRawOutputContextLineRadius:]
	}
}

func (s *providerHistoryRawOutputContextScanner) recordMatchedLine(lineIndex int, line, term string) {
	s.matched = true
	s.matchTerm = term
	s.matchLine = lineIndex
	s.matchStart = lineIndex - len(s.prevLines)
	s.matchLines = append(append([]string(nil), s.prevLines...), line)
	s.afterLines = providerHistoryRawOutputContextLineRadius
}

func providerHistoryRawOutputLineMatchedTerm(line string, hints []string) string {
	if len(hints) == 0 || strings.TrimSpace(line) == "" {
		return ""
	}
	lowerLine := strings.ToLower(line)
	for _, hint := range hints {
		if hint != "" && strings.Contains(lowerLine, hint) {
			return hint
		}
	}
	return ""
}

func providerHistoryRawOutputTail(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	return value[len(value)-maxBytes:]
}

func providerHistoryRawOutputLineAroundTerm(line, term string, maxBytes int) string {
	if maxBytes <= 0 || len(line) <= maxBytes {
		return line
	}
	index := strings.Index(strings.ToLower(line), strings.ToLower(term))
	if index < 0 {
		return providerHistoryRawOutputTail(line, maxBytes)
	}
	start := index - maxBytes/2
	if start < 0 {
		start = 0
	}
	end := start + maxBytes
	if end > len(line) {
		end = len(line)
		start = providerHistoryRawOutputMaxInt(0, end-maxBytes)
	}
	return line[start:end]
}
