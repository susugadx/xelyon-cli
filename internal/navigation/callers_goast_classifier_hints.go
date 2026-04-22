package navigation

import "github.com/susugadx/xelyon-cli/internal/ast"

func applyGoASTDefinitionHint(result *Reference) {
	result.Class = ast.ClassDef
	result.NodeType = "identifier"
}

func applyGoASTIdentCallHint(result *Reference) {
	result.Class = ast.ClassCall
	result.NodeType = "identifier"
}

func applyGoASTSelectorCallHint(result *Reference, selectorKind, receiverType string) {
	result.Class = ast.ClassCall
	result.NodeType = "field_identifier"
	result.SelectorKind = selectorKind
	if result.SelectorKind == "method" {
		result.ReceiverType = receiverType
	}
}

func applyGoASTSelectorRefHint(result *Reference, selectorKind, receiverType string) {
	if result.Class == ast.ClassUnknown {
		result.Class = ast.ClassRef
	}
	if result.NodeType == "" {
		result.NodeType = "field_identifier"
	}
	if result.SelectorKind == "" || result.SelectorKind == "unknown" {
		result.SelectorKind = selectorKind
	}
	if result.ReceiverType == "" && result.SelectorKind == "method" {
		result.ReceiverType = receiverType
	}
}

func applyGoASTIdentRefHint(result *Reference) {
	if result.Class == ast.ClassUnknown {
		result.Class = ast.ClassRef
	}
	if result.NodeType == "" {
		result.NodeType = "identifier"
	}
}
