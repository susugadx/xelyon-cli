package ast

import "github.com/odvcencio/gotreesitter"

// SymbolKind はシンボルの種別を表す。
type SymbolKind string

const (
	SymbolFunction  SymbolKind = "function"
	SymbolMethod    SymbolKind = "method"
	SymbolType      SymbolKind = "type"
	SymbolInterface SymbolKind = "interface"
	SymbolStruct    SymbolKind = "struct"
	SymbolConst     SymbolKind = "const"
	SymbolVar       SymbolKind = "var"
	SymbolClass     SymbolKind = "class"
	SymbolEnum      SymbolKind = "enum"
	SymbolTrait     SymbolKind = "trait"
	SymbolImpl      SymbolKind = "impl"
	SymbolUnknown   SymbolKind = "unknown"
)

// MatchClass は検索マッチの分類を表す。
type MatchClass string

const (
	ClassDef     MatchClass = "def"
	ClassCall    MatchClass = "call"
	ClassRef     MatchClass = "ref"
	ClassImport  MatchClass = "import"
	ClassComment MatchClass = "comment"
	ClassString  MatchClass = "string"
	ClassUnknown MatchClass = "unknown"
)

// Symbol はコード内の定義シンボルを表す。
type Symbol struct {
	Name      string
	Kind      SymbolKind
	Signature string
	Line      int
	EndLine   int
	Exported  bool
}

// MatchInfo は特定行のマッチ分類情報を表す。
type MatchInfo struct {
	Class        MatchClass
	Scope        string
	NodeType     string
	SelectorKind string
	ReceiverType string
}

// ParsedFile はパース済みファイルの情報を保持する。再利用のためにキャッシュできる。
type ParsedFile struct {
	tree *gotreesitter.Tree
	src  []byte
}

// SyntaxError は AST 構文検証で検出した構文エラー情報を表す。
type SyntaxError struct {
	Line    int
	Column  int
	Message string
}
