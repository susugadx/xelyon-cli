package search

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type isUsableJSFamilyNearbyTest func(absPath string, displayPath string) bool

func allJSFamilyTests(directTests []genericSymbolRef, nearbyTests []genericSymbolRef) []genericSymbolRef {
	tests := make([]genericSymbolRef, 0, len(directTests)+len(nearbyTests))
	tests = append(tests, directTests...)
	tests = append(tests, nearbyTests...)
	return tests
}

func findNearbyJSFamilyTests(rootPath string, candidates []string, directTests []genericSymbolRef, isUsable isUsableJSFamilyNearbyTest) []genericSymbolRef {
	if strings.TrimSpace(rootPath) == "" || len(candidates) == 0 || isUsable == nil {
		return nil
	}

	directTestFiles := jsFamilyDirectTestFiles(directTests)
	refs := make([]genericSymbolRef, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.ToSlash(filepath.Clean(candidate))
		if _, ok := directTestFiles[candidate]; ok {
			continue
		}

		absPath := filepath.Join(rootPath, filepath.FromSlash(candidate))
		if !isUsable(absPath, candidate) {
			continue
		}
		line, snippet := firstNearbyJSFamilyTestSnippet(absPath)
		refs = append(refs, genericSymbolRef{
			File:    candidate,
			Line:    line,
			Snippet: snippet,
			IsTest:  true,
		})
	}
	return refs
}

func jsFamilyDirectTestFiles(directTests []genericSymbolRef) map[string]struct{} {
	files := make(map[string]struct{}, len(directTests))
	for _, test := range directTests {
		if test.File == "" {
			continue
		}
		files[filepath.ToSlash(filepath.Clean(test.File))] = struct{}{}
	}
	return files
}

func isUsableNearbyJSFamilyTest(absPath string, displayPath string, opts SearchOptions) bool {
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return false
	}
	return jsFamilySearchCandidateAllowed(absPath, displayPath, opts)
}

func firstNearbyJSFamilyTestSnippet(absPath string) (int, string) {
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
