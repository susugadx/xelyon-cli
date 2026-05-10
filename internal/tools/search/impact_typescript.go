package search

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

const (
	structuredTypeScriptImpactRouteTag            = "impact-structured-typescript-v1"
	typeScriptImpactRecommendedReadLimit          = 5
	typeScriptImpactHighNonTestReferenceThreshold = 8
	typeScriptImpactMediumReferenceThreshold      = 4
)

func tryStructuredTypeScriptImpactSearch(cache tools.ToolCacheInterface, opts SearchOptions) (string, bool) {
	result, ok := tryExpandedStructuredTypeScriptImpactSearchResult(cache, opts)
	if !ok {
		return "", false
	}
	return result.Rendered, true
}

func tryExpandedStructuredTypeScriptImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	result, ok := tryStructuredTypeScriptImpactSearchResult(cache, opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return expandStructuredImpactSearchResult(cache, opts, result), true
}

func tryStructuredTypeScriptImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	ctx, resolverOpts, ok := newStructuredTypeScriptImpactSearchContext(opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return tryStructuredImpactSearchResult(cache, ctx, resolverOpts, resolveStructuredTypeScriptImpactSymbol)
}

func tryStructuredTypeScriptImpactSearchArtifact(cache tools.ToolCacheInterface, opts SearchOptions) (SearchExecutionArtifact, bool) {
	result, ok := tryExpandedStructuredTypeScriptImpactSearchResult(cache, opts)
	if !ok {
		return SearchExecutionArtifact{}, false
	}
	return newStructuredImpactSearchArtifact(result), true
}

func newStructuredTypeScriptImpactSearchContext(opts SearchOptions) (structuredImpactSearchContext, SearchOptions, bool) {
	pattern := strings.TrimSpace(opts.Pattern)
	resolverOpts, ok := normalizeStructuredTypeScriptImpactOptions(opts)
	if !shouldAttemptStructuredTypeScriptImpactSearch(opts, pattern) || !ok {
		return structuredImpactSearchContext{}, SearchOptions{}, false
	}

	route, ok := structuredTypeScriptImpactRoute(pattern, opts)
	if !ok {
		return structuredImpactSearchContext{}, SearchOptions{}, false
	}

	return structuredImpactSearchContext{
		Pattern:  pattern,
		Route:    route,
		CacheKey: buildStructuredImpactCacheKey(opts, route, structuredTypeScriptImpactRouteTag),
	}, resolverOpts, true
}

func shouldAttemptStructuredTypeScriptImpactSearch(opts SearchOptions, pattern string) bool {
	return shouldAttemptSinglePatternImpactSearch(opts, pattern)
}

func structuredTypeScriptImpactRoute(pattern string, opts SearchOptions) (searchRouteTrace, bool) {
	analysis := analyzeSearchQuery(pattern)
	if !analysis.LooksLikeBareIdentifier && !analysis.LooksLikeDottedSymbol {
		return searchRouteTrace{}, false
	}

	route := planSearchRoute(pattern, opts)
	if route.InitialLane == searchLaneSymbol {
		if route.Language == "" {
			route.Language = "js"
		}
		return route, true
	}

	mode := SearchMode(opts.Mode)
	if mode == SearchModeLiteral || mode == SearchModeRegex {
		return searchRouteTrace{}, false
	}

	return assignRouteSymbolQuery(searchRouteTrace{
		RequestedMode: mode,
		Language:      "js",
		FallbackLane:  analysis.defaultTextLane(),
		Decision:      structuredTypeScriptImpactRouteTag,
		Analysis:      analysis,
	}, analysis.TrimmedPattern, []string{analysis.TrimmedPattern}), true
}

func normalizeStructuredTypeScriptImpactOptions(opts SearchOptions) (SearchOptions, bool) {
	fileType := strings.ToLower(strings.TrimSpace(opts.FileType))
	filePattern := filepath.ToSlash(strings.TrimSpace(opts.FilePattern))

	switch fileType {
	case "ts":
		opts.FileType = "ts"
		opts.FilePattern = ""
		return opts, true
	case "typescript", "tsx", "js", "jsx", "mjs", "cjs", "javascript":
		return SearchOptions{}, false
	case "":
	default:
		return SearchOptions{}, false
	}

	switch {
	case filePattern == "*.ts":
		opts.FileType = ""
		opts.FilePattern = "*.ts"
		return opts, true
	case filePattern != "":
		return SearchOptions{}, false
	case isTypeScriptSourceFilePath(opts.Path):
		return opts, true
	default:
		return SearchOptions{}, false
	}
}

func isTypeScriptSourceFilePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(path)))
	return ext == ".ts"
}

