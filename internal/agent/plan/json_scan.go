package plan

import (
	"encoding/json"
	"strconv"
)

type jsonValueKind int

const (
	jsonValueInvalid jsonValueKind = iota
	jsonValueObject
	jsonValueArray
	jsonValueString
	jsonValueScalar
)

func topLevelRawValue(jsonStr string, key string) (json.RawMessage, bool, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		_, ok := topLevelValueStart(jsonStr, key)
		return nil, ok, false
	}
	value, ok := obj[key]
	return value, ok, true
}

func jsonValueKindAt(s string, start int) jsonValueKind {
	start = skipJSONWhitespace(s, start)
	if start >= len(s) {
		return jsonValueInvalid
	}

	switch ch := s[start]; {
	case ch == '{':
		return jsonValueObject
	case ch == '[':
		return jsonValueArray
	case ch == '"':
		return jsonValueString
	case ch == '-' || ch >= '0' && ch <= '9' || ch == 't' || ch == 'f' || ch == 'n':
		return jsonValueScalar
	default:
		return jsonValueInvalid
	}
}

func topLevelValueStart(jsonStr string, key string) (int, bool) {
	i := skipJSONWhitespace(jsonStr, 0)
	if i >= len(jsonStr) || jsonStr[i] != '{' {
		return 0, false
	}

	depth := 0
	expectKey := false
	for i < len(jsonStr) {
		switch jsonStr[i] {
		case '"':
			value, next, ok := readJSONStringLiteral(jsonStr, i)
			if !ok {
				return 0, false
			}
			if depth == 1 && expectKey {
				colon := skipJSONWhitespace(jsonStr, next)
				if colon < len(jsonStr) && jsonStr[colon] == ':' {
					if value == key {
						return skipJSONWhitespace(jsonStr, colon+1), true
					}
					expectKey = false
					i = colon + 1
					continue
				}
			}
			i = next
			continue
		case '{', '[':
			depth++
			if depth == 1 {
				expectKey = true
			}
		case '}':
			if depth == 1 {
				return 0, false
			}
			depth--
		case ']':
			if depth == 0 {
				return 0, false
			}
			depth--
		case ',':
			if depth == 1 {
				expectKey = true
			}
		}
		i++
	}
	return 0, false
}

func readJSONStringLiteral(s string, start int) (string, int, bool) {
	escaped := false
	for i := start + 1; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		switch s[i] {
		case '\\':
			escaped = true
		case '"':
			value, err := strconv.Unquote(s[start : i+1])
			if err != nil {
				return "", i + 1, false
			}
			return value, i + 1, true
		}
	}
	return "", len(s), false
}

func skipJSONWhitespace(s string, start int) int {
	for start < len(s) {
		switch s[start] {
		case ' ', '\n', '\r', '\t':
			start++
		default:
			return start
		}
	}
	return start
}

// findClosingBrace は対応する閉じ括弧の位置を探す
func findClosingBrace(response string, start int) int {
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(response); i++ {
		ch := response[i]

		// エスケープ処理
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}

		// 文字列内チェック
		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		// 括弧の深さチェック
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}

	return -1
}
