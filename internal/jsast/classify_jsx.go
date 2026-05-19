package jsast

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/odvcencio/gotreesitter"
	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

func classifyJSXTarget(parsed *ParsedFile, node *gotreesitter.Node, targetName string, startByte uint32, endByte uint32) (codeast.MatchClass, bool) {
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		return "", false
	}
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "jsx_opening_element", "jsx_self_closing_element":
			return classifyJSXOpeningTarget(parsed, current, targetName, startByte, endByte)
		case "jsx_element":
			if opening := jsxOpeningElement(parsed, current); opening != nil {
				return classifyJSXOpeningTarget(parsed, opening, targetName, startByte, endByte)
			}
			return "", false
		case "jsx_closing_element":
			return classifyJSXClosingTarget(parsed, current, targetName, startByte, endByte)
		}
	}
	return "", false
}

func classifyJSXOpeningTarget(parsed *ParsedFile, node *gotreesitter.Node, targetName string, startByte uint32, endByte uint32) (codeast.MatchClass, bool) {
	if !jsxTargetIsBareLocalName(parsed, jsxElementName(parsed, node), targetName, startByte, endByte) {
		return "", false
	}
	if !jsxBareLocalNameIsComponent(targetName) {
		return ClassIgnored, true
	}
	return codeast.ClassCall, true
}

func classifyJSXClosingTarget(parsed *ParsedFile, node *gotreesitter.Node, targetName string, startByte uint32, endByte uint32) (codeast.MatchClass, bool) {
	if !jsxTargetIsBareLocalName(parsed, jsxElementName(parsed, node), targetName, startByte, endByte) {
		return "", false
	}
	return ClassIgnored, true
}

func jsxOpeningElement(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	if node == nil {
		return nil
	}
	if opening := childByField(parsed, node, "open_tag"); opening != nil {
		return opening
	}
	for i := 0; i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		kind := nodeKind(parsed, child)
		if kind == "jsx_opening_element" || kind == "jsx_self_closing_element" {
			return child
		}
	}
	return nil
}

func jsxElementName(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	if node == nil {
		return nil
	}
	if name := childByField(parsed, node, "name"); name != nil {
		return name
	}
	return node.NamedChild(0)
}

func jsxTargetIsBareLocalName(parsed *ParsedFile, node *gotreesitter.Node, targetName string, startByte uint32, endByte uint32) bool {
	if node == nil {
		return false
	}
	if node.StartByte() > startByte || endByte > node.EndByte() {
		return false
	}
	if strings.TrimSpace(nodeText(parsed, node)) != targetName {
		return false
	}
	start, end := node.StartByte(), node.EndByte()
	if start > 0 && parsed.src[int(start)-1] == '.' {
		return false
	}
	if int(end) < len(parsed.src) && parsed.src[int(end)] == '.' {
		return false
	}
	return true
}

func jsxBareLocalNameIsComponent(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return false
	}
	return r == '_' || r == '$' || unicode.IsUpper(r)
}
