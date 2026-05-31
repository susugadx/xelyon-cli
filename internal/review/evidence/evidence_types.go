package evidence

import (
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

// TargetKind は review 対象の種類を表す。
type TargetKind = domain.TargetKind

const (
	// TargetCurrentChanges は現在の作業ツリー差分を review 対象にする。
	TargetCurrentChanges = domain.TargetCurrentChanges
)

// ReviewWebSearchEvidence は /review 用の外部 Web 検索 evidence を表す。
type ReviewWebSearchEvidence = externaldoc.WebSearchEvidence

// ReviewWebSearchEvidenceQuery は 1 件の検索 query と結果を表す。
type ReviewWebSearchEvidenceQuery = externaldoc.WebSearchEvidenceQuery

// ReviewWebSearchEvidenceResult は検索結果 URL と discovery-only snippet を表す。
type ReviewWebSearchEvidenceResult = externaldoc.WebSearchEvidenceResult

// ReviewWebSearchQueryResult は検索 provider と URL 付き結果を表す。
type ReviewWebSearchQueryResult = externaldoc.WebSearchQueryResult

// ReviewExternalDocEvidence は検索結果 URL から取得した引用可能 snippet 群を表す。
type ReviewExternalDocEvidence = externaldoc.Evidence

// ReviewExternalDocSnippetEvidence は external_doc evidence の引用可能な bounded snippet。
type ReviewExternalDocSnippetEvidence = externaldoc.SnippetEvidence

// ReviewExternalDocFetchRequest は external_doc fetch 境界へ渡す検索結果 URL と判定 hint。
type ReviewExternalDocFetchRequest = externaldoc.FetchRequest

// ReviewExternalDocFocusTerm は external_doc snippet で優先して引用範囲へ寄せる語句。
type ReviewExternalDocFocusTerm = externaldoc.FocusTerm

// ReviewExternalDocFetcher は検索結果 URL から external_doc snippet を取得する境界。
type ReviewExternalDocFetcher = externaldoc.Fetcher

// HTTPReviewExternalDocFetcher は HTTPS URL から bounded text snippet を取得する。
type HTTPReviewExternalDocFetcher = externaldoc.HTTPFetcher

// ReviewEvidenceBundle は `/review current_changes` の Pass1 入力に渡す一次情報を表す。
// ReviewReport schema ではなく、review 実行時の入力契約として扱う。
type ReviewEvidenceBundle struct {
	TargetKind TargetKind
	RepoRoot   string
	CWD        string

	StatusShort          string
	StatusShortTruncated bool
	Diffs                []ReviewDiffEvidence

	ChangedFiles                         []ReviewChangedFile
	ChangedFileContext                   []ReviewContextFileEvidence
	RelatedContextFiles                  []ReviewContextFileEvidence
	RelatedSearchHits                    []ReviewRelatedSearchHit
	GenericImpactCandidatePaths          []string
	GenericImpactCandidateListTruncated  bool
	GenericImpactCandidatePathsCollected bool
	GenericImpactCandidates              ReviewGenericImpactCandidates
	WebSearchEvidence                    ReviewWebSearchEvidence
	UntrackedFiles                       []ReviewUntrackedFile
	RelatedCandidateListTruncated        bool
	RelatedSearchTruncated               bool
	UntrackedListTruncated               bool
	// UntrackedSnapshotsTruncated は snapshot 読み取りの file count / total bytes budget 到達を表す。
	UntrackedSnapshotsTruncated bool
	RuleFiles                   []ReviewRuleFileEvidence
	Inventory                   ReviewChangeInventory
	Limits                      ReviewEvidenceLimits
}

// ReviewDiffEvidence は staged / unstaged それぞれの diff 一次情報を保持する。
type ReviewDiffEvidence struct {
	Source string

	Stat          string
	StatTruncated bool

	NameStatus          string
	NameStatusTruncated bool

	Diff          string
	DiffTruncated bool
}

// ReviewChangedFile は git diff --name-status 由来の変更 file を表す。
type ReviewChangedFile struct {
	Path     string
	OldPath  string
	Status   string
	Staged   bool
	Unstaged bool
}

// ReviewContextFileEvidence は changed file と近傍 context file の安全に制限された snapshot を表す。
type ReviewContextFileEvidence struct {
	Path       string
	Role       string
	Content    string
	Truncated  bool
	Skipped    bool
	SkipReason string
	SizeBytes  int64
	ReadBytes  int64
}

// ReviewRelatedSearchHit は changed file 由来の軽量 search term に一致した repo-local 行を表す。
type ReviewRelatedSearchHit struct {
	Path    string
	Line    int
	Snippet string
	Reason  string
}

// ReviewGenericImpactCandidates は言語非依存 heuristic で広げた review 用 impact 候補を表す。
// 完全な import/caller graph ではなく、Pass1 の impact surface 検討に使う lead として扱う。
type ReviewGenericImpactCandidates struct {
	Tokens     []string
	Candidates []ReviewGenericImpactCandidate
	Truncated  bool
}

// ReviewGenericImpactCandidate は generic impact expansion が検出した 1 候補を表す。
type ReviewGenericImpactCandidate struct {
	Path    string
	Role    string
	Reason  string
	Token   string
	Line    int
	Snippet string
}

// ReviewUntrackedFile は untracked path の安全に制限された snapshot または symlink metadata を表す。
type ReviewUntrackedFile struct {
	Path       string
	Symlink    bool
	LinkTarget string
	Snapshot   string
	Binary     bool
	Truncated  bool
	SizeBytes  int64
	ReadBytes  int64
}

// ReviewRuleFileEvidence は review 方針に影響する repo-local rule file を表す。
type ReviewRuleFileEvidence struct {
	Path      string
	Content   string
	Truncated bool
	SizeBytes int64
}

// ReviewChangeInventory は変更 surface を分類した一覧を表す。
type ReviewChangeInventory struct {
	Generated  []string
	Tests      []string
	Docs       []string
	Config     []string
	Production []string

	NewFiles     []string
	DeletedFiles []string
	RenamedFiles []string
	Untracked    []string
}

// ReviewEvidenceLimits は EvidenceBuilder の resource budget を表す。
type ReviewEvidenceLimits struct {
	MaxCommandOutputBytes      int64
	MaxUntrackedFileBytes      int64
	MaxRuleFileBytes           int64
	MaxTotalUntrackedBytes     int64
	MaxUntrackedFiles          int
	MaxContextFileBytes        int64
	MaxTotalContextBytes       int64
	MaxContextFiles            int
	MaxRelatedSearchTerms      int
	MaxRelatedSearchFiles      int
	MaxTotalRelatedSearchBytes int64
	MaxRelatedSearchFileBytes  int64
	MaxRelatedSearchHits       int
	MaxSearchSnippetBytes      int64
	CommandTimeout             time.Duration
}
