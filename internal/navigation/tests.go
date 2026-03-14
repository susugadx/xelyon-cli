package navigation

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

// findRelatedTests はテストファイル内のシンボル参照からテスト関数を抽出する。
// refs は findReferences の結果全体を受け取り、_test.go のものだけを対象にする。
func findRelatedTests(symbol string, refs []Reference, limit int) ([]TestRef, bool) {
	// テストファイル参照を収集
	testFiles := make(map[string]bool)
	for _, ref := range refs {
		if ref.IsTest {
			testFiles[ref.File] = true
		}
	}

	if len(testFiles) == 0 {
		return nil, false
	}

	var tests []TestRef
	seen := make(map[string]bool)

	for file := range testFiles {
		absPath := mustAbs(file)
		symbols, err := ast.ExtractSymbols(absPath)
		if err != nil {
			continue
		}

		for _, s := range symbols {
			if !isTestFunction(s.Name) {
				continue
			}
			// テスト名にシンボル名が含まれるか、テスト本文にシンボルが出現するか
			if containsSymbolReference(s.Name, symbol, refs, file) {
				key := file + ":" + s.Name
				if seen[key] {
					continue
				}
				seen[key] = true
				tests = append(tests, TestRef{
					File:    file,
					Name:    s.Name,
					Line:    s.Line,
					EndLine: s.EndLine,
				})
			}
		}
	}

	if len(tests) > limit {
		return tests[:limit], true
	}
	return tests, false
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
