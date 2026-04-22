package tools

import (
	"encoding/json"
	"strings"
)

// parseXMLParams は XML 内部コンテンツからパラメータを抽出する。
// パターン1: <args><param>value</param>...</args> （args ラッパーあり）
// パターン2: <param>value</param>... （args ラッパーなし）
// パターン3: {"key": "value"} （JSON 形式）
func parseXMLParams(content string) map[string]string {
	content = unwrapXMLArgsContent(content)
	args := parseXMLTagParams(content)
	if len(args) > 0 {
		return args
	}
	return parseXMLJSONParams(content)
}

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
		// 開始タグを探す
		loc := xmlOpenTagPattern.FindStringSubmatchIndex(content[searchFrom:])
		if loc == nil {
			break
		}
		tagStart := searchFrom + loc[1]
		tagName := content[searchFrom+loc[2] : searchFrom+loc[3]]

		// "args" タグ自体はスキップ
		if tagName == "args" {
			searchFrom = tagStart
			continue
		}

		// 対応する閉じタグを探す
		closeTag := "</" + tagName + ">"
		closeIdx := strings.Index(content[tagStart:], closeTag)
		if closeIdx == -1 {
			searchFrom = tagStart
			continue
		}

		value := content[tagStart : tagStart+closeIdx]
		args[tagName] = value
		searchFrom = tagStart + closeIdx + len(closeTag)
	}

	return args
}

func parseXMLJSONParams(content string) map[string]string {
	args := make(map[string]string)
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") {
		return args
	}

	var jsonArgs map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &jsonArgs); err != nil {
		return args
	}

	for k, v := range jsonArgs {
		switch val := v.(type) {
		case string:
			args[k] = val
		default:
			if b, err := json.Marshal(v); err == nil {
				args[k] = string(b)
			}
		}
	}
	return args
}
