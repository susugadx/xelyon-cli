package navigation

import (
	goast "go/ast"
	"go/token"
)

// checkNestedDeclInStmt は複合文（if/for/switch 等）のサブブロック内で name の宣言を検出する。
func checkNestedDeclInStmt(stmt goast.Stmt, fset *token.FileSet, useLine int, name string) bool {
	switch s := stmt.(type) {
	case *goast.IfStmt:
		return checkNestedDeclInIfStmt(s, fset, useLine, name)
	case *goast.ForStmt:
		return checkNestedDeclInForStmt(s, fset, useLine, name)
	case *goast.RangeStmt:
		return checkNestedDeclInRangeStmt(s, fset, useLine, name)
	case *goast.SwitchStmt:
		return checkNestedDeclInSwitchStmt(s, fset, useLine, name)
	case *goast.TypeSwitchStmt:
		return checkNestedDeclInTypeSwitchStmt(s, fset, useLine, name)
	case *goast.SelectStmt:
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.CaseClause:
		return hasLocalDeclInStmts(s.Body, fset, useLine, name)
	case *goast.CommClause:
		return checkNestedDeclInCommClause(s, fset, useLine, name)
	case *goast.BlockStmt:
		return hasLocalDeclInBlock(s, fset, useLine, name)
	default:
		return false
	}
}

func checkNestedDeclInIfStmt(stmt *goast.IfStmt, fset *token.FileSet, useLine int, name string) bool {
	if stmt == nil {
		return false
	}
	if stmt.Init != nil && matchesDeclName(stmt.Init, name) {
		return true
	}
	if hasLocalDeclInBlock(stmt.Body, fset, useLine, name) {
		return true
	}
	if stmt.Else == nil {
		return false
	}
	switch e := stmt.Else.(type) {
	case *goast.BlockStmt:
		return hasLocalDeclInBlock(e, fset, useLine, name)
	case *goast.IfStmt:
		return checkNestedDeclInStmt(e, fset, useLine, name)
	default:
		return false
	}
}

func checkNestedDeclInForStmt(stmt *goast.ForStmt, fset *token.FileSet, useLine int, name string) bool {
	if stmt == nil {
		return false
	}
	if stmt.Init != nil && matchesDeclName(stmt.Init, name) {
		return true
	}
	return hasLocalDeclInBlock(stmt.Body, fset, useLine, name)
}

func checkNestedDeclInRangeStmt(stmt *goast.RangeStmt, fset *token.FileSet, useLine int, name string) bool {
	if stmt == nil {
		return false
	}
	if stmt.Tok == token.DEFINE {
		if key, ok := stmt.Key.(*goast.Ident); ok && key.Name == name {
			return true
		}
		if value, ok := stmt.Value.(*goast.Ident); ok && value.Name == name {
			return true
		}
	}
	return hasLocalDeclInBlock(stmt.Body, fset, useLine, name)
}

func checkNestedDeclInSwitchStmt(stmt *goast.SwitchStmt, fset *token.FileSet, useLine int, name string) bool {
	if stmt == nil {
		return false
	}
	if stmt.Init != nil && matchesDeclName(stmt.Init, name) {
		return true
	}
	return hasLocalDeclInBlock(stmt.Body, fset, useLine, name)
}

func checkNestedDeclInTypeSwitchStmt(stmt *goast.TypeSwitchStmt, fset *token.FileSet, useLine int, name string) bool {
	if stmt == nil {
		return false
	}
	if stmt.Init != nil && matchesDeclName(stmt.Init, name) {
		return true
	}
	if stmt.Assign != nil && matchesDeclName(stmt.Assign, name) {
		return true
	}
	return hasLocalDeclInBlock(stmt.Body, fset, useLine, name)
}

func checkNestedDeclInCommClause(stmt *goast.CommClause, fset *token.FileSet, useLine int, name string) bool {
	if stmt == nil {
		return false
	}
	if stmt.Comm != nil && matchesDeclName(stmt.Comm, name) {
		return true
	}
	return hasLocalDeclInStmts(stmt.Body, fset, useLine, name)
}
