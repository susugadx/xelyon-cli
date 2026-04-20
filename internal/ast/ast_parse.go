package ast

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

var goTreeSitterMu sync.Mutex

// IsSupportedFile は AST 解析に対応しているファイルかを返す。
// Phase 1: Go のみ。Phase 2 で grammars.DetectLanguage に差し替え予定。
func IsSupportedFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".go")
}

// ParseFile はファイルをパースして AST ツリーとソースコードを返す。
func ParseFile(path string) (*gotreesitter.Tree, []byte, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseBytes(path, src)
}

// ParseBytes はソースコードバイト列をパースして AST ツリーとソースコードを返す。
func ParseBytes(path string, src []byte) (*gotreesitter.Tree, []byte, error) {
	if !IsSupportedFile(path) {
		return nil, nil, fmt.Errorf("unsupported language: %s", path)
	}

	tree, err := parseGoSource(src)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return tree, src, nil
}

// ParseBytesForReuse はファイルをパースして再利用可能な結果を返す。
// 同一ファイルの複数行分類でパースコストを削減する。
func ParseBytesForReuse(path string, src []byte) (*ParsedFile, error) {
	tree, src, err := ParseBytes(path, src)
	if err != nil {
		return nil, err
	}
	return &ParsedFile{tree: tree, src: src}, nil
}

func parseGoSource(src []byte) (*gotreesitter.Tree, error) {
	goTreeSitterMu.Lock()
	defer goTreeSitterMu.Unlock()

	parser := gotreesitter.NewParser(grammars.GoLanguage())
	return parser.Parse(src)
}
