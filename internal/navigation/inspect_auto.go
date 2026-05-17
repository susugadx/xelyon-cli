package navigation

import (
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

// SymbolAutoStatus はシンボル自動解決の結果種別。
type SymbolAutoStatus string

const (
	SymbolAutoSingle   SymbolAutoStatus = "single"
	SymbolAutoMultiple SymbolAutoStatus = "multiple"
	SymbolAutoNone     SymbolAutoStatus = "none"
)

// InspectSymbolAutoOptions は search_code 向けの内部構造化オプション。
type InspectSymbolAutoOptions struct {
	Budget             Budget
	Registry           *locator.Registry
	LSPClient          LSPClient
	ProjectMap         *repomap.ProjectMap
	ProjectMapRootPath string
	ProjectMapStateKey string
	InvocationCWD      string
	ReferenceFilter    ReferenceFilter
	// FallbackReferenceSearchPath は LSP が参照を返さない場合の rg 探索開始位置。
	// 空文字は従来どおり "." を使う。
	FallbackReferenceSearchPath string
}

// ResolveInspectSymbolAuto はシンボル自動解決の構造化結果を返す。
// public tool ではなく、search_code の symbol bundle 構築用に使う。
func ResolveInspectSymbolAuto(symbol, pathHint string, opts InspectSymbolAutoOptions) (InspectResult, string, SymbolAutoStatus) {
	if symbol == "" {
		return InspectResult{}, "", SymbolAutoNone
	}

	runtime := GoSymbolRuntime{
		ProjectMap:         opts.ProjectMap,
		ProjectMapRootPath: opts.ProjectMapRootPath,
		ProjectMapStateKey: opts.ProjectMapStateKey,
		InvocationCWD:      opts.InvocationCWD,
	}

	budget := opts.Budget
	if isZeroInspectBudget(budget) {
		budget = SummaryBudget
	}
	return resolveInspectSymbol(symbol, pathHint, inspectResolveOptions{
		budget:                      budget,
		runtime:                     runtime,
		registry:                    opts.Registry,
		lspClient:                   opts.LSPClient,
		referenceFilter:             opts.ReferenceFilter,
		fallbackReferenceSearchPath: opts.FallbackReferenceSearchPath,
		normalizePaths:              true,
	})
}

// InspectSymbolAuto はシンボル名の自動解決を試みる。
// single: 単一候補が見つかった → summary 形式の結果を返す
// multiple: 複数候補 → 候補一覧を返す
// none: 見つからない → 空文字を返す（呼び出し側で text search にフォールバック）
// reg が nil でない場合、出力に Locator ID を付与する。
func InspectSymbolAuto(symbol, pathHint string, reg *locator.Registry, lspClient LSPClient) (output string, status SymbolAutoStatus) {
	_, output, status = ResolveInspectSymbolAuto(symbol, pathHint, InspectSymbolAutoOptions{
		Budget:    SummaryBudget,
		Registry:  reg,
		LSPClient: lspClient,
	})
	return output, status
}
