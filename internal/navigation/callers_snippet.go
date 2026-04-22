package navigation

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func applySnippetReferenceHints(snippet, symbol string, class ast.MatchClass, nodeType, selectorKind, receiverType string) (ast.MatchClass, string, string, string) {
	snippet = strings.TrimSpace(snippet)
	if snippet == "" || symbol == "" {
		return class, nodeType, selectorKind, receiverType
	}

	if nodeType == "" {
		switch {
		case strings.Contains(snippet, "."+symbol):
			nodeType = "field_identifier"
		case strings.Contains(snippet, symbol):
			nodeType = "identifier"
		}
	}

	operand := selectorOperandFromSnippet(snippet, symbol)
	if selectorKind == "" && operand != "" {
		if looksLikePackageOperand(operand) {
			selectorKind = "package"
		} else {
			selectorKind = "method"
		}
	}
	if receiverType == "" && selectorKind == "method" {
		receiverType = inferReceiverTypeFromSnippetOperand(operand)
	}

	if class != ast.ClassDef && class != ast.ClassImport && class != ast.ClassComment && class != ast.ClassString {
		switch {
		case class == ast.ClassUnknown && isDefinitionSnippet(snippet, symbol):
			class = ast.ClassDef
		case strings.Contains(snippet, "."+symbol+"("):
			class = ast.ClassCall
		case containsBareSymbolCall(snippet, symbol):
			class = ast.ClassCall
		case class == ast.ClassUnknown && strings.Contains(snippet, "."+symbol):
			class = ast.ClassRef
		case class == ast.ClassUnknown && strings.Contains(snippet, symbol):
			class = ast.ClassRef
		}
	}

	return class, nodeType, selectorKind, receiverType
}

func isDefinitionSnippet(snippet, symbol string) bool {
	snippet = strings.TrimSpace(snippet)
	return strings.Contains(snippet, "func "+symbol+"(") || strings.Contains(snippet, ") "+symbol+"(")
}

func containsBareSymbolCall(snippet, symbol string) bool {
	idx := strings.Index(snippet, symbol+"(")
	if idx < 0 {
		return false
	}
	if idx > 0 && snippet[idx-1] == '.' {
		return false
	}
	return true
}

func selectorOperandFromSnippet(snippet, symbol string) string {
	idx := strings.Index(snippet, "."+symbol)
	if idx <= 0 {
		return ""
	}
	end := idx
	start := end - 1
	for start >= 0 {
		if !isSnippetOperandChar(snippet[start]) {
			break
		}
		start--
	}
	return strings.TrimSpace(snippet[start+1 : end])
}

func isSnippetOperandChar(ch byte) bool {
	switch {
	case ch >= 'a' && ch <= 'z':
		return true
	case ch >= 'A' && ch <= 'Z':
		return true
	case ch >= '0' && ch <= '9':
		return true
	}
	switch ch {
	case '_', '.', '*', '&', '(', ')', '[', ']', '{', '}':
		return true
	default:
		return false
	}
}

func looksLikePackageOperand(operand string) bool {
	operand = strings.TrimSpace(strings.TrimLeft(operand, "*&("))
	operand = strings.TrimRight(operand, ")")
	if operand == "" {
		return false
	}
	if strings.ContainsAny(operand, "{}[].") {
		return false
	}
	first := operand[0]
	return first >= 'a' && first <= 'z'
}

func inferReceiverTypeFromSnippetOperand(operand string) string {
	operand = strings.TrimSpace(operand)
	if operand == "" {
		return ""
	}
	operand = strings.TrimSpace(strings.TrimLeft(operand, "*&"))
	operand = strings.Trim(operand, "()")
	if idx := strings.Index(operand, "{"); idx > 0 {
		return canonicalReceiver(strings.TrimSpace(operand[:idx]))
	}
	if operand != "" && operand[0] >= 'A' && operand[0] <= 'Z' && !strings.ContainsAny(operand, ".[]") {
		return canonicalReceiver(operand)
	}
	return ""
}
