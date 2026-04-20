package ast

import (
	"regexp"
	"strings"

	"github.com/odvcencio/gotreesitter"
)

func inferTypeFromCompositeLiteralText(expr string) string {
	expr = strings.TrimSpace(strings.TrimPrefix(expr, "&"))
	if expr == "" {
		return ""
	}
	if idx := strings.Index(expr, "{"); idx > 0 {
		return strings.TrimSpace(expr[:idx])
	}
	return ""
}

func inferIdentifierType(node *gotreesitter.Node, src []byte, lang *gotreesitter.Language, matchByte uint32) string {
	if node == nil {
		return ""
	}

	name := strings.TrimSpace(node.Text(src))
	if name == "" {
		return ""
	}

	scope := findEnclosingCallable(node, lang)
	if scope == nil {
		return ""
	}

	signature := extractSignature(scope, src, lang)
	if typ := inferIdentifierTypeFromSignature(signature, name); typ != "" {
		return typ
	}

	if matchByte <= scope.StartByte() || matchByte > uint32(len(src)) {
		return ""
	}
	return inferIdentifierTypeFromPrefix(string(src[scope.StartByte():matchByte]), name)
}

func findEnclosingCallable(node *gotreesitter.Node, lang *gotreesitter.Language) *gotreesitter.Node {
	for current := node; current != nil; current = current.Parent() {
		switch current.Type(lang) {
		case "function_declaration", "method_declaration":
			return current
		}
	}
	return nil
}

func inferIdentifierTypeFromSignature(signature, name string) string {
	signature = strings.TrimSpace(signature)
	if !strings.HasPrefix(signature, "func") {
		return ""
	}

	rest := strings.TrimSpace(strings.TrimPrefix(signature, "func"))
	if strings.HasPrefix(rest, "(") {
		receiverSpec, after := splitLeadingGroup(rest)
		if typ := inferNamedTypeFromSection(receiverSpec, name); typ != "" {
			return typ
		}
		rest = strings.TrimSpace(after)
	}

	openIdx := strings.Index(rest, "(")
	if openIdx < 0 || openIdx >= len(rest) {
		return ""
	}
	paramsSpec, _ := splitLeadingGroup(rest[openIdx:])
	return inferNamedTypeFromSection(paramsSpec, name)
}

func splitLeadingGroup(s string) (string, string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "(") {
		return "", s
	}

	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[1:i]), strings.TrimSpace(s[i+1:])
			}
		}
	}
	return "", ""
}

func inferNamedTypeFromSection(section, name string) string {
	section = strings.TrimSpace(section)
	if section == "" || name == "" {
		return ""
	}

	// カンマで分割してパラメータグループを解析する。
	// Go の "a, b Config" のようなグループ化パラメータに対応する。
	entries := splitTopLevelCommas(section)
	var pendingNames []string
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		fields := strings.Fields(entry)
		if len(fields) == 1 {
			// 型なし: グループ内の名前
			pendingNames = append(pendingNames, fields[0])
		} else {
			// "paramName Type" 形式
			paramName := fields[0]
			typeName := strings.TrimSpace(entry[len(paramName):])
			if paramName == name {
				return typeName
			}
			for _, pn := range pendingNames {
				if pn == name {
					return typeName
				}
			}
			pendingNames = nil
		}
	}
	return ""
}

// splitTopLevelCommas は括弧のネストを考慮してトップレベルのカンマで分割する。
func splitTopLevelCommas(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func inferIdentifierTypeFromPrefix(prefix, name string) string {
	if prefix == "" || name == "" {
		return ""
	}

	quotedName := regexp.QuoteMeta(name)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)\b` + quotedName + `\s*:?=\s*&?\s*([A-Za-z_][A-Za-z0-9_]*(?:\[[^\]\n]+\])?)\s*\{`),
		regexp.MustCompile(`(?m)\bvar\s+` + quotedName + `\s*=\s*&?\s*([A-Za-z_][A-Za-z0-9_]*(?:\[[^\]\n]+\])?)\s*\{`),
		regexp.MustCompile(`(?m)\bvar\s+` + quotedName + `\s+([*A-Za-z_][A-Za-z0-9_\.\[\]\*]*)`),
	}

	latestPos := -1
	latestType := ""
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatchIndex(prefix, -1)
		for _, loc := range matches {
			if len(loc) < 4 {
				continue
			}
			if loc[0] >= latestPos {
				latestPos = loc[0]
				latestType = strings.TrimSpace(prefix[loc[2]:loc[3]])
			}
		}
	}

	// グループ化 var 宣言: "var a, name Type"
	if latestType == "" {
		latestType = inferTypeFromGroupedVarDecl(prefix, name)
	}

	return latestType
}

// inferTypeFromGroupedVarDecl はグループ化 var 宣言 ("var a, b Type") から型を推論する。
func inferTypeFromGroupedVarDecl(prefix, name string) string {
	groupedVarRe := regexp.MustCompile(`(?m)\bvar\s+(\w+(?:\s*,\s*\w+)+)\s+([*A-Za-z_][A-Za-z0-9_.\[\]*]*)`)
	for _, match := range groupedVarRe.FindAllStringSubmatch(prefix, -1) {
		if len(match) < 3 {
			continue
		}
		for _, n := range strings.Split(match[1], ",") {
			if strings.TrimSpace(n) == name {
				return strings.TrimSpace(match[2])
			}
		}
	}
	return ""
}
