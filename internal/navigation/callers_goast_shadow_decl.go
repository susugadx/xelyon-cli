package navigation

import (
	goast "go/ast"
	"go/token"
)

// matchesDeclName は文が name を直接宣言しているかを判定する。
func matchesDeclName(stmt goast.Stmt, name string) bool {
	switch s := stmt.(type) {
	case *goast.AssignStmt:
		return assignStmtDeclaresName(s, name)
	case *goast.DeclStmt:
		return declStmtDeclaresName(s, name)
	default:
		return false
	}
}

func assignStmtDeclaresName(stmt *goast.AssignStmt, name string) bool {
	if stmt == nil || stmt.Tok != token.DEFINE {
		return false
	}
	for _, lhs := range stmt.Lhs {
		if ident, ok := lhs.(*goast.Ident); ok && ident.Name == name {
			return true
		}
	}
	return false
}

func declStmtDeclaresName(stmt *goast.DeclStmt, name string) bool {
	if stmt == nil {
		return false
	}
	genDecl, ok := stmt.Decl.(*goast.GenDecl)
	if !ok {
		return false
	}
	for _, spec := range genDecl.Specs {
		if valueSpecDeclaresName(spec, name) {
			return true
		}
	}
	return false
}

func valueSpecDeclaresName(spec goast.Spec, name string) bool {
	vs, ok := spec.(*goast.ValueSpec)
	if !ok {
		return false
	}
	for _, declared := range vs.Names {
		if declared.Name == name {
			return true
		}
	}
	return false
}
