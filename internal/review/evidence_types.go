package review

import "time"

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

// ReviewWebSearchEvidence は /review 用の外部 Web 検索 evidence を表す。
type ReviewWebSearchEvidence struct {
	Enabled      bool                           `json:"enabled"`
	Provider     string                         `json:"provider,omitempty"`
	Queries      []ReviewWebSearchEvidenceQuery `json:"queries,omitempty"`
	ExternalDocs []ReviewExternalDocEvidence    `json:"external_docs,omitempty"`
	Error        string                         `json:"error,omitempty"`
	Truncated    bool                           `json:"truncated,omitempty"`
	Inconclusive bool                           `json:"inconclusive,omitempty"`
}

// ReviewWebSearchEvidenceQuery は 1 件の検索 query と結果を表す。
type ReviewWebSearchEvidenceQuery struct {
	Query   string                          `json:"query"`
	Reason  string                          `json:"reason"`
	Results []ReviewWebSearchEvidenceResult `json:"results,omitempty"`
	Error   string                          `json:"error,omitempty"`
}

// ReviewWebSearchEvidenceResult は検索結果 URL と discovery-only snippet を表す。
type ReviewWebSearchEvidenceResult struct {
	Title        string `json:"title,omitempty"`
	URL          string `json:"url"`
	Snippet      string `json:"snippet,omitempty"`
	SourceDomain string `json:"source_domain,omitempty"`
}

// ReviewExternalDocEvidence は検索結果 URL から取得した引用可能 snippet 群を表す。
type ReviewExternalDocEvidence struct {
	DocID        string                             `json:"doc_id"`
	URL          string                             `json:"url"`
	SourceDomain string                             `json:"source_domain,omitempty"`
	FetchedAt    time.Time                          `json:"fetched_at,omitempty"`
	StatusCode   int                                `json:"status_code,omitempty"`
	ContentType  string                             `json:"content_type,omitempty"`
	ContentHash  string                             `json:"content_hash,omitempty"`
	Truncated    bool                               `json:"truncated,omitempty"`
	Snippets     []ReviewExternalDocSnippetEvidence `json:"snippets,omitempty"`
	Error        string                             `json:"error,omitempty"`
}

// ReviewExternalDocSnippetEvidence は external_doc evidence の引用可能な bounded snippet。
type ReviewExternalDocSnippetEvidence struct {
	SnippetID   string `json:"snippet_id"`
	Content     string `json:"content"`
	ContentHash string `json:"content_hash"`
	Truncated   bool   `json:"truncated,omitempty"`
}
