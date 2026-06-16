package search

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

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
