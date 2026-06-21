package analysis

import "github.com/susugadx/xelyon-cli/internal/review/externaldoc"

const outsideRepoPathDisplay = "<outside-repo>"

// EvidenceInput は review evidence bundle を pure analysis 用に正規化した DTO。
type EvidenceInput struct {
	ChangeInventory     ChangeInventory
	ChangedFiles        []ChangedFile
	ChangedFileContext  []ContextFile
	RelatedContextFiles []ContextFile
	RelatedSearchHits   []RelatedSearchHit
	GenericImpact       GenericImpact
	WebSearchEvidence   externaldoc.WebSearchEvidence
	Diffs               []Diff
	UntrackedFiles      []UntrackedFile
	TruncationFlags     TruncationFlags
	KnownRuleFilePaths  []string
}

// TextBlock は本文と truncation flag を組にした DTO。
type TextBlock struct {
	Content   string
	Truncated bool
}

// ChangedFile は changed file の path 表示を正規化した DTO。
type ChangedFile struct {
	Path    string
	OldPath string
}

// ContextFile は context file evidence を analysis 用に表す。
type ContextFile struct {
	Path       string
	Role       string
	Truncated  bool
	Skipped    bool
	SkipReason string
}

// RelatedSearchHit は search hit evidence を analysis 用に表す。
type RelatedSearchHit struct {
	Path string
}

// GenericImpact は generic impact candidates の analysis 用表現。
type GenericImpact struct {
	Tokens     []string
	Candidates []GenericImpactCandidate
	Truncated  bool
}

// GenericImpactCandidate は generic impact candidate の analysis 用表現。
type GenericImpactCandidate struct {
	Path  string
	Role  string
	Token string
}

// ChangeInventory は変更 surface の一覧を analysis 用に表す。
type ChangeInventory struct {
	Generated    []string
	Tests        []string
	Docs         []string
	Config       []string
	Production   []string
	NewFiles     []string
	DeletedFiles []string
	RenamedFiles []string
	Untracked    []string
}

// Diff は diff evidence の analysis 用表現。
type Diff struct {
	Stat       TextBlock
	NameStatus TextBlock
	Diff       TextBlock
}

// UntrackedFile は untracked snapshot の path を analysis 用に表す。
type UntrackedFile struct {
	Path string
}

// TruncationFlags は bundle 全体の truncation 状態を固定順序で表す。
type TruncationFlags struct {
	StatusShort         bool
	Diffs               []DiffTruncation
	UntrackedList       bool
	RelatedCandidates   bool
	RelatedSearch       bool
	WebSearchEvidence   bool
	UntrackedSnapshots  bool
	UntrackedFiles      []PathTruncation
	RuleFiles           []PathTruncation
	ChangedFileContext  []PathTruncation
	RelatedContextFiles []PathTruncation
}

// DiffTruncation は diff ごとの truncation 状態を表す。
type DiffTruncation struct {
	Source     string
	Stat       bool
	NameStatus bool
	Diff       bool
}

// PathTruncation は path 付き evidence の truncation 状態を表す。
type PathTruncation struct {
	Path      string
	Truncated bool
}

// PressureSignalOptions は review pressure signal 生成時に親 package が持つ source of truth を渡す。
type PressureSignalOptions struct {
	KnownRuleFilePaths []string
}

// PressureSignal は Pass1 plan に注意喚起する deterministic signal。
type PressureSignal struct {
	Signal   string   `json:"signal"`
	Summary  string   `json:"summary"`
	Evidence []string `json:"evidence"`
}
