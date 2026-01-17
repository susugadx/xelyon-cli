//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// SupportedLanguages は対応言語のマップ
var SupportedLanguages = map[string]*sitter.Language{
	// 既存言語
	".go":  golang.GetLanguage(),
	".js":  javascript.GetLanguage(),
	".ts":  typescript.GetLanguage(),
	".py":  python.GetLanguage(),
	".jsx": javascript.GetLanguage(),
	".tsx": typescript.GetLanguage(),
	".mjs": javascript.GetLanguage(),
	// Tier 1: 高優先度言語
	".rs":   rust.GetLanguage(),
	".java": java.GetLanguage(),
	".c":    c.GetLanguage(),
	".h":    c.GetLanguage(),
	".cpp":  cpp.GetLanguage(),
	".hpp":  cpp.GetLanguage(),
	".cc":   cpp.GetLanguage(),
	".rb":   ruby.GetLanguage(),
	// Tier 2: 中優先度言語
	".kt":    kotlin.GetLanguage(),
	".kts":   kotlin.GetLanguage(),
	".swift": swift.GetLanguage(),
	".cs":    csharp.GetLanguage(),
	".scala": scala.GetLanguage(),
	".php":   php.GetLanguage(),
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