func resolveStructuredTypeScriptImpactSymbol(symbol string, opts SearchOptions) symbolResolveResult {
	defs := normalizeStructuredTypeScriptDefs(findGenericDefinitions(symbol, opts))
	if len(defs) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	if len(defs) > 1 {
		return symbolResolveResult{
			Output:        formatGenericMultipleDefsWithOptions(symbol, defs, opts.LocatorRegistry, opts),
			Status:        symbolResolveMultiple,
			AffectedFiles: collectStructuredTypeScriptDefAffectedFiles(defs, opts),
		}
	}

	def := defs[0]
	refs := normalizeStructuredTypeScriptRefs(findGenericReferences(symbol, opts))
	filteredRefs := filterGenericRefs(refs, def)
	classifiedRefs := classifyJSFamilySymbolRefs(filteredRefs, symbol)
	bundle := buildTypeScriptImpactBundle(symbol, def, opts, classifiedRefs)
	if bundle == nil || bundle.Impact == nil || len(bundle.Impact.RecommendedReads) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}

	return symbolResolveResult{
		Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
		Status: symbolResolveSingle,
		Bundle: bundle,
	}
}

func normalizeStructuredTypeScriptDefs(defs []genericSymbolDef) []genericSymbolDef {
	normalized := make([]genericSymbolDef, len(defs))
	for i, def := range defs {
		normalized[i] = def
		normalized[i].File = cleanStructuredTypeScriptDisplayPath(def.File)
	}
	return normalized
}

func normalizeStructuredTypeScriptRefs(refs []genericSymbolRef) []genericSymbolRef {
	normalized := make([]genericSymbolRef, len(refs))
	for i, ref := range refs {
		normalized[i] = ref
		normalized[i].File = cleanStructuredTypeScriptDisplayPath(ref.File)
	}
	return normalized
}

func cleanStructuredTypeScriptDisplayPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." {
		return ""
	}
	return clean
}

func buildTypeScriptImpactBundle(symbol string, def genericSymbolDef, opts SearchOptions, refs jsFamilySymbolRefs) *SymbolBundle {
	rootPath := structuredTypeScriptImpactFileRoot(opts)
	impact := buildTypeScriptImpactMetadata(def, refs, rootPath)
	if impact == nil || len(impact.RecommendedReads) == 0 {
		return nil
	}

	displayName := def.Name
	if displayName == "" {
		displayName = symbol
	}
	bundle := &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "typescript",
			Query:       symbol,
			Canonical:   canonicalSymbolBundleKey("typescript", def.File, def.Line, displayName),
			DisplayName: displayName,
			Kind:        def.Kind,
			File:        def.File,
			Line:        def.Line,
			EndLine:     def.Line,
		},
		Definition: SymbolBundleDefinition{
			File:      def.File,
			Line:      def.Line,
			EndLine:   def.Line,
			Signature: def.Signature,
			Body:      []string{fmt.Sprintf("%d: %s", def.Line, def.Signature)},
		},
		Impact: impact,
		Debug: SymbolBundleDebug{
			Source:       "typescript-impact-structured",
			FileRootPath: rootPath,
		},
	}

	appendTypeScriptImpactSection(bundle, def, "imports", "Imports", refs.imports, jsImportLimit, false, rootPath, symbol)
	appendTypeScriptImpactSection(bundle, def, "callers", "Callers", refs.callers, jsCallerLimit, false, rootPath, symbol)
	appendTypeScriptImpactSection(bundle, def, "type_refs", "Type References", refs.typeRefs, jsTypeRefLimit, false, rootPath, symbol)
	appendTypeScriptImpactSection(bundle, def, "references", "References", refs.others, genericRefLimit, false, rootPath, symbol)
	appendTypeScriptImpactSection(bundle, def, "tests", "Related Tests", refs.tests, genericTestLimit, true, rootPath, symbol)

	return bundle
}

func appendTypeScriptImpactSection(bundle *SymbolBundle, def genericSymbolDef, kind, title string, refs []genericSymbolRef, limit int, isTest bool, rootPath string, symbol string) {
	items := typeScriptImpactItemsFromRefs(def, refs, kind, limit, isTest, rootPath, symbol)
	if len(items) == 0 {
		return
	}

	total := len(dedupeGenericRefs(refs))
	bundle.Sections = append(bundle.Sections, SymbolBundleSection{
		Kind:  kind,
		Title: title,
		Items: items,
		Total: total,
		More:  total > len(items),
	})
}

func buildTypeScriptImpactMetadata(def genericSymbolDef, refs jsFamilySymbolRefs, rootPath string) *SymbolBundleImpact {
	impact := &SymbolBundleImpact{
		RiskLevel:        classifyTypeScriptImpactRisk(def, refs),
		RecommendedReads: make([]SymbolBundleItem, 0, typeScriptImpactRecommendedReadLimit),
	}

	seen := make(map[string]struct{}, typeScriptImpactRecommendedReadLimit)
	add := func(item SymbolBundleItem) {
		if item.File == "" || item.Line <= 0 || len(impact.RecommendedReads) >= typeScriptImpactRecommendedReadLimit {
			return
		}
		key := fmt.Sprintf("%s:%d", item.File, item.Line)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		impact.RecommendedReads = append(impact.RecommendedReads, item)
	}

	add(typeScriptImpactDefinitionItem(def, rootPath))
	for _, group := range typeScriptImpactRecommendedReadGroups(def, refs) {
		for _, item := range typeScriptImpactItemsFromRefs(def, group.refs, group.kind, group.limit, group.isTest, rootPath, def.Name) {
			add(item)
		}
	}

	if len(impact.RecommendedReads) == 0 {
		return nil
	}
	return impact
}

