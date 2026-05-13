package search

type typeScriptImpactRefs struct {
	imports     []genericSymbolRef
	callers     []genericSymbolRef
	typeRefs    []genericSymbolRef
	others      []genericSymbolRef
	directTests []genericSymbolRef
	nearbyTests []genericSymbolRef
}

func typeScriptImpactRefsForDef(def genericSymbolDef, refs []genericSymbolRef, opts SearchOptions) typeScriptImpactRefs {
	classified := classifyJSFamilySymbolRefsFromAST(refs)
	result := typeScriptImpactRefs{
		imports:     classified.imports,
		callers:     classified.callers,
		typeRefs:    classified.typeRefs,
		others:      classified.others,
		directTests: classified.tests,
	}
	result.nearbyTests = findNearbyTypeScriptTests(def, opts, result.directTests)
	return result
}

func (refs typeScriptImpactRefs) allTests() []genericSymbolRef {
	return allJSFamilyTests(refs.directTests, refs.nearbyTests)
}

func findNearbyTypeScriptTests(def genericSymbolDef, opts SearchOptions, directTests []genericSymbolRef) []genericSymbolRef {
	rootPath := structuredTypeScriptImpactFileRoot(opts)
	target, ok := structuredTypeScriptImplementationTargetForPath(def.File)
	if rootPath == "" || !ok {
		return nil
	}

	return findNearbyJSFamilyTests(rootPath, typeScriptNearbyTestCandidatePaths(def.File), directTests, func(absPath string, displayPath string) bool {
		return isUsableNearbyTypeScriptTest(absPath, displayPath, opts, target)
	})
}

func typeScriptNearbyTestCandidatePaths(defFile string) []string {
	target, ok := structuredTypeScriptImplementationTargetForPath(defFile)
	if !ok {
		return nil
	}
	return target.nearbyTestCandidatePaths(defFile)
}

func isUsableNearbyTypeScriptTest(absPath, displayPath string, opts SearchOptions, target structuredTypeScriptImpactTarget) bool {
	if !target.matchesNearbyTestPath(displayPath) {
		return false
	}
	return isUsableNearbyJSFamilyTest(absPath, displayPath, opts)
}
