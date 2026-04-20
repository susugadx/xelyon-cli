package ast

import (
	"regexp"
	"strings"

	"github.com/odvcencio/gotreesitter"
)

func analyzeSelectorMatch(root, node *gotreesitter.Node, src []byte, lang *gotreesitter.Language, matchByte uint32) (string, string) {
	if node == nil || node.Type(lang) != "field_identifier" {
		return "", ""
	}

	selector := findAncestorByType(node, lang, "selector_expression")
	if selector == nil {
		return "unknown", ""
	}

	operand := selector.ChildByFieldName("operand", lang)
	if operand == nil {
		operand = firstNamedChild(selector)
	}
	if operand == nil {
		return "unknown", ""
	}

	if selectorOperandIsImportedPackage(root, operand, src, lang, matchByte) {
		return "package", ""
	}
	if receiverType := inferReceiverTypeFromOperand(node, operand, src, lang, matchByte); receiverType != "" {
		return "method", receiverType
	}
	return "unknown", ""
}

func findAncestorByType(node *gotreesitter.Node, lang *gotreesitter.Language, want string) *gotreesitter.Node {
	for current := node; current != nil; current = current.Parent() {
		if current.Type(lang) == want {
			return current
		}
	}
	return nil
}

func selectorOperandIsImportedPackage(root, operand *gotreesitter.Node, src []byte, lang *gotreesitter.Language, matchByte uint32) bool {
	if root == nil || operand == nil || operand.Type(lang) != "identifier" {
		return false
	}

	name := strings.TrimSpace(operand.Text(src))
	if name == "" {
		return false
	}
	if !collectImportedPackageNames(root, src, lang)[name] {
		return false
	}
	// 型推論でローカル変数を検出
	if inferIdentifierType(operand, src, lang, matchByte) != "" {
		return false
	}
	// 型が不明でもローカル宣言が存在すればインポートのシャドーイング
	scope := findEnclosingCallable(operand, lang)
	if scope != nil && matchByte > scope.StartByte() && matchByte <= uint32(len(src)) {
		if detectLocalDeclaration(string(src[scope.StartByte():matchByte]), name) {
			return false
		}
	}
	return true
}

// detectLocalDeclaration はプレフィックスコード内に name のローカル宣言が存在するか検出する。
// inferIdentifierType が型を特定できない場合の補完チェック用。
func detectLocalDeclaration(prefix, name string) bool {
	quotedName := regexp.QuoteMeta(name)
	// 複数値の短変数宣言: name, x := or x, name :=
	shortVarRe := regexp.MustCompile(
		`(?m)(?:\b\w+[ \t]*,[ \t]*)*\b` + quotedName + `\b(?:[ \t]*,[ \t]*\w+)*[ \t]*:=`)
	if shortVarRe.MatchString(prefix) {
		return true
	}
	// グループ化 var 宣言: var name, other Type or var ( ... name Type ... )
	groupedVarRe := regexp.MustCompile(
		`(?m)\bvar\s+(?:\w+\s*,\s*)*` + quotedName + `\b`)
	return groupedVarRe.MatchString(prefix)
}

func collectImportedPackageNames(root *gotreesitter.Node, src []byte, lang *gotreesitter.Language) map[string]bool {
	packages := make(map[string]bool)
	if root == nil {
		return packages
	}

	stack := []*gotreesitter.Node{root}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			continue
		}
		if current.Type(lang) == "import_spec" {
			if name := importNameFromSpec(strings.TrimSpace(current.Text(src))); name != "" {
				packages[name] = true
			}
		}
		for i := int(current.NamedChildCount()) - 1; i >= 0; i-- {
			child := current.NamedChild(i)
			if child != nil {
				stack = append(stack, child)
			}
		}
	}
	return packages
}

func importNameFromSpec(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}

	fields := strings.Fields(spec)
	if len(fields) == 0 {
		return ""
	}

	if len(fields) == 1 {
		return defaultImportName(fields[0])
	}

	alias := strings.TrimSpace(fields[0])
	if alias == "" || alias == "." || alias == "_" {
		return ""
	}
	return alias
}

func defaultImportName(pathLiteral string) string {
	pathLiteral = strings.Trim(pathLiteral, "`\"")
	if pathLiteral == "" {
		return ""
	}
	if idx := strings.LastIndex(pathLiteral, "/"); idx >= 0 && idx < len(pathLiteral)-1 {
		return pathLiteral[idx+1:]
	}
	return pathLiteral
}

func inferReceiverTypeFromOperand(node, operand *gotreesitter.Node, src []byte, lang *gotreesitter.Language, matchByte uint32) string {
	if operand == nil {
		return ""
	}

	switch operand.Type(lang) {
	case "parenthesized_expression":
		return inferReceiverTypeFromOperand(node, firstNamedChild(operand), src, lang, matchByte)
	case "unary_expression":
		return inferReceiverTypeFromOperand(node, lastNamedChild(operand), src, lang, matchByte)
	case "composite_literal":
		if typeNode := operand.ChildByFieldName("type", lang); typeNode != nil {
			return strings.TrimSpace(typeNode.Text(src))
		}
		return inferTypeFromCompositeLiteralText(strings.TrimSpace(operand.Text(src)))
	case "identifier":
		return inferIdentifierType(operand, src, lang, matchByte)
	case "type_identifier", "generic_type", "pointer_type", "array_type", "slice_type", "map_type":
		return strings.TrimSpace(operand.Text(src))
	default:
		return inferTypeFromCompositeLiteralText(strings.TrimSpace(operand.Text(src)))
	}
}

func firstNamedChild(node *gotreesitter.Node) *gotreesitter.Node {
	if node == nil || node.NamedChildCount() == 0 {
		return nil
	}
	return node.NamedChild(0)
}

func lastNamedChild(node *gotreesitter.Node) *gotreesitter.Node {
	if node == nil || node.NamedChildCount() == 0 {
		return nil
	}
	return node.NamedChild(int(node.NamedChildCount()) - 1)
}
