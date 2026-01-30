//go:build !norepomap
// +build !norepomap

package repomap

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
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

// Build はリポジトリをスキャンしてマップを構築（並列処理 + キャッシュ対応）
func (rm *RepoMap) Build() error {
	// キャッシュ読み込み（失敗しても続行）
	cache, _ := LoadCache(rm.RootPath)
	cachedFiles := make(map[string]*CachedFile)
	if cache != nil {
		cachedFiles = cache.Files
	}

	// git status で変更ファイル取得（オプション）
	gitChanged := getGitChangedFiles(rm.RootPath)

	// ファイル一覧を収集
	var files []string
	err := filepath.Walk(rm.RootPath, func(path string, info os.FileInfo, err error) error {
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
		if IsSupportedFile(path) {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 並列処理
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]*FileSymbols, 0, len(files))
	newCache := &RepoMapCache{
		RootPath:  rm.RootPath,
		UpdatedAt: time.Now(),
		Files:     make(map[string]*CachedFile),
	}

	// ワーカー数 = CPU数（最大16）
	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16
	}
	sem := make(chan struct{}, numWorkers)

	for _, file := range files {
		wg.Add(1)
		sem <- struct{}{}

		go func(f string) {
			defer wg.Done()
			defer func() { <-sem }()

			info, err := os.Stat(f)
			if err != nil {
				return // ファイルが存在しない（削除された）
			}
			modTime := info.ModTime()

			var symbols []Symbol
			needReparse := true

			// キャッシュヒット判定
			if cached, ok := cachedFiles[f]; ok {
				// git status がある場合: 変更ファイルリストに含まれてなければキャッシュ使用
				if gitChanged != nil {
					if !gitChanged[f] && cached.ModTime.Equal(modTime) {
						symbols = cached.Symbols
						needReparse = false
					}
				} else {
					// git なし: ModTime のみで判定
					if cached.ModTime.Equal(modTime) {
						symbols = cached.Symbols
						needReparse = false
					}
				}
			}

			// キャッシュミス: 再解析
			if needReparse {
				fileSymbols, err := ExtractSymbols(f)
				if err != nil || fileSymbols == nil {
					return
				}
				symbols = fileSymbols.Symbols
			}

			if len(symbols) > 0 {
				mu.Lock()
				results = append(results, &FileSymbols{Path: f, Symbols: symbols})
				newCache.Files[f] = &CachedFile{
					Path:    f,
					ModTime: modTime,
					Symbols: symbols,
				}
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()
	rm.Files = results

	// キャッシュ保存（失敗しても続行）
	_ = SaveCache(newCache)

	return nil
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
