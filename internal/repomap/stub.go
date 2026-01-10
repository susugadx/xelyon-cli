//go:build norepomap
// +build norepomap

package repomap

// RepoMap はリポジトリのコード構造マップ（スタブ実装）
type RepoMap struct {
	RootPath  string
	Files     []*FileSymbols
	MaxTokens int
}

// FileSymbols はファイルのシンボル情報（スタブ実装）
type FileSymbols struct{}

// NewRepoMap は新しいRepoMapを作成（スタブ実装）
func NewRepoMap(rootPath string, maxTokens int) *RepoMap {
	return &RepoMap{
		RootPath:  rootPath,
		Files:     []*FileSymbols{},
		MaxTokens: maxTokens,
	}
}

// Build はリポジトリをスキャンしてマップを構築（スタブ実装）
func (rm *RepoMap) Build() error {
	return nil
}

// Generate はマップを文字列として生成（スタブ実装）
func (rm *RepoMap) Generate() string {
	return "⚠️  Repo Map feature is disabled in this build (requires CGO for tree-sitter)"
}

// GetSymbolCount はシンボル数を返す（スタブ実装）
func (rm *RepoMap) GetSymbolCount() int {
	return 0
}

// String はマップを文字列として返す（スタブ実装）
func (rm *RepoMap) String() string {
	return rm.Generate()
}
