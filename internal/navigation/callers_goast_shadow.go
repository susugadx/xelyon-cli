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
