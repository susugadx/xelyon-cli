package jsast

import (
	"strings"

	"github.com/odvcencio/gotreesitter"
)

type localBindingScopeRange struct {
	startByte uint32
	endByte   uint32
}

func namedImportAliasShadowScopes(parsed *ParsedFile, alias NamedImportAlias) []localBindingScopeRange {
	return localBindingShadowScopes(parsed, alias.Local, alias.localStartByte, alias.localEndByte)
}

func localBindingShadowScopes(parsed *ParsedFile, localName string, localStartByte uint32, localEndByte uint32) []localBindingScopeRange {
	if parsed == nil || parsed.tree == nil || localName == "" {
		return nil
	}
	var scopes []localBindingScopeRange
	walkNamed(parsed.tree.RootNode(), func(node *gotreesitter.Node) {
		nameNode, ok := localBindingNameNode(parsed, node)
		if !ok || strings.TrimSpace(nodeText(parsed, nameNode)) != localName {
			return
		}
		if nameNode.StartByte() == localStartByte && nameNode.EndByte() == localEndByte {
			return
		}
		scope := localBindingScope(parsed, node)
		if scope == nil {
			return
		}
		scopes = append(scopes, localBindingScopeRange{
			startByte: scope.StartByte(),
			endByte:   scope.EndByte(),
		})
	})
	return scopes
}

func jsxLocalNameShadowedByScopes(usageName *gotreesitter.Node, scopes []localBindingScopeRange) bool {
	if usageName == nil {
		return false
	}
	for _, scope := range scopes {
		if scope.startByte <= usageName.StartByte() && usageName.EndByte() <= scope.endByte {
			return true
		}
	}
	return false
}

func localBindingNameNode(parsed *ParsedFile, node *gotreesitter.Node) (*gotreesitter.Node, bool) {
	kind := nodeKind(parsed, node)
	switch {
	case kind == "variable_declarator":
		name := childByField(parsed, node, "name")
		if name != nil && nodeKind(parsed, name) == "identifier" {
			return name, true
		}
	case nodeKindIsBlockScopedValueDeclaration(kind):
		if name := localValueDeclarationNameNode(parsed, node); name != nil {
			return name, true
		}
	case nodeKindIsSelfScopedValueDeclaration(kind):
		if name := childByField(parsed, node, "name"); name != nil {
			return name, true
		}
	}

	switch kind {
	case "required_parameter", "optional_parameter":
		if name := firstNamedChildOfKind(parsed, node, "identifier"); name != nil {
			return name, true
		}
	case "rest_pattern":
		if name := firstNamedChildOfKind(parsed, node, "identifier"); name != nil {
			return name, true
		}
	case "shorthand_property_identifier_pattern":
		return node, true
	case "identifier":
		if identifierIsParameterBinding(parsed, node) ||
			identifierIsCatchBinding(parsed, node) ||
			identifierIsLoopBinding(parsed, node) ||
			identifierIsDestructuredBinding(parsed, node) {
			return node, true
		}
	}
	return nil, false
}

func localBindingScope(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	kind := nodeKind(parsed, node)
	switch {
	case kind == "variable_declarator":
		return variableDeclaratorBindingScope(parsed, node)
	case nodeKindIsBlockScopedValueDeclaration(kind):
		return nearestLexicalScope(parsed, node)
	case nodeKindIsSelfScopedValueDeclaration(kind):
		return node
	}

	switch kind {
	case "required_parameter", "optional_parameter":
		return nearestFunctionOrProgramScope(parsed, node)
	case "rest_pattern":
		return destructuredBindingScope(parsed, node)
	case "shorthand_property_identifier_pattern":
		return destructuredBindingScope(parsed, node)
	case "identifier":
		if identifierIsParameterBinding(parsed, node) {
			return nearestFunctionOrProgramScope(parsed, node)
		}
		if identifierIsCatchBinding(parsed, node) {
			return nearestCatchScope(parsed, node)
		}
		if loop, declarationKind, ok := loopBindingDeclaration(parsed, node); ok {
			if declarationKind == "var" {
				return nearestFunctionOrProgramScope(parsed, node)
			}
			return loop
		}
		if identifierIsDestructuredBinding(parsed, node) {
			return destructuredBindingScope(parsed, node)
		}
		return nil
	default:
		return nil
	}
}

