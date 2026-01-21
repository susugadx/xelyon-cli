package lsp

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// languageExtensions は拡張子 -> サーバーキーのマッピング
var languageExtensions = map[string]string{
	// Existing (4 languages)
	".go":  "go",
	".ts":  "typescript",
	".tsx": "typescript",
	".js":  "typescript", // JS/JSX も vtsls で対応（React含む）
	".jsx": "typescript",
	".py":  "python",
	".rs":  "rust",

	// Tier 1: Backend languages (11 languages)
	".java":  "java",
	".c":     "c",
	".h":     "c",
	".cpp":   "cpp",
	".hpp":   "cpp",
	".cc":    "cpp",
	".cxx":   "cpp",
	".hxx":   "cpp",
	".rb":    "ruby",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".swift": "swift",
	".cs":    "csharp",
	".scala": "scala",
	".php":   "php",
	".ex":    "elixir",
	".exs":   "elixir",
	".lua":   "lua",

	// Tier 2: Frontend languages (4 languages)
	".css":    "css",
	".scss":   "css",
	".sass":   "css",
	".less":   "css",
	".html":   "html",
	".htm":    "html",
	".vue":    "vue",
	".svelte": "svelte",

	// Tier 3: Config/Script languages (5 languages)
	".yaml":     "yaml",
	".yml":      "yaml",
	".toml":     "toml",
	".sql":      "sql",
	".sh":       "bash",
	".bash":     "bash",
	".zsh":      "bash",
	".md":       "markdown",
	".markdown": "markdown",
}

// skipDirs は検出対象外のディレクトリ
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	"dist":         true,
	"build":        true,
	"target":       true, // Rust
	".idea":        true,
	".vscode":      true,
}

// LanguageInfo は検出された言語の情報
type LanguageInfo struct {
	ServerKey  string   // LSP設定のキー（go, typescript等）
	Extensions []string // 検出された拡張子
	FileCount  int      // ファイル数
}

// DetectProjectLanguages はプロジェクト内の言語を検出
func DetectProjectLanguages(rootDir string) ([]LanguageInfo, error) {
	// serverKey -> (extension -> count)
	langStats := make(map[string]map[string]int)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーは無視して続行
		}

		// ディレクトリのスキップ判定
		if info.IsDir() {
			name := info.Name()
			if skipDirs[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// 拡張子から言語を判定
		ext := strings.ToLower(filepath.Ext(path))
		serverKey, ok := languageExtensions[ext]
		if !ok {
			return nil
		}

		// 統計を更新
		if langStats[serverKey] == nil {
			langStats[serverKey] = make(map[string]int)
		}
		langStats[serverKey][ext]++

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 結果を整形
	var result []LanguageInfo
	for serverKey, extCounts := range langStats {
		info := LanguageInfo{
			ServerKey:  serverKey,
			Extensions: make([]string, 0, len(extCounts)),
			FileCount:  0,
		}

		for ext, count := range extCounts {
			info.Extensions = append(info.Extensions, ext)
			info.FileCount += count
		}

		// 拡張子をソート
		sort.Strings(info.Extensions)
		result = append(result, info)
	}

	// サーバーキーでソート
	sort.Slice(result, func(i, j int) bool {
		return result[i].ServerKey < result[j].ServerKey
	})

	return result, nil
}

// GetLanguageExtensions は指定されたサーバーキーに対応する拡張子を返す
func GetLanguageExtensions(serverKey string) []string {
	var exts []string
	for ext, key := range languageExtensions {
		if key == serverKey {
			exts = append(exts, ext)
		}
	}
	sort.Strings(exts)
	return exts
}
