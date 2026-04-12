package navigation

import (
	"bufio"
	"context"
	goast "go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// maxRipgrepResults は findReferences が返す参照の上限。
// これを超えた場合は truncated=true を返す。
const maxRipgrepResults = 500

const (
	ripgrepScannerInitialBufferSize = 64 * 1024
	ripgrepScannerMaxBufferSize     = 1024 * 1024
	lspReferenceTimeout             = 5 * time.Second
)

// referenceSearchResult は ripgrep 参照検索の内部状態を保持する。
type referenceSearchResult struct {
	Refs          []Reference
	Truncated     bool
	Incomplete    bool
	StopRequested bool
}

// referenceParseCache は同一ファイルの重複パースを防ぐキャッシュ。
type referenceParseCache struct {
	files map[string]*cachedFile
}

// cachedFile はファイルごとのパース済みデータを保持する。
type cachedFile struct {
	src         []byte
	tsParsed    *ast.ParsedFile
	tsAttempted bool
	goFile      *goast.File
	goFSet      *token.FileSet
	goImports   map[string]bool
	goAttempted bool
}

func newReferenceParseCache() *referenceParseCache {
	return &referenceParseCache{files: make(map[string]*cachedFile)}
}

func (c *referenceParseCache) get(absPath string) *cachedFile {
	cf, exists := c.files[absPath]
	if exists {
		return cf
	}
	src, err := os.ReadFile(absPath)
	if err != nil {
		c.files[absPath] = nil
		return nil
	}
	cf = &cachedFile{src: src}
	c.files[absPath] = cf
	return cf
}

func (cf *cachedFile) ensureTreeSitter(absPath string) *ast.ParsedFile {
	if cf.tsAttempted {
		return cf.tsParsed
	}
	cf.tsAttempted = true
	pf, err := ast.ParseBytesForReuse(absPath, cf.src)
	if err != nil {
		return nil
	}
	cf.tsParsed = pf
	return pf
}

func (cf *cachedFile) ensureGoParser(absPath string) (*goast.File, *token.FileSet, map[string]bool) {
	if cf.goAttempted {
		return cf.goFile, cf.goFSet, cf.goImports
	}
	cf.goAttempted = true
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, cf.src, parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, nil
	}
	cf.goFile = file
	cf.goFSet = fset
	cf.goImports = importedPackageNames(file)
	return cf.goFile, cf.goFSet, cf.goImports
}

// findReferences は ripgrep でシンボル名を検索し、全参照を返す。
// StdoutPipe + scanner で逐次読み取りし、201件目を検出したら早期停止する。
// truncated が true の場合、上流の検索結果が上限を超えたことを示す。
// incomplete が true の場合、読み取り失敗や異常終了により結果が不完全であることを示す。
func findReferences(symbol string) (refs []Reference, truncated bool, incomplete bool) {
	if !common.IsRipgrepAvailable() {
		return nil, false, false
	}

	args := []string{
		"-n",
		"--no-heading",
		"--color", "never",
		"-w", // 単語境界
		"--type", "go",
		"--glob", "!vendor/",
		symbol,
		".",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, true
	}
	if err := cmd.Start(); err != nil {
		return nil, false, true
	}

	return runReferenceSearch(stdout, symbol, cancel, cmd.Wait)
}

// collectReferenceSearchResult は ripgrep の標準出力を読み取り、参照一覧を構築する。
func collectReferenceSearchResult(reader io.Reader, symbol string) referenceSearchResult {
	result := referenceSearchResult{}
	if reader == nil {
		result.Incomplete = true
		return result
	}

	cache := newReferenceParseCache()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, ripgrepScannerInitialBufferSize), ripgrepScannerMaxBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		ref := parseRipgrepLine(line, symbol, cache)
		if ref == nil {
			continue
		}
		result.Refs = append(result.Refs, *ref)

		// 201件目を検出したら truncated=true にし、先頭200件のみ保持して早期停止
		if len(result.Refs) > maxRipgrepResults {
			result.Truncated = true
			result.Refs = result.Refs[:maxRipgrepResults]
			result.StopRequested = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		result.Incomplete = true
	}

	return result
}

// runReferenceSearch は参照ストリームの読み取りと終了待機をまとめて処理する。
func runReferenceSearch(reader io.Reader, symbol string, cancel func(), wait func() error) ([]Reference, bool, bool) {
	result := collectReferenceSearchResult(reader, symbol)
	if result.StopRequested && cancel != nil {
		cancel()
	}
	if wait != nil {
		if err := wait(); err != nil && !result.StopRequested {
			result.Incomplete = true
		}
	}
	return result.Refs, result.Truncated, result.Incomplete
}

func findReferencesWithFallbackRuntime(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime) ([]Reference, bool, bool) {
	ambiguousFiles := findAmbiguousFilesWithRuntime(baseName, cand, runtime)
	allRefs, truncated, incomplete := findReferences(baseName)
	return filterRefsByCandidate(allRefs, cand, ambiguousFiles), truncated, incomplete
}

// findReferencesViaLSP resolves references through the LSP client and converts them to Reference values.
func findReferencesViaLSP(client LSPClient, cand SymbolCandidate, invocationCWD string) ([]Reference, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lspReferenceTimeout)
	defer cancel()

	col, err := findSymbolColumn(cand)
	if err != nil {
		return nil, err
	}

	locations, err := client.FindReferences(ctx, cand.File, cand.Line, col, false)
	if err != nil {
		return nil, err
	}

	refs := make([]Reference, 0, len(locations))
	for _, loc := range locations {
		filePath := lspLocationFilePath(loc.File, cand.RootPath, invocationCWD)
		refs = append(refs, Reference{
			File:         filePath,
			ResolvedPath: cleanNavigationResolvedPath(filePath),
			Line:         loc.Line,
			Scope:        findEnclosingFunction(filePath, loc.Line),
			Snippet:      readLineSnippet(filePath, loc.Line),
			IsTest:       isTestFile(filePath),
			Class:        classifyLineByAST(filePath, loc.Line, cand.Name),
		})
	}
	return refs, nil
}

// findImplementationsViaLSP resolves interface implementations through the LSP client.
func findImplementationsViaLSP(client LSPClient, cand SymbolCandidate, invocationCWD string) ([]ImplementationRef, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lspReferenceTimeout)
	defer cancel()

	col, err := findSymbolColumn(cand)
	if err != nil {
		return nil, err
	}

	locations, err := client.GotoImplementation(ctx, cand.File, cand.Line, col)
	if err != nil {
		return nil, err
	}

	impls := make([]ImplementationRef, 0, len(locations))
	for _, loc := range locations {
		filePath := lspLocationFilePath(loc.File, cand.RootPath, invocationCWD)
		impls = append(impls, ImplementationRef{
			File:         filePath,
			ResolvedPath: cleanNavigationResolvedPath(filePath),
			Line:         loc.Line,
			Name:         findTypeNameAtLine(filePath, loc.Line),
		})
	}
	return impls, nil
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

// findSymbolColumn returns the 1-indexed column of the symbol name on the candidate line.