func localValueDeclarationNameNode(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	name := childByField(parsed, node, "name")
	if nodeKind(parsed, node) == "internal_module" {
		return internalModuleTopLevelNameNode(parsed, name)
	}
	return name
}

func internalModuleTopLevelNameNode(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	switch nodeKind(parsed, node) {
	case "identifier":
		return node
	case "nested_identifier":
		return internalModuleTopLevelNameNode(parsed, childByField(parsed, node, "object"))
	default:
		return nil
	}
}

func nodeKindIsBlockScopedValueDeclaration(kind string) bool {
	switch kind {
	case "function_declaration", "generator_function_declaration", "class_declaration",
		"abstract_class_declaration", "enum_declaration", "internal_module":
		return true
	default:
		return false
	}
}

func nodeKindIsSelfScopedValueDeclaration(kind string) bool {
	switch kind {
	case "function", "function_expression", "generator_function", "class", "class_expression":
		return true
	default:
		return false
	}
}

func identifierIsParameterBinding(parsed *ParsedFile, node *gotreesitter.Node) bool {
	parent := node.Parent()
	switch nodeKind(parsed, parent) {
	case "formal_parameters":
		return true
	case "arrow_function":
		return parent.NamedChildCount() > 0 && parent.NamedChild(0) == node
	default:
		return false
	}
}

func identifierIsCatchBinding(parsed *ParsedFile, node *gotreesitter.Node) bool {
	parent := node.Parent()
	if nodeKind(parsed, parent) != "catch_clause" {
		return false
	}
	for i := 0; i < parent.NamedChildCount(); i++ {
		child := parent.NamedChild(i)
		switch nodeKind(parsed, child) {
		case "identifier":
			return child == node
		case "statement_block":
			return false
		}
	}
	return false
}

func identifierIsLoopBinding(parsed *ParsedFile, node *gotreesitter.Node) bool {
	_, _, ok := loopBindingDeclaration(parsed, node)
	return ok
}

func loopBindingDeclaration(parsed *ParsedFile, node *gotreesitter.Node) (*gotreesitter.Node, string, bool) {
	parent := node.Parent()
	if !nodeKindIsLoopStatement(nodeKind(parsed, parent)) {
		return nil, "", false
	}
	declarationKind := loopHeaderBindingDeclarationKind(parsed, parent, node)
	if declarationKind == "" {
		return nil, "", false
	}
	return parent, declarationKind, true
}

func loopHeaderBindingDeclarationKind(parsed *ParsedFile, loop *gotreesitter.Node, name *gotreesitter.Node) string {
	if loop == nil || name == nil || name.StartByte() < loop.StartByte() {
		return ""
	}
	prefix := strings.TrimSpace(string(parsed.src[loop.StartByte():name.StartByte()]))
	if loopHeaderHasSeparatorBeforeName(parsed, loop, prefix) {
		return ""
	}
	return loopHeaderDeclarationKind(prefix)
}

func loopHeaderHasSeparatorBeforeName(parsed *ParsedFile, loop *gotreesitter.Node, prefix string) bool {
	switch nodeKind(parsed, loop) {
	case "for_statement":
		return strings.Contains(prefix, ";")
	case "for_in_statement", "for_of_statement":
		return containsWordToken(prefix, "in") || containsWordToken(prefix, "of")
	default:
		return false
	}
}

func loopHeaderDeclarationKind(prefix string) string {
	if idx := strings.LastIndex(prefix, "("); idx >= 0 {
		prefix = prefix[idx+1:]
	}
	prefix = strings.TrimSpace(prefix)
	for _, marker := range []string{"let", "const", "var"} {
		if startsWithWordToken(prefix, marker) {
			return marker
		}
	}
	return ""
}

