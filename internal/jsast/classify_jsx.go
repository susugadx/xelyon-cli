package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
)

func isJSXUsageTarget(parsed *ParsedFile, node *gotreesitter.Node, targetName string, startByte uint32, endByte uint32) bool {
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		return false
	}
	for current := node; current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "jsx_opening_element", "jsx_self_closing_element":
			return jsxTargetIsBareLocalName(parsed, jsxElementName(parsed, current), targetName, startByte, endByte)
		case "jsx_element":
			if opening := jsxOpeningElement(parsed, current); opening != nil {
				return jsxTargetIsBareLocalName(parsed, jsxElementName(parsed, opening), targetName, startByte, endByte)
			}
			return false
		case "jsx_closing_element":
			return false
		}
	}
	return false
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
