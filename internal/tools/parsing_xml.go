package tools

import (
	"regexp"
	"strings"
)

// xmlOpenTagPattern は <tag_name> 形式の開始タグを検出する正規表現
var xmlOpenTagPattern = regexp.MustCompile(`<([a-zA-Z_][\w-]*)>`)

type xmlToolCallCandidate struct {
	tagName      string
	innerContent string
	start        int
}

type xmlToolCallScanner struct {
	response   string
	searchFrom int
}

type xmlToolCallFilter struct {
	codeBlockRanges [][2]int
	registry        *Registry
	logger          *parseDebugLogger
}

type xmlToolCallDecoder struct {
	logger *parseDebugLogger
}

// parseXMLToolCalls はXML形式のツール呼び出しをパースする
// Kimi K2 等がFC失敗時に出力する XML 形式を rescue する
func parseXMLToolCalls(response string, codeBlockRanges [][2]int, registry *Registry, logger *parseDebugLogger) []*ToolCall {
	scanner := newXMLToolCallScanner(response)
	filter := newXMLToolCallFilter(codeBlockRanges, registry, logger)
	decoder := newXMLToolCallDecoder(logger)

	var results []*ToolCall
	for {
		candidate, ok := scanner.Next()
		if !ok {
			break
		}
		if !filter.Accept(candidate) {
			continue
		}
		results = append(results, decoder.Decode(candidate))
	}

	return results
}

func newXMLToolCallScanner(response string) *xmlToolCallScanner {
	return &xmlToolCallScanner{response: response}
}

func newXMLToolCallFilter(codeBlockRanges [][2]int, registry *Registry, logger *parseDebugLogger) *xmlToolCallFilter {
	return &xmlToolCallFilter{
		codeBlockRanges: codeBlockRanges,
		registry:        registry,
		logger:          logger,
	}
}

func newXMLToolCallDecoder(logger *parseDebugLogger) *xmlToolCallDecoder {
	return &xmlToolCallDecoder{logger: logger}
}

func (s *xmlToolCallScanner) Next() (xmlToolCallCandidate, bool) {
	for s.searchFrom < len(s.response) {
		loc := xmlOpenTagPattern.FindStringSubmatchIndex(s.response[s.searchFrom:])
		if loc == nil {
			return xmlToolCallCandidate{}, false
		}

		absStart := s.searchFrom + loc[0]
		tagEnd := s.searchFrom + loc[1]
		tagName := s.response[s.searchFrom+loc[2] : s.searchFrom+loc[3]]

		closeTag := "</" + tagName + ">"
		closeIdx := strings.Index(s.response[tagEnd:], closeTag)
		if closeIdx == -1 {
			s.searchFrom = tagEnd
			continue
		}

		absCloseStart := tagEnd + closeIdx
		fullEnd := absCloseStart + len(closeTag)
		innerContent := s.response[tagEnd:absCloseStart]
		s.searchFrom = fullEnd

		return xmlToolCallCandidate{
			tagName:      tagName,
			innerContent: innerContent,
			start:        absStart,
		}, true
	}

	return xmlToolCallCandidate{}, false
}

func (f *xmlToolCallFilter) Accept(candidate xmlToolCallCandidate) bool {
	if isInCodeBlock(candidate.start, f.codeBlockRanges) {
		f.logger.Logf("[DEBUG ParseToolCalls] XML rescue: skipping %q in code block\n", candidate.tagName)
		return false
	}

	if !f.registry.HasTool(candidate.tagName) {
		f.logger.Logf("[DEBUG ParseToolCalls] XML rescue: skipping unknown tool %q\n", candidate.tagName)
		return false
	}

	return true
}

func (d *xmlToolCallDecoder) Decode(candidate xmlToolCallCandidate) *ToolCall {
	args := parseXMLParams(candidate.innerContent)
	d.logger.Logf("[DEBUG ParseToolCalls] XML rescue: tool=%s, args=%v\n", candidate.tagName, args)
	return &ToolCall{
		Tool: candidate.tagName,
		Args: args,
	}
}
