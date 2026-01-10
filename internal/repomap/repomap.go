//go:build !norepomap
// +build !norepomap

package repomap

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RepoMap はリポジトリのコード構造マップ
type RepoMap struct {
	RootPath  string
	Files     []*FileSymbols
	MaxTokens int // トークン制限
}

// NewRepoMap は新しいRepoMapを作成
func NewRepoMap(rootPath string, maxTokens int) *RepoMap {
	return &RepoMap{
		RootPath:  rootPath,
		Files:     []*FileSymbols{},
		MaxTokens: maxTokens,
	}
}

// Build はリポジトリをスキャンしてマップを構築
func (rm *RepoMap) Build() error {
	return filepath.Walk(rm.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーはスキップ
		}

		// 除外パターン
		if shouldIgnore(path, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// サポートされているファイルのみ処理
		if !IsSupportedFile(path) {
			return nil
		}

		fileSymbols, err := ExtractSymbols(path)
		if err != nil || fileSymbols == nil {
			return nil
		}

		if len(fileSymbols.Symbols) > 0 {
			rm.Files = append(rm.Files, fileSymbols)
		}

		return nil
	})
}

// shouldIgnore は除外すべきパスかどうか
func shouldIgnore(path string, info os.FileInfo) bool {
	name := info.Name()

	// 隠しファイル/ディレクトリ
	if strings.HasPrefix(name, ".") {
		return true
	}

	// 一般的な除外パターン
	ignoreDirs := []string{
		"node_modules", "vendor", "dist", "build",
		"__pycache__", ".git", ".svn", "target",
		"bin", "obj", "coverage", ".next",
	}

	for _, ignore := range ignoreDirs {
		if name == ignore {
			return true
		}
	}

	return false
}

// Generate はRepo Map文字列を生成
func (rm *RepoMap) Generate() string {
	if len(rm.Files) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Repository Map\n\n")

	// ファイルをパスでソート
	sort.Slice(rm.Files, func(i, j int) bool {
		return rm.Files[i].Path < rm.Files[j].Path
	})

	totalTokens := 0
	estimatedTokensPerChar := 0.25 // 大まかな見積もり

	for _, file := range rm.Files {
		// 相対パス
		relPath, _ := filepath.Rel(rm.RootPath, file.Path)
		if relPath == "" {
			relPath = file.Path
		}

		// 文字列連結効率化（strings.Builder使用）
		var fileSB strings.Builder
		fileSB.WriteString(fmt.Sprintf("### %s\n", relPath))
		for _, sym := range file.Symbols {
			fileSB.WriteString(fmt.Sprintf("  %d: %s\n", sym.Line, sym.Signature))
		}
		fileSB.WriteString("\n")
		fileSection := fileSB.String()

		// トークン制限チェック
		sectionTokens := int(float64(len(fileSection)) * estimatedTokensPerChar)
		if rm.MaxTokens > 0 && totalTokens+sectionTokens > rm.MaxTokens {
			sb.WriteString("... (truncated due to token limit)\n")
			break
		}

		sb.WriteString(fileSection)
		totalTokens += sectionTokens
	}

	return sb.String()
}

// GetSymbolCount はシンボル総数を返す
func (rm *RepoMap) GetSymbolCount() int {
	count := 0
	for _, file := range rm.Files {
		count += len(file.Symbols)
	}
	return count
}
