package search

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/searchcache"
)

type methodProbeReceiverRole string

const (
	methodProbeReceiverRoleUnknown   methodProbeReceiverRole = "unknown"
	methodProbeReceiverRoleConcrete  methodProbeReceiverRole = "concrete"
	methodProbeReceiverRoleInterface methodProbeReceiverRole = "interface"
)

type qualifiedReceiverProbeContext struct {
	receiver    string
	currentPath string
	currentSrc  []byte
	opts        SearchOptions
	deps        *structuredGoImpactProbeDeps
}

type qualifiedReceiverLocalHint struct {
	baseName string
	pathHint string
	repoRoot string
}

var methodProbeReceiverRoleCache sync.Map
var methodProbeQualifiedReceiverDirectMethodCache sync.Map

func init() {
	searchcache.AddSearchCacheLifecycleHooks(clearStructuredGoImpactMethodProbeCaches, clearStructuredGoImpactMethodProbeCachesWithKeys, clearStructuredGoImpactMethodProbeCachesWithKeys)
}

func clearStructuredGoImpactMethodProbeCaches() {
	methodProbeReceiverRoleCache.Range(func(key, value any) bool {
		methodProbeReceiverRoleCache.Delete(key)
		return true
	})
	methodProbeQualifiedReceiverDirectMethodCache.Range(func(key, value any) bool {
		methodProbeQualifiedReceiverDirectMethodCache.Delete(key)
		return true
	})
}

func clearStructuredGoImpactMethodProbeCachesWithKeys(_ []string) {
	clearStructuredGoImpactMethodProbeCaches()
}

func structuredGoImpactProbeReceiver(symbol navigation.SymbolCandidate) string {
	if receiver := strings.TrimSpace(symbol.ReceiverNorm); receiver != "" {
		return receiver
	}
	return canonicalProbeReceiver(symbol.Receiver)
}

func canonicalProbeReceiver(receiver string) string {
	receiver, _ = splitProbeReceiverQualifier(receiver)
	return receiver
}

func methodProbeQualifiedReceiverRole(receiver, currentPath string, currentSrc []byte, opts SearchOptions) methodProbeReceiverRole {
	return newQualifiedReceiverProbeContext(receiver, currentPath, currentSrc, opts).role()
}

func methodProbeQualifiedReceiverHasDirectMethod(receiver, methodName, currentPath string, currentSrc []byte, opts SearchOptions) bool {
	return newQualifiedReceiverProbeContext(receiver, currentPath, currentSrc, opts).hasDirectMethod(methodName)
}

func newQualifiedReceiverProbeContext(receiver, currentPath string, currentSrc []byte, opts SearchOptions) qualifiedReceiverProbeContext {
	return newQualifiedReceiverProbeContextWithDeps(receiver, currentPath, currentSrc, opts, nil)
}

func newQualifiedReceiverProbeContextWithDeps(receiver, currentPath string, currentSrc []byte, opts SearchOptions, deps *structuredGoImpactProbeDeps) qualifiedReceiverProbeContext {
	return qualifiedReceiverProbeContext{
		receiver:    strings.TrimSpace(receiver),
		currentPath: strings.TrimSpace(currentPath),
		currentSrc:  currentSrc,
		opts:        opts,
		deps:        deps,
	}
}

func (ctx qualifiedReceiverProbeContext) role() methodProbeReceiverRole {
	if ctx.receiver == "" || !strings.Contains(ctx.receiver, ".") {
		return methodProbeReceiverRoleUnknown
	}

	if value, ok := methodProbeReceiverRoleCache.Load(ctx.cacheKey()); ok {
		if role, ok := value.(methodProbeReceiverRole); ok {
			return role
		}
	}

	role := methodProbeReceiverRoleUnknown
	hint := ctx.resolveLocalHint()
	if hint.baseName != "" && hint.pathHint != "" {
		ctx.deps.addDirGoFiles(hint.pathHint)
		if fastRole, complete := qualifiedReceiverRoleFromLocalDir(hint.baseName, hint.pathHint); fastRole != methodProbeReceiverRoleUnknown || complete {
			role = fastRole
		} else {
			autoOpts := qualifiedReceiverInspectAutoOptions(ctx.opts, hint.repoRoot, hint.pathHint)
			result, _, status := navigation.ResolveInspectSymbolAuto(hint.baseName, hint.pathHint, autoOpts)
			switch status {
			case navigation.SymbolAutoSingle:
				if result.Symbol != nil {
					role = methodProbeReceiverRoleForKind(result.Symbol.Kind)
				}
			case navigation.SymbolAutoMultiple:
				role = methodProbeReceiverRoleFromCandidates(result.Candidates)
			}
		}
	}

	methodProbeReceiverRoleCache.Store(ctx.cacheKey(), role)
	return role
}

