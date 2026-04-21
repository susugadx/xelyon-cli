package navigation

import (
	"context"
	"path/filepath"
	"strings"
)

// findReferencesViaLSP は LSP の参照結果を navigation.Reference に変換する。
func findReferencesViaLSP(client LSPClient, cand SymbolCandidate, invocationCWD string) ([]Reference, error) {
	locations, err := queryLSPLocations(cand, func(ctx context.Context, filePath string, line, col int) ([]LSPLocation, error) {
		return client.FindReferences(ctx, filePath, line, col, false)
	})
	if err != nil {
		return nil, err
	}

	refs := make([]Reference, 0, len(locations))
	for _, loc := range locations {
		refs = append(refs, newReferenceFromLSPLocation(loc, cand, invocationCWD))
	}
	return refs, nil
}

// findImplementationsViaLSP は LSP の実装検索結果を navigation.ImplementationRef に変換する。
func findImplementationsViaLSP(client LSPClient, cand SymbolCandidate, invocationCWD string) ([]ImplementationRef, error) {
	locations, err := queryLSPLocations(cand, func(ctx context.Context, filePath string, line, col int) ([]LSPLocation, error) {
		return client.GotoImplementation(ctx, filePath, line, col)
	})
	if err != nil {
		return nil, err
	}

	impls := make([]ImplementationRef, 0, len(locations))
	for _, loc := range locations {
		impls = append(impls, newImplementationFromLSPLocation(loc, cand, invocationCWD))
	}
	return impls, nil
}

// lspLocationQuery はシンボル位置を取得する LSP クエリ関数。
type lspLocationQuery func(ctx context.Context, filePath string, line, character int) ([]LSPLocation, error)

func queryLSPLocations(cand SymbolCandidate, query lspLocationQuery) ([]LSPLocation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lspReferenceTimeout)
	defer cancel()

	col, err := findSymbolColumn(cand)
	if err != nil {
		return nil, err
	}
	return query(ctx, cand.File, cand.Line, col)
}

func newReferenceFromLSPLocation(loc LSPLocation, cand SymbolCandidate, invocationCWD string) Reference {
	filePath := lspLocationFilePath(loc.File, cand.RootPath, invocationCWD)
	return Reference{
		File:         filePath,
		ResolvedPath: cleanNavigationResolvedPath(filePath),
		Line:         loc.Line,
		Scope:        findEnclosingFunction(filePath, loc.Line),
		Snippet:      readLineSnippet(filePath, loc.Line),
		IsTest:       isTestFile(filePath),
		Class:        classifyLineByAST(filePath, loc.Line, cand.Name),
	}
}

func newImplementationFromLSPLocation(loc LSPLocation, cand SymbolCandidate, invocationCWD string) ImplementationRef {
	filePath := lspLocationFilePath(loc.File, cand.RootPath, invocationCWD)
	return ImplementationRef{
		File:         filePath,
		ResolvedPath: cleanNavigationResolvedPath(filePath),
		Line:         loc.Line,
		Name:         findTypeNameAtLine(filePath, loc.Line),
	}
}

func lspLocationFilePath(file, rootPath, invocationCWD string) string {
	file = strings.TrimSpace(file)
	if file == "" || filepath.IsAbs(file) {
		return file
	}
	file = filepath.Clean(filepath.FromSlash(file))
	if resolved, ok := resolveExistingRelativeLSPPath(invocationCWD, file); ok {
		return resolved
	}
	if resolved, ok := resolveExistingRelativeLSPPath(rootPath, file); ok {
		return resolved
	}
	if base := strings.TrimSpace(invocationCWD); base != "" {
		return filepath.Join(base, file)
	}
	if base := strings.TrimSpace(rootPath); base != "" {
		return filepath.Join(base, file)
	}
	return file
}

func resolveExistingRelativeLSPPath(base, file string) (string, bool) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", false
	}
	candidate := filepath.Join(base, file)
	if !pathExists(candidate) {
		return "", false
	}
	return candidate, true
}
