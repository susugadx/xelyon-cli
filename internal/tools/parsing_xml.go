package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// xmlOpenTagPattern は <tag_name> 形式の開始タグを検出する正規表現
var xmlOpenTagPattern = regexp.MustCompile(`<([a-zA-Z_][\w-]*)>`)

// parseXMLToolCalls はXML形式のツール呼び出しをパースする
// Kimi K2 等がFC失敗時に出力する XML 形式を rescue する
func parseXMLToolCalls(response string, codeBlockRanges [][2]int, debug bool, registry *Registry, debugOut io.Writer) []*ToolCall {
	var results []*ToolCall
	searchFrom := 0

	for searchFrom < len(response) {
		// 開始タグを探す
		loc := xmlOpenTagPattern.FindStringIndex(response[searchFrom:])
		if loc == nil {
			break
		}
		absStart := searchFrom + loc[0]
		tagEnd := searchFrom + loc[1]

		// タグ名を抽出
		tagName := xmlOpenTagPattern.FindStringSubmatch(response[searchFrom:])[1]

		// 対応する閉じタグを探す
		closeTag := "</" + tagName + ">"
		closeIdx := strings.Index(response[tagEnd:], closeTag)
		if closeIdx == -1 {
			searchFrom = tagEnd
			continue
		}
		absCloseStart := tagEnd + closeIdx
		fullEnd := absCloseStart + len(closeTag)

		innerContent := response[tagEnd:absCloseStart]

		// 次の検索位置を更新
		searchFrom = fullEnd

		// コードブロック内はスキップ
		if isInCodeBlock(absStart, codeBlockRanges) {
			if debug {
				fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] XML rescue: skipping %q in code block\n", tagName)
			}
			continue
		}

		// 指定 Registry に登録されているツール名のみ許可
		if !registry.HasTool(tagName) {
			if debug {
				fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] XML rescue: skipping unknown tool %q\n", tagName)
			}
			continue
		}

		// 内部コンテンツからパラメータを抽出
		args := parseXMLParams(innerContent)

		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] XML rescue: tool=%s, args=%v\n", tagName, args)
		}

		tc := &ToolCall{
			Tool: tagName,
			Args: args,
		}
		results = append(results, tc)
	}

	return results
}

// parseXMLParams は XML 内部コンテンツからパラメータを抽出する
// パターン1: <args><param>value</param>...</args> （args ラッパーあり）
// パターン2: <param>value</param>... （args ラッパーなし）
// パターン3: {"key": "value"} （JSON 形式）
func parseXMLParams(content string) map[string]string {
	args := make(map[string]string)

	// <args>...</args> ラッパーがある場合は中身を取り出す
	if argsStart := strings.Index(content, "<args>"); argsStart != -1 {
		if argsEnd := strings.Index(content, "</args>"); argsEnd != -1 && argsEnd > argsStart {
			content = content[argsStart+len("<args>") : argsEnd]
		}
	}

	// <param>value</param> を手動パースで抽出（バックリファレンス不要）
	searchFrom := 0
	for searchFrom < len(content) {
		// 開始タグを探す
		loc := xmlOpenTagPattern.FindStringIndex(content[searchFrom:])
		if loc == nil {
			break
		}
		tagStart := searchFrom + loc[1]
		tagName := xmlOpenTagPattern.FindStringSubmatch(content[searchFrom:])[1]

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

	// XML パラメータが見つからなかった場合、内容を JSON としてパース
	if len(args) == 0 {
		trimmed := strings.TrimSpace(content)
		if strings.HasPrefix(trimmed, "{") {
			var jsonArgs map[string]interface{}
			if err := json.Unmarshal([]byte(trimmed), &jsonArgs); err == nil {
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
			}
		}
	}

	return args
}
