package search

// SymbolBundle は編集直結の最小シンボル surface をまとめた結果。
type SymbolBundle struct {
	Identity    SymbolBundleIdentity
	Definition  SymbolBundleDefinition
	Sections    []SymbolBundleSection
	Impact      *SymbolBundleImpact
	Diagnostics SymbolBundleDiagnostics
	Debug       SymbolBundleDebug
}

// SymbolBundleIdentity は bundle のシンボル同一性を表す。
type SymbolBundleIdentity struct {
	Language    string
	Query       string
	Canonical   string
	DisplayName string
	Kind        string
	File        string
	Line        int
	EndLine     int
}

// SymbolBundleDefinition は編集起点になる定義情報。
type SymbolBundleDefinition struct {
	File      string
	Line      int
	EndLine   int
	Signature string
	Body      []string
}

// SymbolBundleSection は caller/test などの代表 surface 群。
type SymbolBundleSection struct {
	Kind  string
	Title string
	Items []SymbolBundleItem
	Total int
	More  bool
}

// SymbolBundleImpact は impact intent 向けの付加メタデータ。
type SymbolBundleImpact struct {
	RiskLevel        string
	RecommendedReads []SymbolBundleItem
}

// SymbolBundleDiagnostics は bundle の注意喚起メタデータ。
type SymbolBundleDiagnostics struct {
	ResolvedViaLSP     bool
	UpstreamTruncated  bool
	UpstreamIncomplete bool
}

// SymbolBundleItem は bundle 内の 1 つの編集候補箇所。
type SymbolBundleItem struct {
	Kind    string
	File    string
	Line    int
	EndLine int
	Snippet string
	Scope   string
	Name    string
	IsTest  bool
}

// SymbolBundleDebug は router / builder の内部メタデータ。
type SymbolBundleDebug struct {
	Source          string
	FileRootPath    string
	Route           searchRouteTrace
	MatchedPatterns []string
	DependencyFiles []string
}
