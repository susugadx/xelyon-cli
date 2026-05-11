package search

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type typeScriptImpactRefs struct {
	imports     []genericSymbolRef
	callers     []genericSymbolRef
	typeRefs    []genericSymbolRef
	others      []genericSymbolRef
	directTests []genericSymbolRef
	nearbyTests []genericSymbolRef
}

func typeScriptImpactRefsForDef(def genericSymbolDef, refs []genericSymbolRef, opts SearchOptions, symbol string) typeScriptImpactRefs {
	classified := classifyJSFamilySymbolRefs(refs, symbol)
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
	tests := make([]genericSymbolRef, 0, len(refs.directTests)+len(refs.nearbyTests))
	tests = append(tests, refs.directTests...)
	tests = append(tests, refs.nearbyTests...)
	return tests
}

func findNearbyTypeScriptTests(def genericSymbolDef, opts SearchOptions, directTests []genericSymbolRef) []genericSymbolRef {
	rootPath := structuredTypeScriptImpactFileRoot(opts)
	if rootPath == "" || !isTypeScriptImplementationFilePath(def.File) {
		return nil
	}

	directTestFiles := make(map[string]struct{}, len(directTests))
	for _, test := range directTests {
		if test.File == "" {
			continue
		}
		directTestFiles[filepath.ToSlash(filepath.Clean(test.File))] = struct{}{}
	}

	candidates := typeScriptNearbyTestCandidatePaths(def.File)
	refs := make([]genericSymbolRef, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.ToSlash(filepath.Clean(candidate))
		if _, ok := directTestFiles[candidate]; ok {
			continue
		}

		absPath := filepath.Join(rootPath, filepath.FromSlash(candidate))
		if !isUsableNearbyTypeScriptTest(absPath, candidate, opts) {
			continue
		}
		line, snippet := firstNearbyTypeScriptTestSnippet(absPath)
		refs = append(refs, genericSymbolRef{
			File:    candidate,
			Line:    line,
			Snippet: snippet,
			IsTest:  true,
		})
	}
	return refs
}

func typeScriptNearbyTestCandidatePaths(defFile string) []string {
	cleanFile := filepath.ToSlash(filepath.Clean(defFile))
	dir := filepath.ToSlash(filepath.Dir(cleanFile))
	base := strings.TrimSuffix(filepath.Base(cleanFile), filepath.Ext(cleanFile))

	candidates := []string{
		filepath.ToSlash(filepath.Join(dir, base+".test.ts")),
		filepath.ToSlash(filepath.Join(dir, base+".spec.ts")),
		filepath.ToSlash(filepath.Join(dir, "__tests__", base+".test.ts")),
		filepath.ToSlash(filepath.Join("tests", base+".test.ts")),
	}
	return dedupeStringList(candidates)
}

func isUsableNearbyTypeScriptTest(absPath, displayPath string, opts SearchOptions) bool {
	if strings.ToLower(filepath.Ext(absPath)) != ".ts" {
		return false
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return false
	}
	if !nearbyTypeScriptTestInSearchScope(absPath, opts) {
		return false
	}
	if matchesSearchIgnoreFilter(displayPath, opts) {
		return false
	}
	return matchesSearchFileFilter(displayPath, opts)
}

func nearbyTypeScriptTestInSearchScope(absPath string, opts SearchOptions) bool {
	basis := resolveSearchPathBasisForOptions(opts)
	base := basis.Workdir
	if strings.TrimSpace(base) == "" {
		base = invocationCWDOrGetwd(opts)
	}
	target := strings.TrimSpace(basis.Target)
	if target == "" {
		target = "."
	}

	var targetPath string
	if filepath.IsAbs(target) {
		targetPath = filepath.Clean(target)
	} else {
		targetPath = filepath.Clean(filepath.Join(base, target))
	}

	info, err := os.Stat(targetPath)
	if err == nil && !info.IsDir() {
		return filepath.Clean(absPath) == targetPath
	}

	rel, err := filepath.Rel(targetPath, filepath.Clean(absPath))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func firstNearbyTypeScriptTestSnippet(absPath string) (int, string) {
	file, err := os.Open(absPath)
	if err != nil {
		return 1, ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			return line, text
		}
	}
	return 1, ""
}

func dedupeStringList(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
