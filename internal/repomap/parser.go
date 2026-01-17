//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/lua"
	markdown "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"
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
	// Tier 2.5: Elixir/Lua（Neovim/Phoenix人気）
	".ex":  elixir.GetLanguage(),
	".exs": elixir.GetLanguage(),
	".lua": lua.GetLanguage(),
	// Tier 3: CSS/SCSS/フロントエンド (#59)
	".css":  css.GetLanguage(),
	".scss": css.GetLanguage(), // SCSSはCSSパーサーで基本構文を抽出
	// Issue #64: HTML id/class属性抽出
	".html": html.GetLanguage(),
	".htm":  html.GetLanguage(),
	// Tier 4: 設定/マークアップ言語 (#60)
	".yaml":       yaml.GetLanguage(),
	".yml":        yaml.GetLanguage(),
	".toml":       toml.GetLanguage(),
	".sql":        sql.GetLanguage(),
	".sh":         bash.GetLanguage(),
	".bash":       bash.GetLanguage(),
	".zsh":        bash.GetLanguage(), // ZshはBashパーサーで基本構文を抽出
	".md":         markdown.GetLanguage(),
	".markdown":   markdown.GetLanguage(),
	"Dockerfile":  dockerfile.GetLanguage(),
	"Makefile":    nil, // Makefileは正規表現ベースで処理
	".mk":         nil, // Makefileの.mk拡張子
	"Jenkinsfile": nil, // Jenkinsfileは正規表現ベースで処理
	// Issue #63: Vue/Svelte SFC（ハイブリッド方式で処理）
	".vue":    nil,
	".svelte": nil,
}

// GetLanguage はファイル拡張子から言語を取得
func GetLanguage(filePath string) *sitter.Language {
	ext := strings.ToLower(filepath.Ext(filePath))
	if lang, ok := SupportedLanguages[ext]; ok {
		return lang
	}
	// 拡張子がない場合はファイル名で判定（Dockerfile, Makefile等）
	baseName := filepath.Base(filePath)
	return SupportedLanguages[baseName]
}

// IsSupportedFile はサポートされているファイルかどうか
func IsSupportedFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	if _, ok := SupportedLanguages[ext]; ok {
		return true
	}
	// 拡張子がない場合はファイル名で判定
	baseName := filepath.Base(filePath)
	_, ok := SupportedLanguages[baseName]
	return ok
}

// IsConfigFile は設定ファイル系言語かどうか（正規表現ベース抽出用）
func IsConfigFile(filePath string) bool {
	baseName := filepath.Base(filePath)
	switch baseName {
	case "Makefile", "Jenkinsfile":
		return true
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mk":
		return true
	}
	return false
}

// IsSFCFile は Vue/Svelte SFC ファイルかどうか（Issue #63）
func IsSFCFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".vue" || ext == ".svelte"
}

// IsTailwindConfigFile は Tailwind CSS 設定ファイルかどうか（Issue #65）
func IsTailwindConfigFile(filePath string) bool {
	baseName := filepath.Base(filePath)
	// tailwind.config.js, tailwind.config.ts, tailwind.config.mjs, tailwind.config.cjs
	return strings.HasPrefix(baseName, "tailwind.config.")
}