func containsWordToken(text string, word string) bool {
	for offset := 0; offset <= len(text)-len(word); {
		idx := strings.Index(text[offset:], word)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(word)
		if tokenBoundary(text, start-1) && tokenBoundary(text, end) {
			return true
		}
		offset = end
	}
	return false
}

func startsWithWordToken(text string, word string) bool {
	if !strings.HasPrefix(text, word) {
		return false
	}
	return tokenBoundary(text, len(word))
}

func tokenBoundary(text string, index int) bool {
	if index < 0 || index >= len(text) {
		return true
	}
	b := text[index]
	return !isIdentifierByte(b)
}

func isIdentifierByte(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '_' ||
		b == '$'
}

func identifierIsDestructuredBinding(parsed *ParsedFile, node *gotreesitter.Node) bool {
	parent := node.Parent()
	switch nodeKind(parsed, parent) {
	case "object_pattern", "array_pattern":
		return true
	case "pair_pattern":
		return pairPatternValueContainsNode(parsed, parent, node)
	case "assignment_pattern":
		return parent.NamedChildCount() > 0 && nodeWithin(node, parent.NamedChild(0)) && hasAncestorKind(parsed, parent, "object_pattern", "array_pattern")
	default:
		return false
	}
}

func pairPatternValueContainsNode(parsed *ParsedFile, pair *gotreesitter.Node, node *gotreesitter.Node) bool {
	if value := childByField(parsed, pair, "value"); value != nil {
		return nodeWithin(node, value)
	}
	count := pair.NamedChildCount()
	if count == 0 {
		return false
	}
	return nodeWithin(node, pair.NamedChild(count-1))
}

func destructuredBindingScope(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "variable_declarator":
			return variableDeclaratorBindingScope(parsed, current)
		case "for_statement", "for_in_statement", "for_of_statement":
			return current
		case "required_parameter", "optional_parameter", "formal_parameters", "arrow_function":
			return nearestFunctionOrProgramScope(parsed, current)
		case "catch_clause":
			return current
		default:
			if nodeKindIsLexicalScope(nodeKind(parsed, current)) {
				return current
			}
		}
	}
	return nil
}

func variableDeclaratorBindingScope(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	if variableDeclaratorIsVar(parsed, node) {
		return nearestFunctionOrProgramScope(parsed, node)
	}
	if loop := nearestLoopStatementScope(parsed, node); loop != nil {
		return loop
	}
	return nearestLexicalScope(parsed, node)
}

func nearestLexicalScope(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if nodeKindIsLexicalScope(nodeKind(parsed, current)) {
			return current
		}
	}
	return nil
}

func nearestLoopStatementScope(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if nodeKindIsLoopStatement(nodeKind(parsed, current)) {
			return current
		}
		if nodeKindIsLexicalScope(nodeKind(parsed, current)) {
			return nil
		}
	}
	return nil
}

func nodeKindIsLoopStatement(kind string) bool {
	switch kind {
	case "for_statement", "for_in_statement", "for_of_statement":
		return true
	default:
		return false
	}
}

func nodeKindIsLexicalScope(kind string) bool {
	switch kind {
	case "statement_block", "switch_body", "program":
		return true
	default:
		return false
	}
}

func nearestCatchScope(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		if nodeKind(parsed, current) == "catch_clause" {
			return current
		}
	}
	return nil
}

func variableDeclaratorIsVar(parsed *ParsedFile, node *gotreesitter.Node) bool {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "variable_declaration":
			return true
		case "lexical_declaration":
			return false
		default:
			if nodeKindIsLexicalScope(nodeKind(parsed, current)) {
				return false
			}
		}
	}
	return false
}

func nearestFunctionOrProgramScope(parsed *ParsedFile, node *gotreesitter.Node) *gotreesitter.Node {
	for current := node.Parent(); current != nil; current = current.Parent() {
		switch nodeKind(parsed, current) {
		case "function_declaration", "generator_function_declaration", "function", "function_expression",
			"generator_function", "arrow_function", "method_definition", "program":
			return current
		}
	}
	return nil
}