func (ctx qualifiedReceiverProbeContext) hasDirectMethod(methodName string) bool {
	methodName = strings.TrimSpace(methodName)
	if ctx.receiver == "" || methodName == "" {
		return false
	}

	cacheKey := ctx.cacheKey(methodName)
	if value, ok := methodProbeQualifiedReceiverDirectMethodCache.Load(cacheKey); ok {
		if direct, ok := value.(bool); ok {
			return direct
		}
	}

	direct := false
	hint := ctx.resolveLocalHint()
	if hint.baseName != "" && hint.pathHint != "" {
		ctx.deps.addDirGoFiles(hint.pathHint)
		if fastDirect, complete := qualifiedReceiverDirectMethodFromLocalDir(hint.baseName, methodName, hint.pathHint); fastDirect || complete {
			direct = fastDirect
		} else {
			autoOpts := qualifiedReceiverInspectAutoOptions(ctx.opts, hint.repoRoot, hint.pathHint)
			result, _, status := navigation.ResolveInspectSymbolAuto(hint.baseName+"."+methodName, hint.pathHint, autoOpts)
			switch status {
			case navigation.SymbolAutoSingle:
				if result.Symbol != nil && result.Symbol.Kind == "method" {
					direct = canonicalProbeReceiver(result.Symbol.ReceiverNorm) == hint.baseName
				}
			case navigation.SymbolAutoMultiple:
				for _, candidate := range result.Candidates {
					if strings.TrimSpace(candidate.Kind) != "method" {
						continue
					}
					if canonicalProbeReceiver(candidate.ReceiverNorm) == hint.baseName {
						direct = true
						break
					}
				}
			}
		}
	}

	methodProbeQualifiedReceiverDirectMethodCache.Store(cacheKey, direct)
	return direct
}

func (ctx qualifiedReceiverProbeContext) cacheKey(parts ...string) string {
	keyParts := []string{
		ctx.receiver,
		ctx.currentPath,
		strings.TrimSpace(ctx.opts.Path),
		strings.TrimSpace(ctx.opts.ProjectMapRootPath),
		strings.TrimSpace(ctx.opts.ProjectMapStateKey),
		strings.TrimSpace(ctx.opts.InvocationCWD),
	}
	keyParts = append(keyParts, parts...)
	return strings.Join(keyParts, "|")
}

func (ctx qualifiedReceiverProbeContext) resolveLocalHint() qualifiedReceiverLocalHint {
	qualifier, baseName, qualified := splitProbeReceiverImportQualifier(ctx.receiver)
	if !qualified || qualifier == "" || baseName == "" {
		return qualifiedReceiverLocalHint{}
	}

	importPath := resolveQualifiedReceiverImportPath(qualifier, ctx.currentPath, ctx.currentSrc, ctx.opts)
	if importPath == "" {
		return qualifiedReceiverLocalHint{}
	}

	basePath := structuredGoImpactImportResolveBasePath(ctx.opts, ctx.currentPath)
	if basePath == "" {
		return qualifiedReceiverLocalHint{}
	}

	localDir, repoRoot := resolveLocalImportPathHint(basePath, importPath)
	if localDir == "" {
		return qualifiedReceiverLocalHint{}
	}
	return qualifiedReceiverLocalHint{
		baseName: baseName,
		pathHint: localDir,
		repoRoot: repoRoot,
	}
}

func qualifiedReceiverInspectAutoOptions(opts SearchOptions, repoRoot, fallbackSearchPath string) navigation.InspectSymbolAutoOptions {
	autoOpts := navigation.InspectSymbolAutoOptions{
		Budget:                      navigation.SummaryBudget,
		ProjectMapStateKey:          opts.ProjectMapStateKey,
		InvocationCWD:               opts.InvocationCWD,
		FallbackReferenceSearchPath: strings.TrimSpace(fallbackSearchPath),
	}
	if repoRoot == "" {
		return autoOpts
	}
	autoOpts.ProjectMapRootPath = repoRoot
	return autoOpts
}

func methodProbeReceiverRoleForKind(kind string) methodProbeReceiverRole {
	switch strings.TrimSpace(kind) {
	case "interface":
		return methodProbeReceiverRoleInterface
	case "":
		return methodProbeReceiverRoleUnknown
	default:
		return methodProbeReceiverRoleConcrete
	}
}

func methodProbeReceiverRoleFromCandidates(candidates []navigation.SymbolCandidate) methodProbeReceiverRole {
	role := methodProbeReceiverRoleUnknown
	for _, candidate := range candidates {
		switch strings.TrimSpace(candidate.Kind) {
		case "interface":
			return methodProbeReceiverRoleInterface
		case "":
			continue
		default:
			role = methodProbeReceiverRoleConcrete
		}
	}
	return role
}

func qualifiedReceiverRoleFromLocalDir(baseName, dir string) (methodProbeReceiverRole, bool) {
	files, complete := parseGoPackageFilesInDir(dir)
	if len(files) == 0 {
		return methodProbeReceiverRoleUnknown, complete
	}

	role := methodProbeReceiverRoleUnknown
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name == nil || typeSpec.Name.Name != baseName {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
					return methodProbeReceiverRoleInterface, complete
				}
				role = methodProbeReceiverRoleConcrete
			}
		}
	}
	return role, complete
}

func qualifiedReceiverDirectMethodFromLocalDir(baseName, methodName, dir string) (bool, bool) {
	files, complete := parseGoPackageFilesInDir(dir)
	if len(files) == 0 {
		return false, complete
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != methodName || fn.Recv == nil {
				continue
			}
			for _, recv := range fn.Recv.List {
				if receiverTypeName(recv.Type) == baseName {
					return true, complete
				}
			}
		}
	}
	return false, complete
}

func parseGoPackageFilesInDir(dir string) ([]*ast.File, bool) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, false
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}

	complete := true
	files := make([]*ast.File, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil || file == nil {
			complete = false
			continue
		}
		files = append(files, file)
	}
	return files, complete
}

func receiverTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return receiverTypeName(e.X)
	case *ast.IndexExpr:
		return receiverTypeName(e.X)
	case *ast.IndexListExpr:
		return receiverTypeName(e.X)
	case *ast.SelectorExpr:
		if e.Sel != nil {
			return e.Sel.Name
		}
	}
	return ""
}
