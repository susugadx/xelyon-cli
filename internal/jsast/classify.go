package jsast

import (
	"github.com/odvcencio/gotreesitter"
	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

const (
	ClassTypeRef codeast.MatchClass = "type_ref"
	ClassExport  codeast.MatchClass = "export"
	ClassIgnored codeast.MatchClass = "ignored"
)

func classifyNode(parsed *ParsedFile, root *gotreesitter.Node, node *gotreesitter.Node, startByte uint32, endByte uint32, targetName string) codeast.MatchClass {
	if isCommonJSExportAssignment(parsed, root, node, targetName) {
		return ClassExport
	}
	if hasAncestorKind(parsed, node, "comment") || hasAncestorKind(parsed, node, "ERROR") {
		return codeast.ClassComment
	}
	if isStringLikeMatch(parsed, node) {
		return codeast.ClassString
	}
	if isRequireBindingReference(parsed, node) {
		return codeast.ClassImport
	}
	if isDefinitionName(parsed, node, startByte, endByte) {
		return codeast.ClassDef
	}
	if isCallTarget(parsed, node, startByte, endByte) || isNewTarget(parsed, node, startByte, endByte) || isJSXUsageTarget(parsed, node, targetName, startByte, endByte) {
		return codeast.ClassCall
	}
	if isTypeReference(parsed, node, startByte, endByte) {
		return ClassTypeRef
	}
	if class, ok := classifyImportOrExportReference(parsed, node, startByte, endByte); ok {
		return class
	}
	return codeast.ClassRef
}

func matchClassPriority(class codeast.MatchClass) int {
	switch class {
	case codeast.ClassDef:
		return 90
	case codeast.ClassImport, ClassExport:
		return 80
	case codeast.ClassCall:
		return 70
	case ClassTypeRef:
		return 60
	case codeast.ClassRef:
		return 50
	case codeast.ClassString, codeast.ClassComment:
		return 10
	case ClassIgnored:
		return 5
	default:
		return 0
	}
}
