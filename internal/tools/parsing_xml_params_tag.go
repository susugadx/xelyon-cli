package tools

import "strings"

func unwrapXMLArgsContent(content string) string {
	// <args>...</args> ラッパーがある場合は中身を取り出す
	if argsStart := strings.Index(content, "<args>"); argsStart != -1 {
		if argsEnd := strings.Index(content, "</args>"); argsEnd != -1 && argsEnd > argsStart {
			return content[argsStart+len("<args>") : argsEnd]
		}
	}
	return content
}

func parseXMLTagParams(content string) map[string]string {
	args := make(map[string]string)

	// <param>value</param> を手動パースで抽出（バックリファレンス不要）
	searchFrom := 0
	for searchFrom < len(content) {
		openTag, ok := findNextXMLOpenTag(content, searchFrom)
		if !ok {
			break
		}

		// "args" タグ自体はスキップ
		if openTag.tagName == "args" {
			searchFrom = openTag.contentStart
			continue
		}

		// 対応する閉じタグを探す
		closeIdx := findXMLCloseTagIndex(content, openTag.contentStart, openTag.tagName)
		if closeIdx == -1 {
			searchFrom = openTag.contentStart
			continue
		}

		closeTag := xmlCloseTag(openTag.tagName)
		value := content[openTag.contentStart : openTag.contentStart+closeIdx]
		args[openTag.tagName] = value
		searchFrom = openTag.contentStart + closeIdx + len(closeTag)
	}

	return args
}

func (xmlTagParamsParseStrategy) Parse(content string) xmlParamsParseOutcome {
	args := parseXMLTagParams(content)
	if len(args) == 0 {
		return unhandledXMLParamsOutcome()
	}
	return handledXMLParamsOutcome(args)
}
