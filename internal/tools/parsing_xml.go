package tools

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
		openTag, ok := findNextXMLOpenTag(s.response, s.searchFrom)
		if !ok {
			return xmlToolCallCandidate{}, false
		}

		closeIdx := findXMLCloseTagIndex(s.response, openTag.contentStart, openTag.tagName)
		if closeIdx == -1 {
			s.searchFrom = openTag.contentStart
			continue
		}

		closeTag := xmlCloseTag(openTag.tagName)
		absCloseStart := openTag.contentStart + closeIdx
		fullEnd := absCloseStart + len(closeTag)
		innerContent := s.response[openTag.contentStart:absCloseStart]
		s.searchFrom = fullEnd

		return xmlToolCallCandidate{
			tagName:      openTag.tagName,
			innerContent: innerContent,
			start:        openTag.openStart,
		}, true
	}

	return xmlToolCallCandidate{}, false
}

func (f *xmlToolCallFilter) Accept(candidate xmlToolCallCandidate) bool {
	if isInCodeBlock(candidate.start, f.codeBlockRanges) {
		f.logger.LogEvent(newParseDebugXMLRescueSkipInCodeBlockEvent(candidate.tagName))
		return false
	}

	if !f.registry.HasTool(candidate.tagName) {
		f.logger.LogEvent(newParseDebugXMLRescueSkipUnknownToolEvent(candidate.tagName))
		return false
	}

	return true
}

func (d *xmlToolCallDecoder) Decode(candidate xmlToolCallCandidate) *ToolCall {
	args := parseXMLParams(candidate.innerContent)
	d.logger.LogEvent(newParseDebugXMLRescueToolCallEvent(candidate.tagName, args))
	return &ToolCall{
		Tool: candidate.tagName,
		Args: args,
	}
}
