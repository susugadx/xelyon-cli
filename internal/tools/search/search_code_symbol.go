package search

import "github.com/susugadx/xelyon-cli/internal/navigation"

type symbolResolveStatus string

const (
	symbolResolveSingle   symbolResolveStatus = "single"
	symbolResolveMultiple symbolResolveStatus = "multiple"
	symbolResolveNone     symbolResolveStatus = "none"
)

type symbolResolveResult struct {
	Output        string
	Status        symbolResolveStatus
	Bundle        *SymbolBundle
	AffectedFiles []string
}

type symbolResolver interface {
	Resolve(symbol string, opts SearchOptions) symbolResolveResult
}

type goSymbolResolver struct{}

func (goSymbolResolver) Resolve(symbol string, opts SearchOptions) symbolResolveResult {
	result, output, status := navigation.ResolveInspectSymbolAuto(symbol, opts.Path, navigation.InspectSymbolAutoOptions{
		Budget:             searchCodeGoSymbolBudget,
		Registry:           nil,
		LSPClient:          opts.LSPClient,
		ProjectMap:         opts.ProjectMap,
		ProjectMapRootPath: opts.ProjectMapRootPath,
		ProjectMapStateKey: opts.ProjectMapStateKey,
		InvocationCWD:      opts.InvocationCWD,
	})
	switch status {
	case navigation.SymbolAutoSingle:
		bundle := buildGoSymbolBundle(symbol, result)
		if bundle == nil {
			return symbolResolveResult{Status: symbolResolveNone}
		}
		return symbolResolveResult{
			Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
			Status: symbolResolveSingle,
			Bundle: bundle,
		}
	case navigation.SymbolAutoMultiple:
		affectedFiles := collectNavigationCandidatesAffectedFiles(result.Candidates, opts)
		if opts.LocatorRegistry != nil {
			_, output, _ = navigation.ResolveInspectSymbolAuto(symbol, opts.Path, navigation.InspectSymbolAutoOptions{
				Budget:             searchCodeGoSymbolBudget,
				Registry:           opts.LocatorRegistry,
				LSPClient:          opts.LSPClient,
				ProjectMap:         opts.ProjectMap,
				ProjectMapRootPath: opts.ProjectMapRootPath,
				ProjectMapStateKey: opts.ProjectMapStateKey,
				InvocationCWD:      opts.InvocationCWD,
			})
		}
		return symbolResolveResult{Output: output, Status: symbolResolveMultiple, AffectedFiles: affectedFiles}
	default:
		return symbolResolveResult{Status: symbolResolveNone}
	}
}

type genericLanguageResolver struct {
	lang string
}

func (r genericLanguageResolver) Resolve(symbol string, opts SearchOptions) symbolResolveResult {
	var result genericResolveResult
	switch r.lang {
	case "js":
		result = resolveJSSymbol(symbol, opts)
	case "python":
		result = resolvePythonSymbol(symbol, opts)
	case "rust":
		result = resolveRustSymbol(symbol, opts)
	case "java":
		result = resolveJavaSymbol(symbol, opts)
	case "csharp":
		result = resolveCSharpSymbol(symbol, opts)
	case "php":
		result = resolvePHPSymbol(symbol, opts)
	case "ruby":
		result = resolveRubySymbol(symbol, opts)
	case "swift":
		result = resolveSwiftSymbol(symbol, opts)
	case "scala":
		result = resolveScalaSymbol(symbol, opts)
	case "elixir":
		result = resolveElixirSymbol(symbol, opts)
	case "lua":
		result = resolveLuaSymbol(symbol, opts)
	case "cpp":
		result = resolveCppSymbol(symbol, opts)
	default:
		result = resolveGenericSymbol(symbol, opts)
	}

	switch result.Status {
	case genericSymbolSingle:
		return symbolResolveResult{Output: result.Output, Status: symbolResolveSingle, Bundle: result.Bundle}
	case genericSymbolMultiple:
		return symbolResolveResult{Output: result.Output, Status: symbolResolveMultiple, AffectedFiles: result.AffectedFiles}
	default:
		return symbolResolveResult{Status: symbolResolveNone}
	}
}

func resolverForLanguage(lang string) symbolResolver {
	switch lang {
	case "go":
		return goSymbolResolver{}
	case "js", "python", "rust", "java", "csharp", "php", "ruby", "swift", "scala", "elixir", "lua", "cpp":
		return genericLanguageResolver{lang: lang}
	case "":
		return genericLanguageResolver{lang: ""}
	default:
		return nil
	}
}
