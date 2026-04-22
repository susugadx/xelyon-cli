package navigation

import (
	"sort"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func extractASTSymbols(path string) ([]ast.Symbol, error) {
	return ast.ExtractSymbols(path)
}

func sortSymbolCandidates(candidates []SymbolCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].File != candidates[j].File {
			return candidates[i].File < candidates[j].File
		}
		if candidates[i].Line != candidates[j].Line {
			return candidates[i].Line < candidates[j].Line
		}
		if candidates[i].EndLine != candidates[j].EndLine {
			return candidates[i].EndLine < candidates[j].EndLine
		}
		return candidateDisplayName(candidates[i]) < candidateDisplayName(candidates[j])
	})
}
