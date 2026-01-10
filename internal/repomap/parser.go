// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// SupportedLanguages は対応言語のマップ
var SupportedLanguages = map[string]*sitter.Language{
	".go":  golang.GetLanguage(),
	".js":  javascript.GetLanguage(),
	".ts":  typescript.GetLanguage(),
	".py":  python.GetLanguage(),
	".jsx": javascript.GetLanguage(),
	".tsx": typescript.GetLanguage(),
	".mjs": javascript.GetLanguage(),
}

// GetLanguage はファイル拡張子から言語を取得
func GetLanguage(filePath string) *sitter.Language {
	ext := strings.ToLower(filepath.Ext(filePath))
	return SupportedLanguages[ext]
}

// IsSupportedFile はサポートされているファイルかどうか
func IsSupportedFile(filePath string) bool {
	return GetLanguage(filePath) != nil
}
