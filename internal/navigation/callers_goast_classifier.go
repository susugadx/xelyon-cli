package navigation

import (
	goast "go/ast"
	"go/token"
	"strings"

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

func enclosingScopeFromGoAST(file *goast.File, fset *token.FileSet, line int) string {
	for _, decl := range file.Decls {
		fn, ok := decl.(*goast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		startLine := fset.Position(fn.Pos()).Line
		endLine := fset.Position(fn.End()).Line
		if line < startLine || line > endLine {
			continue
		}
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			return "method " + fn.Name.Name
		}
		return "func " + fn.Name.Name
	}
	return "package-level"
}

func importedPackageNames(file *goast.File) map[string]bool {
	imports := make(map[string]bool)
	for _, spec := range file.Imports {
		if spec == nil {
			continue
		}
		if spec.Name != nil {
			name := strings.TrimSpace(spec.Name.Name)
			if name != "" && name != "." && name != "_" {
				imports[name] = true
			}
			continue
		}
		pathValue := strings.Trim(spec.Path.Value, "\"")
		if pathValue == "" {
			continue
		}
		if idx := strings.LastIndex(pathValue, "/"); idx >= 0 {
			pathValue = pathValue[idx+1:]
		}
		if pathValue != "" {
			imports[pathValue] = true
		}
	}
	return imports
}

func selectorKindFromGoExpr(expr goast.Expr, imports map[string]bool, file *goast.File, fset *token.FileSet, line int) string {
	ident, ok := expr.(*goast.Ident)
	if !ok {
		return "method"
	}
	if !imports[ident.Name] {
		return "method"
	}
	// ローカル変数がインポート名をシャドーイングしている場合はメソッド呼び出し。
	if isIdentShadowedInGoFunc(file, fset, line, ident.Name) {
		return "method"
	}
	return "package"
}

func receiverTypeFromGoExpr(expr goast.Expr) string {
	switch node := expr.(type) {
	case *goast.CompositeLit:
		return receiverTypeFromGoExpr(node.Type)
	case *goast.Ident:
		return canonicalReceiver(node.Name)
	case *goast.StarExpr:
		return receiverTypeFromGoExpr(node.X)
	case *goast.ParenExpr:
		return receiverTypeFromGoExpr(node.X)
	case *goast.SelectorExpr:
		// フィールドチェーン（foo.Bar.Method()）では Sel は型ではなくフィールド名の可能性が高い。
		// 型チェッカーなしでは実型を解決できないため空文字を返す。
		return ""
	}
	return ""
}
