package navigation

import (
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

// findRelatedTests はテストファイル内のシンボル参照からテスト関数を抽出する。
// refs は findReferences の結果全体を受け取り、_test.go のものだけを対象にする。
func findRelatedTests(symbol string, refs []Reference, limit int) ([]TestRef, int, bool) {
	// テストファイル参照を収集
	testFiles := make(map[string]bool)
	for _, ref := range refs {
		if ref.IsTest {
			testFiles[ref.File] = true
		}
	}

	if len(testFiles) == 0 {
		return nil, 0, false
	}

	var results []TestRef
	seen := make(map[string]bool)

	var files []string
	for f := range testFiles {
		files = append(files, f)
	}
	sort.Strings(files)

	for _, f := range files {
		// テストファイルからテスト関数を抽出
		symbols, err := ast.ExtractSymbols(f)
		if err != nil {
			continue
		}
		for _, s := range symbols {
			if s.Kind != ast.SymbolFunction || !isTestFunction(s.Name) {
				continue
			}
			// テスト名にシンボル名が含まれるか、テスト本文にシンボルが出現するか
			if containsSymbolReference(s.Name, symbol, refs, f) {
				key := f + ":" + s.Name
				if seen[key] {
					continue
				}
				seen[key] = true
				results = append(results, TestRef{
					File:         toRelativePath(f),
					ResolvedPath: cleanNavigationResolvedPath(f),
					Line:         s.Line,
					Name:         s.Name,
				})
			}
		}
	}

	total := len(results)
	if total > limit {
		return results[:limit], total, true
	}

	return results, total, false
}

// isTestFunction はテスト関数名かどうかを判定する。
func isTestFunction(name string) bool {
	return strings.HasPrefix(name, "Test") ||
		strings.HasPrefix(name, "Benchmark") ||
		strings.HasPrefix(name, "Example")
}

// containsSymbolReference はテスト関数がシンボルを参照しているかを判定する。
// テスト名にシンボル名が含まれるか、テスト関数の行範囲内にシンボル参照があるかで判定する。
func containsSymbolReference(testName, symbol string, refs []Reference, testFile string) bool {
	// テスト名にシンボル名が含まれるか（例: TestBuild_Normal → Build）
	if strings.Contains(testName, symbol) {
		return true
	}

	// テスト関数のスコープ内にシンボル参照があるか
	for _, ref := range refs {
		if ref.File != testFile {
			continue
		}
		if ref.Scope != "" && strings.Contains(ref.Scope, testName) {
			return true
		}
	}

	return false
}
