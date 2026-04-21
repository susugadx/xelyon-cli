package navigation

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"os"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

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
