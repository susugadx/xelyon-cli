package navigation

import (
	goast "go/ast"
	"go/token"
	"strings"
)

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
