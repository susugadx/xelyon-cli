package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
)

type JSXLocalNameUsage struct {
	Name    string
	Line    int
	Snippet string
}

type JSXLocalNameUsageVisitor func(JSXLocalNameUsage) bool

func JSXLocalNameUsagesWithParsed(parsed *ParsedFile, localName string) []JSXLocalNameUsage {
	var usages []JSXLocalNameUsage
	VisitJSXLocalNameUsagesWithParsed(parsed, localName, func(usage JSXLocalNameUsage) bool {
		usages = append(usages, usage)
		return true
	})
	return usages
}

func VisitJSXLocalNameUsagesWithParsed(parsed *ParsedFile, localName string, visit JSXLocalNameUsageVisitor) {
	visitJSXLocalNameUsagesWithParsed(parsed, localName, nil, visit)
}

func JSXLocalNameUsagesForNamedImportAliasWithParsed(parsed *ParsedFile, alias NamedImportAlias) []JSXLocalNameUsage {
	var usages []JSXLocalNameUsage
	VisitJSXLocalNameUsagesForNamedImportAliasWithParsed(parsed, alias, func(usage JSXLocalNameUsage) bool {
		usages = append(usages, usage)
		return true
	})
	return usages
}

func VisitJSXLocalNameUsagesForNamedImportAliasWithParsed(parsed *ParsedFile, alias NamedImportAlias, visit JSXLocalNameUsageVisitor) {
	shadowScopes := namedImportAliasShadowScopes(parsed, alias)
	visitJSXLocalNameUsagesWithParsed(parsed, alias.Local, func(nameNode *gotreesitter.Node) bool {
		return !jsxLocalNameShadowedByScopes(nameNode, shadowScopes)
	}, visit)
}

func visitJSXLocalNameUsagesWithParsed(parsed *ParsedFile, localName string, allow func(*gotreesitter.Node) bool, visit JSXLocalNameUsageVisitor) {
	if parsed == nil || parsed.tree == nil || visit == nil {
		return
	}
	localName = strings.TrimSpace(localName)
	if localName == "" || !jsxBareLocalNameIsComponent(localName) {
		return
	}

	seenLines := make(map[int]struct{})
	walkNamedUntil(parsed.tree.RootNode(), func(node *gotreesitter.Node) bool {
		kind := nodeKind(parsed, node)
		if kind != "jsx_opening_element" && kind != "jsx_self_closing_element" {
			return true
		}
		nameNode := jsxElementName(parsed, node)
		if nameNode == nil || !jsxTargetIsBareLocalName(parsed, nameNode, localName, nameNode.StartByte(), nameNode.EndByte()) {
			return true
		}
		if allow != nil && !allow(nameNode) {
			return true
		}
		line := int(node.StartPoint().Row) + 1
		if _, ok := seenLines[line]; ok {
			return true
		}
		seenLines[line] = struct{}{}
		return visit(JSXLocalNameUsage{
			Name:    localName,
			Line:    line,
			Snippet: parsedLineSnippet(parsed, line),
		})
	})
}

func walkNamedUntil(node *gotreesitter.Node, visit func(*gotreesitter.Node) bool) bool {
	if node == nil {
		return true
	}
	if !visit(node) {
		return false
	}
	for i := 0; i < node.NamedChildCount(); i++ {
		if !walkNamedUntil(node.NamedChild(i), visit) {
			return false
		}
	}
	return true
}

func parsedLineSnippet(parsed *ParsedFile, line int) string {
	start, end, ok := lineByteRange(parsed.src, line)
	if !ok {
		return ""
	}
	return strings.TrimSpace(string(parsed.src[start:end]))
}
