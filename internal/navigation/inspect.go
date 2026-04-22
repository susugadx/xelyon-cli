package navigation

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

// InspectMode は inspect_symbol の出力モード。
type InspectMode string

const (
	ModeSummary InspectMode = "summary"
	ModeFull    InspectMode = "full"
)

// Budget は inspect_symbol の出力上限。
type Budget struct {
	BodyLines   int
	CallerLimit int
	RefLimit    int
	TestLimit   int
}

// SummaryBudget は summary モードの出力上限。
var SummaryBudget = Budget{
	BodyLines:   15,
	CallerLimit: 2,
	RefLimit:    2,
	TestLimit:   1,
}

// FullBudget は full モードの出力上限。
var FullBudget = Budget{
	BodyLines:   999999,
	CallerLimit: 999999,
	RefLimit:    999999,
	TestLimit:   999999,
}

// SymbolCandidate はシンボル候補。
type SymbolCandidate struct {
	Name               string
	Kind               string
	File               string // プロジェクトルートからの相対パス
	Line               int
	EndLine            int
	Receiver           string // メソッド時のレシーバ型（例: *Config, Config）
	ReceiverNorm       string
	Signature          string
	Exported           bool
	PackageDir         string
	StableKey          string
	StableKeyCollision bool
	RootPath           string
}

// InspectResult は inspect_symbol の結果。
type InspectResult struct {
	// 単一候補の場合
	Symbol          *SymbolCandidate
	Body            []string // 行番号付き本文
	Callers         []Reference
	Refs            []Reference
	Tests           []TestRef
	ResolvedViaLSP  bool
	Implementations []ImplementationRef

	// 複数候補の場合
	Candidates []SymbolCandidate

	// 打ち切り情報
	TotalCallers       int
	TotalRefs          int
	TotalTests         int
	MoreCallers        bool
	MoreRefs           bool
	MoreTests          bool
	UpstreamTruncated  bool
	UpstreamIncomplete bool
}

// ImplementationRef describes an interface implementation discovered via LSP.
type ImplementationRef struct {
	File         string
	ResolvedPath string
	Line         int
	Name         string
}

// Reference はシンボル参照。
type Reference struct {
	File         string
	ResolvedPath string
	Line         int
	Scope        string // 包含関数名
	Snippet      string // マッチ行テキスト
	IsTest       bool
	Class        ast.MatchClass // AST 分類（ClassCall, ClassRef, ClassDef 等）
	NodeType     string         // マッチした識別子ノード型（identifier / field_identifier など）
	SelectorKind string         // selector の種別（package / method / unknown）
	ReceiverType string         // method selector の推定レシーバ型
}

// TestRef は関連テストの参照情報。
type TestRef struct {
	File         string
	ResolvedPath string
	Name         string
	Line         int
}

// InspectSymbol は指定シンボルの定義・caller・ref・テストをまとめて返す。
func InspectSymbol(symbol, pathHint, mode string) string {
	if symbol == "" {
		return "Error: symbol is required"
	}

	query := parseSymbolQuery(symbol)

	inspectMode := ModeSummary
	if mode == "full" {
		inspectMode = ModeFull
	}

	budget := SummaryBudget
	if inspectMode == ModeFull {
		budget = FullBudget
	}

	// 1. シンボル候補を解決
	candidates := resolveSymbolCandidates(symbol, pathHint)
	if len(candidates) == 0 {
		return fmt.Sprintf("No symbol found: %q", symbol)
	}

	// 2. 複数候補 → 一覧のみ
	if len(candidates) > 1 {
		return formatMultipleCandidates(symbol, candidates, nil)
	}

	// 3. 単一候補 → 詳細取得
	cand := candidates[0]
	result := buildInspectResultForSingleCandidate(query, cand, budget, GoSymbolRuntime{}, nil, false)

	return formatInspectResult(result, nil)
}

func buildInspectResultForSingleCandidate(query symbolQuery, cand SymbolCandidate, budget Budget, runtime GoSymbolRuntime, lspClient LSPClient, normalizePaths bool) InspectResult {
	result := InspectResult{Symbol: &cand}
	result.Body = readDefinitionBody(cand, budget.BodyLines)

	allRefs, implementations, resolvedViaLSP, upstreamTruncated, upstreamIncomplete := collectInspectReferences(query.BaseName, cand, runtime, lspClient)
	result.Implementations = implementations
	result.ResolvedViaLSP = resolvedViaLSP
	result.UpstreamTruncated = upstreamTruncated
	result.UpstreamIncomplete = upstreamIncomplete
	result.Callers, result.TotalCallers, result.MoreCallers = classifyCallers(allRefs, cand, budget.CallerLimit)
	result.Refs, result.TotalRefs, result.MoreRefs = classifyRefs(allRefs, cand, budget.RefLimit)
	result.Tests, result.TotalTests, result.MoreTests = findRelatedTests(query.BaseName, allRefs, budget.TestLimit)

	if normalizePaths {
		normalizeInspectResultPaths(&result, runtime)
	}
	return result
}

func collectInspectReferences(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime, lspClient LSPClient) ([]Reference, []ImplementationRef, bool, bool, bool) {
	var implementations []ImplementationRef
	if lspClient != nil {
		if cand.Kind == "interface" {
			if impls, implErr := findImplementationsViaLSP(lspClient, cand, runtime.InvocationCWD); implErr == nil {
				implementations = impls
			}
		}
		lspRefs, err := findReferencesViaLSP(lspClient, cand, runtime.InvocationCWD)
		if err == nil && len(lspRefs) > 0 {
			return lspRefs, implementations, true, false, false
		}
	}

	refs, truncated, incomplete := findReferencesWithFallbackRuntime(baseName, cand, runtime)
	return refs, implementations, false, truncated, incomplete
}
