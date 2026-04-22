package navigation

import "github.com/susugadx/xelyon-cli/internal/ast"

// filterRefsByCandidate は候補に安全に帰属できない参照を除外する。
//   - 候補自身の定義行は除外（本文で表示するため）
//   - 他ファイルの ClassDef は除外
//   - ambiguousFiles（同名シンボルを定義するファイル）内では
//     selector / receiver で厳密に帰属できる参照だけを残す
//
// AST のみでは完全な型解決はできないため、曖昧ファイル内の plain identifier は
// 保守的に除外する。一方で package selector や receiver-qualified method は
// 形状で安全に帰属できるため許可する。
func filterRefsByCandidate(refs []Reference, cand SymbolCandidate, ambiguousFiles map[string]bool) []Reference {
	var filtered []Reference
	for _, ref := range refs {
		// 候補自身の定義行は除外（定義行は本文で表示するため）。
		if ref.File == cand.File && ref.Line >= cand.Line && ref.Line <= cand.EndLine {
			continue
		}
		// 他ファイルの定義行自体は除外。
		if ref.Class == ast.ClassDef && ref.File != cand.File {
			continue
		}
		decision := candidateShapeMatch(ref, cand)
		if !decision.Matched {
			continue
		}
		// 同名シンボルを定義するファイル内では、厳密に帰属できる参照だけ許可する。
		if ambiguousFiles[ref.File] && !decision.Precise {
			continue
		}
		filtered = append(filtered, ref)
	}
	return filtered
}
