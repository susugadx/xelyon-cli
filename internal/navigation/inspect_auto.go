package navigation

import "github.com/susugadx/xelyon-cli/internal/locator"

// SymbolAutoStatus はシンボル自動解決の結果種別。
type SymbolAutoStatus string

const (
	SymbolAutoSingle   SymbolAutoStatus = "single"
	SymbolAutoMultiple SymbolAutoStatus = "multiple"
	SymbolAutoNone     SymbolAutoStatus = "none"
)

// InspectSymbolAuto はシンボル名の自動解決を試みる。
// single: 単一候補が見つかった → summary 形式の結果を返す
// multiple: 複数候補 → 候補一覧を返す
// none: 見つからない → 空文字を返す（呼び出し側で text search にフォールバック）
// reg が nil でない場合、出力に Locator ID を付与する。
func InspectSymbolAuto(symbol, pathHint string, reg *locator.Registry, lspClient LSPClient) (output string, status SymbolAutoStatus) {
	if symbol == "" {
		return "", SymbolAutoNone
	}

	query := parseSymbolQuery(symbol)
	candidates := resolveSymbolCandidates(symbol, pathHint)

	if len(candidates) == 0 {
		return "", SymbolAutoNone
	}

	if len(candidates) > 1 {
		return formatMultipleCandidates(symbol, candidates, reg), SymbolAutoMultiple
	}

	// 単一候補 → SummaryBudget で結果を生成
	cand := candidates[0]
	result := InspectResult{Symbol: &cand}

	result.Body = readDefinitionBody(cand, SummaryBudget.BodyLines)

	var allRefs []Reference
	if lspClient != nil {
		lspRefs, err := findReferencesViaLSP(lspClient, cand)
		if err == nil && len(lspRefs) > 0 {
			allRefs = lspRefs
			result.ResolvedViaLSP = true
		} else {
			allRefs, result.UpstreamTruncated, result.UpstreamIncomplete = findReferencesWithFallback(query.BaseName, cand)
		}
		if cand.Kind == "interface" {
			if impls, err := findImplementationsViaLSP(lspClient, cand); err == nil {
				result.Implementations = impls
			}
		}
	} else {
		allRefs, result.UpstreamTruncated, result.UpstreamIncomplete = findReferencesWithFallback(query.BaseName, cand)
	}
	result.Callers, result.TotalCallers, result.MoreCallers = classifyCallers(allRefs, cand, SummaryBudget.CallerLimit)
	result.Refs, result.TotalRefs, result.MoreRefs = classifyRefs(allRefs, cand, SummaryBudget.RefLimit)
	result.Tests, result.TotalTests, result.MoreTests = findRelatedTests(query.BaseName, allRefs, SummaryBudget.TestLimit)

	return formatInspectResult(result, reg), SymbolAutoSingle
}
