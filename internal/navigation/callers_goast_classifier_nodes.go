package navigation

import (
	goast "go/ast"
	"go/token"
)

func classifyGoASTFuncDecl(ctx goASTLineClassificationContext, result *Reference, fn *goast.FuncDecl) bool {
	if !identMatchesLine(ctx.fset, fn.Name, ctx.symbol, ctx.line) {
		return false
	}
	applyGoASTDefinitionHint(result)
	return true
}

func classifyGoASTCallExpr(ctx goASTLineClassificationContext, result *Reference, call *goast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *goast.Ident:
		if !identMatchesLine(ctx.fset, fun, ctx.symbol, ctx.line) {
			return false
		}
		applyGoASTIdentCallHint(result)
		return true
	case *goast.SelectorExpr:
		if !identMatchesLine(ctx.fset, fun.Sel, ctx.symbol, ctx.line) {
			return false
		}
		selectorKind := selectorKindFromGoExpr(fun.X, ctx.imports, ctx.file, ctx.fset, ctx.line)
		receiverType := ""
		if selectorKind == "method" {
			receiverType = receiverTypeFromGoExpr(fun.X)
		}
		applyGoASTSelectorCallHint(result, selectorKind, receiverType)
		return true
	default:
		return false
	}
}

func classifyGoASTSelectorExpr(ctx goASTLineClassificationContext, result *Reference, selector *goast.SelectorExpr) bool {
	if !identMatchesLine(ctx.fset, selector.Sel, ctx.symbol, ctx.line) {
		return false
	}
	selectorKind := result.SelectorKind
	if selectorKind == "" || selectorKind == "unknown" {
		selectorKind = selectorKindFromGoExpr(selector.X, ctx.imports, ctx.file, ctx.fset, ctx.line)
	}
	receiverType := result.ReceiverType
	if receiverType == "" && selectorKind == "method" {
		receiverType = receiverTypeFromGoExpr(selector.X)
	}
	applyGoASTSelectorRefHint(result, selectorKind, receiverType)
	return true
}

func classifyGoASTIdent(ctx goASTLineClassificationContext, result *Reference, ident *goast.Ident) bool {
	if !identMatchesLine(ctx.fset, ident, ctx.symbol, ctx.line) {
		return false
	}
	applyGoASTIdentRefHint(result)
	return true
}

func identMatchesLine(fset *token.FileSet, ident *goast.Ident, symbol string, line int) bool {
	return ident != nil && ident.Name == symbol && fset.Position(ident.Pos()).Line == line
}
