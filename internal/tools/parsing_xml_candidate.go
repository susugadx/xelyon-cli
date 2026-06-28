package tools

// XMLToolCallCandidateTagNames は XML 形式の tool call 候補タグ名を返す。
// 候補は Markdown code block 外にあり、対応する閉じタグを持つ開始タグだけに限定する。
func XMLToolCallCandidateTagNames(response string) []string {
	if response == "" {
		return nil
	}

	scanner := newXMLToolCallScanner(response)
	codeBlockRanges := findCodeBlockRanges(response)

	var names []string
	for {
		candidate, ok := scanner.Next()
		if !ok {
			break
		}
		if isInCodeBlock(candidate.start, codeBlockRanges) {
			continue
		}
		names = append(names, candidate.tagName)
	}
	return names
}
