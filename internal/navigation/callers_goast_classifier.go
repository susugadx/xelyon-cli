package navigation

import (
	goast "go/ast"
	"go/token"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func classifyLineWithGoAST(file *goast.File, fset *token.FileSet, imports map[string]bool, line int, symbol string) (Reference, bool) {
	if !isGoASTClassificationInputValid(file, fset, line, symbol) {
		return Reference{}, false
	}

	ctx := goASTLineClassificationContext{
		file:    file,
		fset:    fset,
		imports: imports,
		line:    line,
		symbol:  symbol,
	}
	result := Reference{
		Scope: enclosingScopeFromGoAST(file, fset, line),
		Class: ast.ClassUnknown,
	}
	if !classifyLineByGoASTNodes(ctx, &result) {
		return Reference{}, false
	}
	return result, true
}

type goASTLineClassificationContext struct {
	file    *goast.File
	fset    *token.FileSet
	imports map[string]bool
	line    int
	symbol  string
}

func isGoASTClassificationInputValid(file *goast.File, fset *token.FileSet, line int, symbol string) bool {
	return file != nil && fset != nil && line > 0 && symbol != ""
}

func classifyLineByGoASTNodes(ctx goASTLineClassificationContext, result *Reference) bool {
	matched := false
	goast.Inspect(ctx.file, func(n goast.Node) bool {
		if !nodeIncludesLine(ctx.fset, n, ctx.line) {
			return true
		}
		if classifyGoASTNode(ctx, result, n) {
			matched = true
		}
		return true
	})
	return matched
}

func nodeIncludesLine(fset *token.FileSet, n goast.Node, line int) bool {
	if fset == nil || n == nil {
		return false
	}
	startLine := fset.Position(n.Pos()).Line
	endLine := fset.Position(n.End()).Line
	return line >= startLine && line <= endLine
}

func classifyGoASTNode(ctx goASTLineClassificationContext, result *Reference, n goast.Node) bool {
	switch node := n.(type) {
	case *goast.FuncDecl:
		return classifyGoASTFuncDecl(ctx, result, node)
	case *goast.CallExpr:
		return classifyGoASTCallExpr(ctx, result, node)
	case *goast.SelectorExpr:
		return classifyGoASTSelectorExpr(ctx, result, node)
	case *goast.Ident:
		return classifyGoASTIdent(ctx, result, node)
	default:
		return false
	}
}
