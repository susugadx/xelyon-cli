package ast

import (
	"fmt"
	"strings"

	"github.com/odvcencio/gotreesitter"
)

// ClassifyLine は指定行の targetName マッチを AST ベースで分類する。
func ClassifyLine(path string, src []byte, line int, targetName string) (*MatchInfo, error) {
	tree, src, err := ParseBytes(path, src)
	if err != nil {
		return nil, err
	}
	return classifyLineInTree(tree, src, line, targetName)
}

// ClassifyLineWithParsed は事前パース済みのファイルを使って行を分類する。
func ClassifyLineWithParsed(pf *ParsedFile, line int, targetName string) (*MatchInfo, error) {
	if pf == nil {
		return nil, fmt.Errorf("ParsedFile is nil")
	}
	return classifyLineInTree(pf.tree, pf.src, line, targetName)
}

func classifyLineInTree(tree *gotreesitter.Tree, src []byte, line int, targetName string) (*MatchInfo, error) {
	if line <= 0 {
		return nil, fmt.Errorf("line must be >= 1: %d", line)
	}
	if targetName == "" {
		return nil, fmt.Errorf("targetName is required")
	}

	startByte, endByte, ok := lineByteRange(src, line)
	if !ok {
		return &MatchInfo{Class: ClassUnknown, Scope: "package-level"}, nil
	}

	lang := tree.Language()
	root := tree.RootNode()
	occurrences := findIdentifierOccurrences(src[startByte:endByte], targetName)
	for _, offset := range occurrences {
		absStart := startByte + offset
		absEnd := absStart + uint32(len(targetName))
		node := root.DescendantForByteRange(absStart, absEnd)
		if node == nil {
			continue
		}
		class := classifyNode(node, absStart, absEnd, lang)
		selectorKind, receiverType := analyzeSelectorMatch(root, node, src, lang, absStart)
		return &MatchInfo{
			Class:        class,
			Scope:        findEnclosingScope(node, src, lang),
			NodeType:     node.Type(lang),
			SelectorKind: selectorKind,
			ReceiverType: receiverType,
		}, nil
	}

	return &MatchInfo{Class: ClassUnknown, Scope: "package-level"}, nil
}

func classifyNode(node *gotreesitter.Node, startByte, endByte uint32, lang *gotreesitter.Language) MatchClass {
	if hasAncestorType(node, lang, "import_spec", "import_declaration") {
		return ClassImport
	}

	for current := node; current != nil; current = current.Parent() {
		typ := current.Type(lang)
		switch typ {
		case "comment", "line_comment", "block_comment":
			return ClassComment
		case "interpreted_string_literal", "raw_string_literal", "string", "template_string":
			return ClassString
		}

		switch typ {
		case "function_declaration", "method_declaration", "type_alias", "type_spec", "const_spec", "var_spec":
			if fieldContainsRange(current, "name", startByte, endByte, lang) {
				return ClassDef
			}
		case "call_expression":
			if fieldContainsRange(current, "function", startByte, endByte, lang) {
				return ClassCall
			}
		}
	}

	return ClassRef
}

func hasAncestorType(node *gotreesitter.Node, lang *gotreesitter.Language, types ...string) bool {
	for current := node; current != nil; current = current.Parent() {
		typ := current.Type(lang)
		for _, want := range types {
			if typ == want {
				return true
			}
		}
	}
	return false
}

func findEnclosingScope(node *gotreesitter.Node, src []byte, lang *gotreesitter.Language) string {
	for current := node; current != nil; current = current.Parent() {
		switch current.Type(lang) {
		case "function_declaration":
			if nameNode := current.ChildByFieldName("name", lang); nameNode != nil {
				return "func " + strings.TrimSpace(nameNode.Text(src))
			}
		case "method_declaration":
			if nameNode := current.ChildByFieldName("name", lang); nameNode != nil {
				return "method " + strings.TrimSpace(nameNode.Text(src))
			}
		}
	}
	return "package-level"
}

func fieldContainsRange(node *gotreesitter.Node, fieldName string, startByte, endByte uint32, lang *gotreesitter.Language) bool {
	if node == nil {
		return false
	}
	fieldNode := node.ChildByFieldName(fieldName, lang)
	if fieldNode == nil {
		return false
	}
	if startByte < fieldNode.StartByte() || fieldNode.EndByte() < endByte {
		return false
	}
	if fieldNode.Type(lang) != "selector_expression" {
		return true
	}
	for i := 0; i < fieldNode.NamedChildCount(); i++ {
		child := fieldNode.NamedChild(i)
		if child == nil {
			continue
		}
		typ := child.Type(lang)
		if typ != "identifier" && typ != "field_identifier" {
			continue
		}
		if child.StartByte() <= startByte && endByte <= child.EndByte() {
			return true
		}
	}
	return false
}

// lineByteRange は 1 始まりの行番号に対応するバイト範囲を返す。
// 注意: O(n) スキャン。バッチ分類時は行オフセットテーブルの事前構築を検討。
func lineByteRange(src []byte, line int) (uint32, uint32, bool) {
	if line <= 0 {
		return 0, 0, false
	}
	currentLine := 1
	start := 0
	for i, b := range src {
		if currentLine == line {
			end := i
			for end < len(src) && src[end] != '\n' {
				end++
			}
			return uint32(start), uint32(end), true
		}
		if b == '\n' {
			currentLine++
			start = i + 1
		}
	}
	if currentLine == line {
		return uint32(start), uint32(len(src)), true
	}
	return 0, 0, false
}

func findIdentifierOccurrences(line []byte, target string) []uint32 {
	if len(line) == 0 || target == "" {
		return nil
	}

	lineStr := string(line)
	var offsets []uint32
	searchFrom := 0
	for {
		idx := strings.Index(lineStr[searchFrom:], target)
		if idx < 0 {
			break
		}
		idx += searchFrom
		start := idx
		end := idx + len(target)
		if isIdentifierBoundary(lineStr, start-1) && isIdentifierBoundary(lineStr, end) {
			offsets = append(offsets, uint32(start))
		}
		searchFrom = idx + len(target)
	}
	return offsets
}

func isIdentifierBoundary(s string, pos int) bool {
	if pos < 0 || pos >= len(s) {
		return true
	}
	b := s[pos]
	return b != '_' && (b < '0' || b > '9') && (b < 'A' || b > 'Z') && (b < 'a' || b > 'z')
}
