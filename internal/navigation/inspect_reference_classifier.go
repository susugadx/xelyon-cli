package navigation

import "github.com/susugadx/xelyon-cli/internal/ast"

// classifyCallers は参照一覧からコール箇所を抽出する。
// AST 分類（ClassCall）を使い、定義行自体・テスト・定義宣言を除外する。
func classifyCallers(refs []Reference, def SymbolCandidate, limit int) ([]Reference, int, bool) {
	var callers []Reference

	for _, ref := range refs {
		if isRefWithinDefinition(ref, def) {
			continue
		}
		// テストファイルは tests で扱う
		if ref.IsTest {
			continue
		}
		// AST 分類が ClassCall のもののみ caller
		if ref.Class == ast.ClassCall {
			callers = append(callers, ref)
		}
	}

	total := len(callers)
	if total > limit {
		return callers[:limit], total, true
	}
	return callers, total, false
}

// classifyRefs は参照一覧から一般的な参照（変数や型の使用箇所）を抽出する。
// AST 分類を使い、定義行・テスト・コール箇所を除外する。
func classifyRefs(refs []Reference, def SymbolCandidate, limit int) ([]Reference, int, bool) {
	var results []Reference

	for _, ref := range refs {
		if isRefWithinDefinition(ref, def) {
			continue
		}
		// テストファイルは tests で扱う
		if ref.IsTest {
			continue
		}
		// コール、定義宣言、コメント、文字列、インポートは除外
		switch ref.Class {
		case ast.ClassDef, ast.ClassCall, ast.ClassImport, ast.ClassComment, ast.ClassString:
			continue
		default:
			results = append(results, ref)
		}
	}

	total := len(results)
	if total > limit {
		return results[:limit], total, true
	}
	return results, total, false
}

func isRefWithinDefinition(ref Reference, def SymbolCandidate) bool {
	return ref.File == def.File && ref.Line >= def.Line && ref.Line <= def.EndLine
}
