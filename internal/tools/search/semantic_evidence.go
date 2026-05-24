package search

const (
	SemanticReferenceKindDefinition     = "definition"
	SemanticReferenceKindCaller         = "caller"
	SemanticReferenceKindReference      = "reference"
	SemanticReferenceKindImport         = "import"
	SemanticReferenceKindExport         = "export"
	SemanticReferenceKindTypeRef        = "type_ref"
	SemanticReferenceKindTest           = "test"
	SemanticReferenceKindNearbyTest     = "nearby_test"
	SemanticReferenceKindImplementation = "implementation"
)

const (
	SemanticReferenceSectionKindCallers         = "callers"
	SemanticReferenceSectionKindReferences      = "references"
	SemanticReferenceSectionKindTests           = "tests"
	SemanticReferenceSectionKindImplementations = "implementations"
	SemanticReferenceSectionKindImports         = "imports"
	SemanticReferenceSectionKindTypeRefs        = "type_refs"
)

// SemanticEvidence は言語 resolver が集めた定義・参照 evidence の共通 contract。
type SemanticEvidence struct {
	Language          string
	Query             string
	Symbol            string
	Definitions       []SemanticDefinition
	References        []SemanticReference
	ReferenceSections []SemanticReferenceSection
	Diagnostics       *SymbolBundleDiagnostics
	Source            string
	Confidence        string
	RiskLevel         string
}

// SemanticDefinition は編集起点になる定義 evidence。
type SemanticDefinition struct {
	Name           string
	Symbol         string
	DisplayName    string
	Canonical      string
	Kind           string
	Exported       bool
	Implementation bool
	Declaration    bool
	File           string
	ResolvedPath   string
	Line           int
	EndLine        int
	Signature      string
	Body           []string
	RootPath       string
	Source         string
	Confidence     string
}

// SemanticReferenceSection は参照 section の全体件数と省略有無を運ぶ。
type SemanticReferenceSection struct {
	Kind  string
	Total int
	More  bool
}

// SemanticReference は caller/import/test などの参照 evidence。
type SemanticReference struct {
	Kind         string
	File         string
	ResolvedPath string
	Line         int
	EndLine      int
	Snippet      string
	Scope        string
	Name         string
	IsTest       bool
	Source       string
	Confidence   string
}

// LanguageImpactScope は定義解決と evidence 収集の探索条件を分けて渡す adapter contract。
type LanguageImpactScope struct {
	Definition SearchOptions
	Evidence   SearchOptions
}

// LanguageImpactAdapter は言語ごとの semantic evidence 収集境界。
type LanguageImpactAdapter interface {
	Language() string
	CanHandle(opts SearchOptions) bool
	CollectSemanticEvidence(symbol string, scope LanguageImpactScope) (SemanticEvidence, bool)
}
