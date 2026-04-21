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

type xmlToolCallCandidate struct {
	tagName      string
	innerContent string
	start        int
}

type xmlToolCallScanner struct {
	response   string
	searchFrom int
}

// parseXMLToolCalls はXML形式のツール呼び出しをパースする
// Kimi K2 等がFC失敗時に出力する XML 形式を rescue する
func parseXMLToolCalls(response string, codeBlockRanges [][2]int, debug bool, registry *Registry, debugOut io.Writer) []*ToolCall {
	scanner := newXMLToolCallScanner(response)

	var results []*ToolCall
	for {
		candidate, ok := scanner.Next()
		if !ok {
			break
		}
		if !shouldAcceptXMLToolCallCandidate(candidate, codeBlockRanges, debug, registry, debugOut) {
			continue
		}

		args := parseXMLParams(candidate.innerContent)
		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] XML rescue: tool=%s, args=%v\n", candidate.tagName, args)
		}

		results = append(results, &ToolCall{
			Tool: candidate.tagName,
			Args: args,
		})
	}

	return results
}

func newXMLToolCallScanner(response string) *xmlToolCallScanner {
	return &xmlToolCallScanner{response: response}
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

func shouldAcceptXMLToolCallCandidate(candidate xmlToolCallCandidate, codeBlockRanges [][2]int, debug bool, registry *Registry, debugOut io.Writer) bool {
	if isInCodeBlock(candidate.start, codeBlockRanges) {
		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] XML rescue: skipping %q in code block\n", candidate.tagName)
		}
		return false
	}

	if !registry.HasTool(candidate.tagName) {
		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] XML rescue: skipping unknown tool %q\n", candidate.tagName)
		}
		return false
	}

	return true
}

// parseXMLParams は XML 内部コンテンツからパラメータを抽出する
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
