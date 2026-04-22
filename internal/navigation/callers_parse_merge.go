package navigation

import "github.com/susugadx/xelyon-cli/internal/ast"

func referenceClassificationFromReference(ref Reference) referenceClassification {
	return referenceClassification{
		Scope:        ref.Scope,
		Class:        ref.Class,
		NodeType:     ref.NodeType,
		SelectorKind: ref.SelectorKind,
		ReceiverType: ref.ReceiverType,
	}
}

func mergeFallbackReferenceClassification(current, fallback referenceClassification) referenceClassification {
	if (current.Scope == "" || current.Scope == "package-level") && fallback.Scope != "" {
		current.Scope = fallback.Scope
	}
	if fallback.Class == ast.ClassDef {
		current.Class = ast.ClassDef
	} else if current.Class == ast.ClassUnknown && fallback.Class != ast.ClassUnknown {
		current.Class = fallback.Class
	}
	if current.NodeType == "" && fallback.NodeType != "" {
		current.NodeType = fallback.NodeType
	}
	if (current.SelectorKind == "" || current.SelectorKind == "unknown") && fallback.SelectorKind != "" {
		current.SelectorKind = fallback.SelectorKind
	}
	if current.ReceiverType == "" && fallback.ReceiverType != "" {
		current.ReceiverType = fallback.ReceiverType
	}
	return current
}
