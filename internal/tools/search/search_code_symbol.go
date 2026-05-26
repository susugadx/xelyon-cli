package search

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

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
	Observation   *tools.RuntimeObservation
}

type symbolResolver interface {
	Resolve(symbol string, opts SearchOptions) symbolResolveResult
}

type goSymbolResolver struct{}

func (goSymbolResolver) Resolve(symbol string, opts SearchOptions) symbolResolveResult {
	scope := goSymbolSearchScopeForOptions(opts)
	autoOpts := goSymbolInspectAutoOptions(opts, nil, scope)
	result, output, status := navigation.ResolveInspectSymbolAuto(symbol, scope.DefinitionPathHint, autoOpts)
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
		observation := observationForNavigationCandidates(result.Candidates, opts)
		if opts.LocatorRegistry != nil {
			_, output, _ = navigation.ResolveInspectSymbolAuto(symbol, scope.DefinitionPathHint, goSymbolInspectAutoOptions(opts, opts.LocatorRegistry, scope))
		}
		return symbolResolveResult{Output: output, Status: symbolResolveMultiple, AffectedFiles: affectedFiles, Observation: observation}
	default:
		return symbolResolveResult{Status: symbolResolveNone}
	}
}

type goSymbolSearchScope struct {
	DefinitionPathHint          string
	FallbackReferenceSearchPath string
	ReferenceFilter             navigation.ReferenceFilter
}

func goSymbolSearchScopeForOptions(opts SearchOptions) goSymbolSearchScope {
	definitionPathHint := goSymbolDefinitionPathHint(opts)
	scope := goSymbolSearchScope{
		DefinitionPathHint:          definitionPathHint,
		FallbackReferenceSearchPath: definitionPathHint,
	}
	if packageDir, ok := goSymbolDirectFilePackageDir(definitionPathHint); ok {
		scope.FallbackReferenceSearchPath = packageDir
		scope.ReferenceFilter = goSymbolPackageDirReferenceFilter(packageDir)
	} else if directory, ok := goSymbolDirectoryScopePath(definitionPathHint); ok {
		scope.ReferenceFilter = goSymbolDirectoryReferenceFilter(directory)
	}
	return scope
}

func goSymbolInspectAutoOptions(opts SearchOptions, registry *locator.Registry, scope goSymbolSearchScope) navigation.InspectSymbolAutoOptions {
	return navigation.InspectSymbolAutoOptions{
		Budget:                      searchCodeGoSymbolBudget,
		Registry:                    registry,
		LSPClient:                   opts.LSPClient,
		ProjectMap:                  opts.ProjectMap,
		ProjectMapRootPath:          opts.ProjectMapRootPath,
		ProjectMapStateKey:          opts.ProjectMapStateKey,
		InvocationCWD:               opts.InvocationCWD,
		ReferenceFilter:             scope.ReferenceFilter,
		FallbackReferenceSearchPath: scope.FallbackReferenceSearchPath,
	}
}

func goSymbolDefinitionPathHint(opts SearchOptions) string {
	if strings.TrimSpace(opts.Path) == "" {
		if root := strings.TrimSpace(opts.ProjectMapRootPath); root != "" {
			return root
		}
	}
	if target := searchTargetPathForOptions(opts); target != "" {
		return target
	}
	return strings.TrimSpace(opts.Path)
}

func goSymbolDirectFilePackageDir(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	path = filepath.Clean(path)
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() || !strings.EqualFold(filepath.Ext(path), ".go") {
			return "", false
		}
		return filepath.Dir(path), true
	}
	if !strings.EqualFold(filepath.Ext(path), ".go") {
		return "", false
	}
	return filepath.Dir(path), true
}

func goSymbolPackageDirReferenceFilter(packageDir string) navigation.ReferenceFilter {
	packageDir = filepath.Clean(strings.TrimSpace(packageDir))
	return func(ref navigation.Reference) bool {
		refPath := goSymbolReferencePath(ref)
		if refPath == "" {
			return false
		}
		return filepath.Clean(filepath.Dir(refPath)) == packageDir
	}
}

func goSymbolDirectoryScopePath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	path = filepath.Clean(path)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return "", false
	}
	if strings.EqualFold(filepath.Ext(path), ".go") {
		return "", false
	}
	return path, true
}

func goSymbolDirectoryReferenceFilter(directory string) navigation.ReferenceFilter {
	directory = filepath.Clean(strings.TrimSpace(directory))
	return func(ref navigation.Reference) bool {
		refPath := goSymbolReferencePath(ref)
		if refPath == "" {
			return false
		}
		return goSymbolPathWithinDirectory(directory, refPath)
	}
}

func goSymbolReferencePath(ref navigation.Reference) string {
	refPath := strings.TrimSpace(ref.ResolvedPath)
	if refPath == "" {
		refPath = strings.TrimSpace(ref.File)
	}
	if refPath == "" {
		return ""
	}
	if absPath, err := filepath.Abs(refPath); err == nil {
		refPath = absPath
	}
	return filepath.Clean(refPath)
}

func goSymbolPathWithinDirectory(directory, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(directory), filepath.Clean(path))
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

type genericLanguageResolver struct {
	lang string
}

func (r genericLanguageResolver) Resolve(symbol string, opts SearchOptions) symbolResolveResult {
	var result genericResolveResult
	switch r.lang {
	case "js":
		result = resolveJSFamilySymbol(symbol, opts)
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
		return symbolResolveResult{Output: result.Output, Status: symbolResolveSingle, Bundle: result.Bundle, Observation: result.Observation}
	case genericSymbolMultiple:
		return symbolResolveResult{Output: result.Output, Status: symbolResolveMultiple, AffectedFiles: result.AffectedFiles, Observation: result.Observation}
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
