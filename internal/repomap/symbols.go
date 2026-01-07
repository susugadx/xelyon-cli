package repomap

// Symbol はコード内のシンボル（関数、クラス、構造体等）
type Symbol struct {
	Name       string // シンボル名
	Kind       string // "function", "struct", "class", "method", "interface"
	Signature  string // 完全なシグネチャ（func CreateUser(name string) error）
	FilePath   string // ファイルパス
	Line       int    // 行番号
	References int    // 参照回数（ランキング用）
}

// FileSymbols はファイル内のシンボル一覧
type FileSymbols struct {
	Path    string
	Symbols []Symbol
}