type typeScriptImpactReadGroup struct {
	kind   string
	refs   []genericSymbolRef
	limit  int
	isTest bool
}

func typeScriptImpactRecommendedReadGroups(def genericSymbolDef, refs jsFamilySymbolRefs) []typeScriptImpactReadGroup {
	typeRefGroups := []typeScriptImpactReadGroup{
		{kind: "type_refs", refs: refs.typeRefs, limit: typeScriptImpactRecommendedReadLimit},
		{kind: "references", refs: refs.others, limit: typeScriptImpactRecommendedReadLimit},
		{kind: "imports", refs: refs.imports, limit: typeScriptImpactRecommendedReadLimit},
	}
	callerGroup := typeScriptImpactReadGroup{kind: "callers", refs: refs.callers, limit: typeScriptImpactRecommendedReadLimit}
	testGroup := typeScriptImpactReadGroup{kind: "tests", refs: refs.tests, limit: typeScriptImpactRecommendedReadLimit, isTest: true}

	if typeScriptImpactPrefersTypeRefs(def.Kind) {
		groups := append([]typeScriptImpactReadGroup(nil), typeRefGroups...)
		groups = append(groups, testGroup, callerGroup)
		return groups
	}

	groups := []typeScriptImpactReadGroup{callerGroup, testGroup}
	groups = append(groups, typeRefGroups...)
	return groups
}

func typeScriptImpactPrefersTypeRefs(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "interface", "type":
		return true
	default:
		return false
	}
}

func typeScriptImpactItemsFromRefs(def genericSymbolDef, refs []genericSymbolRef, kind string, limit int, isTest bool, rootPath string, symbol string) []SymbolBundleItem {
	selected := prioritizeGenericRefs(def, refs, limit, isTest)
	if len(selected) == 0 {
		return nil
	}

	items := make([]SymbolBundleItem, 0, len(selected))
	for _, ref := range selected {
		items = append(items, typeScriptImpactItemFromRef(kind, ref, rootPath, symbol, isTest))
	}
	return items
}

func typeScriptImpactDefinitionItem(def genericSymbolDef, rootPath string) SymbolBundleItem {
	return SymbolBundleItem{
		Kind:         "definition",
		File:         def.File,
		ResolvedPath: absoluteAffectedFilePathWithBase(def.File, rootPath),
		Line:         def.Line,
		EndLine:      def.Line,
		Snippet:      strings.TrimSpace(def.Signature),
		Name:         def.Name,
	}
}

func typeScriptImpactItemFromRef(kind string, ref genericSymbolRef, rootPath string, symbol string, forceTest bool) SymbolBundleItem {
	isTest := forceTest || ref.IsTest
	name := strings.TrimSpace(symbol)
	if isTest {
		name = ""
	}
	return SymbolBundleItem{
		Kind:         kind,
		File:         ref.File,
		ResolvedPath: absoluteAffectedFilePathWithBase(ref.File, rootPath),
		Line:         ref.Line,
		EndLine:      ref.Line,
		Snippet:      strings.TrimSpace(ref.Snippet),
		Name:         name,
		IsTest:       isTest,
	}
}

func classifyTypeScriptImpactRisk(def genericSymbolDef, refs jsFamilySymbolRefs) string {
	nonTestRefCount := len(dedupeGenericRefs(append(append(append(append([]genericSymbolRef(nil), refs.imports...), refs.callers...), refs.typeRefs...), refs.others...)))
	hasTests := len(dedupeGenericRefs(refs.tests)) > 0
	exported := typeScriptDefinitionIsExported(def)

	switch {
	case nonTestRefCount >= typeScriptImpactHighNonTestReferenceThreshold:
		return goImpactRiskHigh
	case exported && !hasTests:
		return goImpactRiskHigh
	case exported || !hasTests || nonTestRefCount >= typeScriptImpactMediumReferenceThreshold:
		return goImpactRiskMedium
	default:
		return goImpactRiskLow
	}
}

func typeScriptDefinitionIsExported(def genericSymbolDef) bool {
	return strings.HasPrefix(strings.TrimSpace(def.Signature), "export ")
}

func structuredTypeScriptImpactFileRoot(opts SearchOptions) string {
	basis := resolveSearchPathBasisForOptions(opts)
	if strings.TrimSpace(basis.MatchRoot) != "" {
		return basis.MatchRoot
	}
	if strings.TrimSpace(basis.Workdir) != "" {
		return basis.Workdir
	}
	return invocationCWDOrGetwd(opts)
}

func collectStructuredTypeScriptDefAffectedFiles(defs []genericSymbolDef, opts SearchOptions) []string {
	rootPath := structuredTypeScriptImpactFileRoot(opts)
	paths := make([]string, 0, len(defs))
	for _, def := range defs {
		if absPath := absoluteAffectedFilePathWithBase(def.File, rootPath); absPath != "" {
			paths = append(paths, absPath)
		}
	}
	return dedupePaths(paths)
}
