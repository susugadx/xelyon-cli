package navigation

import (
	"os"
	"path/filepath"
	"strings"

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
}

// ResolveInspectSymbolAuto はシンボル自動解決の構造化結果を返す。
// public tool ではなく、search_code の symbol bundle 構築用に使う。
func ResolveInspectSymbolAuto(symbol, pathHint string, opts InspectSymbolAutoOptions) (InspectResult, string, SymbolAutoStatus) {
	if symbol == "" {
		return InspectResult{}, "", SymbolAutoNone
	}

	query := parseSymbolQuery(symbol)
	runtime := GoSymbolRuntime{
		ProjectMap:         opts.ProjectMap,
		ProjectMapRootPath: opts.ProjectMapRootPath,
		ProjectMapStateKey: opts.ProjectMapStateKey,
		InvocationCWD:      opts.InvocationCWD,
	}
	candidates := resolveSymbolCandidatesWithRuntime(symbol, pathHint, runtime)

	if len(candidates) == 0 {
		return InspectResult{}, "", SymbolAutoNone
	}

	if len(candidates) > 1 {
		return InspectResult{Candidates: candidates}, formatMultipleCandidates(symbol, candidates, opts.Registry), SymbolAutoMultiple
	}

	budget := opts.Budget
	if budget.BodyLines == 0 && budget.CallerLimit == 0 && budget.RefLimit == 0 && budget.TestLimit == 0 {
		budget = SummaryBudget
	}

	cand := candidates[0]
	result := InspectResult{Symbol: &cand}
	result.Body = readDefinitionBody(cand, budget.BodyLines)

	var allRefs []Reference
	if opts.LSPClient != nil {
		lspRefs, err := findReferencesViaLSP(opts.LSPClient, cand, runtime.InvocationCWD)
		if err == nil && len(lspRefs) > 0 {
			allRefs = lspRefs
			result.ResolvedViaLSP = true
		} else {
			allRefs, result.UpstreamTruncated, result.UpstreamIncomplete = findReferencesWithFallbackRuntime(query.BaseName, cand, runtime)
		}
		if cand.Kind == "interface" {
			if impls, err := findImplementationsViaLSP(opts.LSPClient, cand, runtime.InvocationCWD); err == nil {
				result.Implementations = impls
			}
		}
	} else {
		allRefs, result.UpstreamTruncated, result.UpstreamIncomplete = findReferencesWithFallbackRuntime(query.BaseName, cand, runtime)
	}

	result.Callers, result.TotalCallers, result.MoreCallers = classifyCallers(allRefs, cand, budget.CallerLimit)
	result.Refs, result.TotalRefs, result.MoreRefs = classifyRefs(allRefs, cand, budget.RefLimit)
	result.Tests, result.TotalTests, result.MoreTests = findRelatedTests(query.BaseName, allRefs, budget.TestLimit)
	normalizeInspectResultPaths(&result, runtime)

	return result, formatInspectResult(result, opts.Registry), SymbolAutoSingle
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

func normalizeInspectResultPaths(result *InspectResult, runtime GoSymbolRuntime) {
	if result == nil || result.Symbol == nil {
		return
	}

	targetRoot := preferredInspectRootPath(result.Symbol.RootPath, runtime.ProjectMapRootPath)
	if targetRoot == "" {
		return
	}
	result.Symbol.RootPath = targetRoot

	sourceBase := strings.TrimSpace(runtime.InvocationCWD)
	if sourceBase == "" {
		if cwd, err := os.Getwd(); err == nil {
			sourceBase = cwd
		}
	}
	if sourceBase != "" {
		if abs, err := filepath.Abs(sourceBase); err == nil {
			sourceBase = abs
		}
	}

	for i := range result.Callers {
		result.Callers[i].File = normalizeResultFilePath(result.Callers[i].File, targetRoot, sourceBase)
	}
	for i := range result.Refs {
		result.Refs[i].File = normalizeResultFilePath(result.Refs[i].File, targetRoot, sourceBase)
	}
	for i := range result.Tests {
		result.Tests[i].File = normalizeResultFilePath(result.Tests[i].File, targetRoot, sourceBase)
	}
	for i := range result.Implementations {
		result.Implementations[i].File = normalizeResultFilePath(result.Implementations[i].File, targetRoot, sourceBase)
	}
}

func preferredInspectRootPath(symbolRoot, projectRoot string) string {
	symbolRoot = normalizeInspectRootPath(symbolRoot)
	projectRoot = normalizeInspectRootPath(projectRoot)

	switch {
	case symbolRoot == "":
		return projectRoot
	case projectRoot == "":
		return symbolRoot
	case pathWithinRoot(symbolRoot, projectRoot):
		return projectRoot
	case pathWithinRoot(projectRoot, symbolRoot):
		return symbolRoot
	default:
		return projectRoot
	}
}

func normalizeInspectRootPath(rootPath string) string {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return ""
	}
	if abs, err := filepath.Abs(rootPath); err == nil {
		return abs
	}
	return filepath.Clean(rootPath)
}

func pathWithinRoot(rootPath, candidatePath string) bool {
	rel, err := filepath.Rel(rootPath, candidatePath)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, "../"))
}

func normalizeResultFilePath(path, targetRoot, sourceBase string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if filepath.IsAbs(path) {
		if rel, ok := absoluteToSnapshotRel(targetRoot, path); ok {
			return filepath.Clean(filepath.ToSlash(rel))
		}
		return filepath.Clean(path)
	}

	if targetRoot != "" {
		rootRelativeAbs := filepath.Join(targetRoot, filepath.FromSlash(path))
		if rel, ok := absoluteToSnapshotRel(targetRoot, rootRelativeAbs); ok && pathExists(rootRelativeAbs) {
			return filepath.Clean(filepath.ToSlash(rel))
		}
	}

	if sourceBase != "" {
		if rel, ok := absoluteToSnapshotRel(targetRoot, filepath.Join(sourceBase, filepath.FromSlash(path))); ok {
			return filepath.Clean(filepath.ToSlash(rel))
		}
	}

	return filepath.Clean(filepath.ToSlash(path))
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
