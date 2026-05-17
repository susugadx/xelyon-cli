package search

import (
	"path/filepath"
	"strings"
)

type javaScriptImpactRefs struct {
	imports          []genericSymbolRef
	callers          []genericSymbolRef
	others           []genericSymbolRef
	directTests      []genericSymbolRef
	nearbyTests      []genericSymbolRef
	hasTotalRefs     bool
	totalImports     []genericSymbolRef
	totalCallers     []genericSymbolRef
	totalOthers      []genericSymbolRef
	totalDirectTests []genericSymbolRef
}

func javaScriptImpactRefsForDef(def genericSymbolDef, refs []genericSymbolRef, opts SearchOptions) javaScriptImpactRefs {
	return javaScriptImpactRefsForDisplayAndTotalRefs(def, refs, refs, opts)
}

func javaScriptImpactRefsForDisplayAndTotalRefs(def genericSymbolDef, refs []genericSymbolRef, totalRefs []genericSymbolRef, opts SearchOptions) javaScriptImpactRefs {
	classified := classifyJSFamilySymbolRefsFromAST(refs)
	totalClassified := classifyJSFamilySymbolRefsFromAST(totalRefs)
	result := javaScriptImpactRefs{
		imports:          classified.imports,
		callers:          classified.callers,
		others:           classified.others,
		directTests:      classified.tests,
		totalImports:     totalClassified.imports,
		totalCallers:     totalClassified.callers,
		totalOthers:      totalClassified.others,
		totalDirectTests: totalClassified.tests,
		hasTotalRefs:     true,
	}
	result.nearbyTests = findNearbyJavaScriptTests(def, opts, result.directTests)
	return result
}

func (refs javaScriptImpactRefs) allTests() []genericSymbolRef {
	return allJSFamilyTests(refs.directTests, refs.nearbyTests)
}

func (refs javaScriptImpactRefs) allTotalTests() []genericSymbolRef {
	if !refs.hasTotalRefs {
		return refs.allTests()
	}
	return allJSFamilyTests(refs.totalDirectTests, refs.nearbyTests)
}

func (refs javaScriptImpactRefs) totalImportsForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalImports
	}
	return refs.imports
}

func (refs javaScriptImpactRefs) totalCallersForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalCallers
	}
	return refs.callers
}

func (refs javaScriptImpactRefs) totalOthersForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalOthers
	}
	return refs.others
}

func findNearbyJavaScriptTests(def genericSymbolDef, opts SearchOptions, directTests []genericSymbolRef) []genericSymbolRef {
	rootPath := structuredJavaScriptImpactFileRoot(opts)
	if rootPath == "" || !isJavaScriptSourceFilePath(def.File) {
		return nil
	}

	return findNearbyJSFamilyTests(rootPath, javaScriptNearbyTestCandidatePaths(def.File), directTests, func(absPath string, displayPath string) bool {
		return isUsableNearbyJavaScriptTest(absPath, displayPath, opts)
	})
}

func javaScriptNearbyTestCandidatePaths(defFile string) []string {
	cleanFile := filepath.ToSlash(filepath.Clean(defFile))
	dir := filepath.ToSlash(filepath.Dir(cleanFile))
	base := strings.TrimSuffix(filepath.Base(cleanFile), ".js")

	candidates := []string{
		filepath.ToSlash(filepath.Join(dir, base+".test.js")),
		filepath.ToSlash(filepath.Join(dir, base+".spec.js")),
		filepath.ToSlash(filepath.Join(dir, "__tests__", base+".test.js")),
		filepath.ToSlash(filepath.Join(dir, "__tests__", base+".spec.js")),
		filepath.ToSlash(filepath.Join("tests", base+".test.js")),
		filepath.ToSlash(filepath.Join("tests", base+".spec.js")),
	}
	return dedupeStringList(candidates)
}

func isUsableNearbyJavaScriptTest(absPath, displayPath string, opts SearchOptions) bool {
	return isUsableNearbyJSFamilyTest(absPath, displayPath, opts)
}
