package navigation

import (
	goast "go/ast"
	"go/token"
)

// isIdentShadowedInGoFunc は Go 関数内でインポート名がローカル変数にシャドーイングされているかを判定する。
func isIdentShadowedInGoFunc(file *goast.File, fset *token.FileSet, line int, name string) bool {
	for _, decl := range file.Decls {
		fn, ok := decl.(*goast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		startLine := fset.Position(fn.Pos()).Line
		endLine := fset.Position(fn.End()).Line
		if line < startLine || line > endLine {
			continue
		}
		// 関数パラメータをチェック
		if fn.Type != nil && fn.Type.Params != nil {
			for _, param := range fn.Type.Params.List {
				for _, paramName := range param.Names {
					if paramName.Name == name {
						return true
					}
				}
			}
		}
		// レシーバをチェック
		if fn.Recv != nil {
			for _, param := range fn.Recv.List {
				for _, paramName := range param.Names {
					if paramName.Name == name {
						return true
					}
				}
			}
		}
		// 関数本体のローカル宣言をチェック
		return hasLocalDeclInBlock(fn.Body, fset, line, name)
	}
	return false
}

// hasLocalDeclInBlock はブロック文内で useLine のスコープから見える name のローカル宣言を検出する。
// ネストブロック（if/for/switch 等）内の宣言も再帰的に走査する。
func hasLocalDeclInBlock(block *goast.BlockStmt, fset *token.FileSet, useLine int, name string) bool {
	if block == nil {
		return false
	}
	return hasLocalDeclInStmts(block.List, fset, useLine, name)
}

func hasLocalDeclInStmts(stmts []goast.Stmt, fset *token.FileSet, useLine int, name string) bool {
	for _, stmt := range stmts {
		stmtLine := fset.Position(stmt.Pos()).Line
		stmtEndLine := fset.Position(stmt.End()).Line
		if stmtLine > useLine {
			break
		}
		// useLine より前の直接宣言をチェック
		if stmtLine < useLine && matchesDeclName(stmt, name) {
			return true
		}
		// useLine を含む文のネストブロックに再帰
		if stmtEndLine >= useLine {
			if checkNestedDeclInStmt(stmt, fset, useLine, name) {
				return true
			}
		}
	}
	return false
}

// matchesDeclName は文が name を直接宣言しているかを判定する。
func matchesDeclName(stmt goast.Stmt, name string) bool {
	switch s := stmt.(type) {
	case *goast.AssignStmt:
		if s.Tok == token.DEFINE {
			for _, lhs := range s.Lhs {
				if ident, ok := lhs.(*goast.Ident); ok && ident.Name == name {
					return true
				}
			}
		}
	case *goast.DeclStmt:
		if genDecl, ok := s.Decl.(*goast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if vs, ok := spec.(*goast.ValueSpec); ok {
					for _, n := range vs.Names {
						if n.Name == name {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// checkNestedDeclInStmt は複合文（if/for/switch 等）のサブブロック内で name の宣言を検出する。
func checkNestedDeclInStmt(stmt goast.Stmt, fset *token.FileSet, useLine int, name string) bool {
	switch s := stmt.(type) {
	case *goast.IfStmt:
		if s.Init != nil && matchesDeclName(s.Init, name) {
			return true
		}
		if hasLocalDeclInBlock(s.Body, fset, useLine, name) {
			return true
		}
		if s.Else != nil {
			switch e := s.Else.(type) {
			case *goast.BlockStmt:
				return hasLocalDeclInBlock(e, fset, useLine, name)
			case *goast.IfStmt:
				return checkNestedDeclInStmt(e, fset, useLine, name)
			}
		}
	case *goast.ForStmt:
		if s.Init != nil && matchesDeclName(s.Init, name) {
			return true
		}
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.RangeStmt:
		if s.Tok == token.DEFINE {
			if key, ok := s.Key.(*goast.Ident); ok && key.Name == name {
				return true
			}
			if s.Value != nil {
				if value, ok := s.Value.(*goast.Ident); ok && value.Name == name {
					return true
				}
			}
		}
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.SwitchStmt:
		if s.Init != nil && matchesDeclName(s.Init, name) {
			return true
		}
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.TypeSwitchStmt:
		if s.Init != nil && matchesDeclName(s.Init, name) {
			return true
		}
		if s.Assign != nil && matchesDeclName(s.Assign, name) {
			return true
		}
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.SelectStmt:
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.CaseClause:
		return hasLocalDeclInStmts(s.Body, fset, useLine, name)
	case *goast.CommClause:
		if s.Comm != nil && matchesDeclName(s.Comm, name) {
			return true
		}
		return hasLocalDeclInStmts(s.Body, fset, useLine, name)
	case *goast.BlockStmt:
		return hasLocalDeclInBlock(s, fset, useLine, name)
	}
	return false
}
