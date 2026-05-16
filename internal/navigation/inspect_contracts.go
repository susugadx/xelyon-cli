package navigation

import "github.com/susugadx/xelyon-cli/internal/ast"

// InspectMode は inspect_symbol の出力モード。
type InspectMode string

const (
	ModeSummary InspectMode = "summary"
	ModeFull    InspectMode = "full"
)

// Budget は inspect_symbol の出力上限。
type Budget struct {
	BodyLines   int
	CallerLimit int
	RefLimit    int
	TestLimit   int
}

// ReferenceFilter は参照候補を caller/ref/test の分類と budget 適用前に絞り込む。
// LSP / ripgrep fallback では AST 分類前にも呼ばれるため、path / line / IsTest
// の基本フィールドだけに依存する。
type ReferenceFilter func(Reference) bool

// SummaryBudget は summary モードの出力上限。
var SummaryBudget = Budget{
	BodyLines:   15,
	CallerLimit: 2,
	RefLimit:    2,
	TestLimit:   1,
}

// FullBudget は full モードの出力上限。
var FullBudget = Budget{
	BodyLines:   999999,
	CallerLimit: 999999,
	RefLimit:    999999,
	TestLimit:   999999,
}

// SymbolCandidate はシンボル候補。
type SymbolCandidate struct {
	Name               string
	Kind               string
	File               string // プロジェクトルートからの相対パス
	Line               int
	EndLine            int
	Receiver           string // メソッド時のレシーバ型（例: *Config, Config）
	ReceiverNorm       string
	Signature          string
	Exported           bool
	PackageDir         string
	StableKey          string
	StableKeyCollision bool
	RootPath           string
}

// InspectResult は inspect_symbol の結果。
type InspectResult struct {
	// 単一候補の場合
	Symbol          *SymbolCandidate
	Body            []string // 行番号付き本文
	Callers         []Reference
	Refs            []Reference
	Tests           []TestRef
	ResolvedViaLSP  bool
	Implementations []ImplementationRef

	// 複数候補の場合
	Candidates []SymbolCandidate

	// 打ち切り情報
	TotalCallers       int
	TotalRefs          int
	TotalTests         int
	MoreCallers        bool
	MoreRefs           bool
	MoreTests          bool
	UpstreamTruncated  bool
	UpstreamIncomplete bool
}

// ImplementationRef は LSP で得られた実装位置を表す。
type ImplementationRef struct {
	File         string
	ResolvedPath string
	Line         int
	Name         string
}

// Reference はシンボル参照。
type Reference struct {
	File         string
	ResolvedPath string
	Line         int
	Scope        string // 包含関数名
	Snippet      string // マッチ行テキスト
	IsTest       bool
	Class        ast.MatchClass // AST 分類（ClassCall, ClassRef, ClassDef 等）
	NodeType     string         // マッチした識別子ノード型（identifier / field_identifier など）
	SelectorKind string         // selector の種別（package / method / unknown）
	ReceiverType string         // method selector の推定レシーバ型
}

// TestRef は関連テストの参照情報。
type TestRef struct {
	File         string
	ResolvedPath string
	Name         string
	Line         int
}
