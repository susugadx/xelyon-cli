package evidence

import "github.com/susugadx/xelyon-cli/internal/review/externaldoc"

const (
	// RepoRootPathDisplay は repo root を LLM 入力へ出すときの固定表現。
	RepoRootPathDisplay = "<repo_root>"

	reviewEvidenceRepoRootPathDisplay = RepoRootPathDisplay
)

// ReviewEvidenceModelInput は ReviewEvidenceBundle を LLM 入力向けに正規化した DTO。
// Evidence renderer は収集済み bundle の整形だけを担当し、evidence 収集は行わない。
type ReviewEvidenceModelInput struct {
	TargetKind          TargetKind                            `json:"target_kind"`
	RepoRoot            string                                `json:"repo_root"`
	CWDDisplay          string                                `json:"cwd_display"`
	GitStatusShort      ReviewEvidenceTextBlock               `json:"git_status_short"`
	ChangeInventory     ReviewEvidenceChangeInventoryInput    `json:"change_inventory"`
	ChangedFiles        []ReviewEvidenceChangedFileInput      `json:"changed_files"`
	ChangedFileContext  []ReviewEvidenceContextFileInput      `json:"changed_file_context"`
	RelatedContextFiles []ReviewEvidenceContextFileInput      `json:"related_context_files"`
	RelatedSearchHits   []ReviewEvidenceRelatedSearchHitInput `json:"related_search_hits"`
	GenericImpact       ReviewEvidenceGenericImpactInput      `json:"generic_impact_candidates"`
	WebSearchEvidence   ReviewWebSearchEvidence               `json:"web_search_evidence"`
	ExternalSupport     externaldoc.ExternalSupportSummary    `json:"external_support"`
	RuleFiles           []ReviewEvidenceRuleFileInput         `json:"rule_files"`
	Diffs               []ReviewEvidenceDiffInput             `json:"diffs"`
	UntrackedFiles      []ReviewEvidenceUntrackedFileInput    `json:"untracked_files"`
	Limits              ReviewEvidenceLimitsInput             `json:"limits"`
	TruncationFlags     ReviewEvidenceTruncationFlagsInput    `json:"truncation_flags"`
}

// ReviewEvidenceTextBlock は本文と truncation flag を組にした DTO。
type ReviewEvidenceTextBlock struct {
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
}

// ReviewEvidenceChangedFileInput は changed file の path 表示を正規化した DTO。
type ReviewEvidenceChangedFileInput struct {
	Path     string `json:"path"`
	OldPath  string `json:"old_path"`
	Status   string `json:"status"`
	Staged   bool   `json:"staged"`
	Unstaged bool   `json:"unstaged"`
}

// ReviewEvidenceContextFileInput は context file evidence を LLM 入力向けに表す。
type ReviewEvidenceContextFileInput struct {
	Path       string `json:"path"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	Truncated  bool   `json:"truncated"`
	Skipped    bool   `json:"skipped"`
	SkipReason string `json:"skip_reason"`
	SizeBytes  int64  `json:"size_bytes"`
	ReadBytes  int64  `json:"read_bytes"`
}

// ReviewEvidenceRelatedSearchHitInput は search hit evidence を LLM 入力向けに表す。
type ReviewEvidenceRelatedSearchHitInput struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
	Reason  string `json:"reason"`
}

// ReviewEvidenceGenericImpactInput は generic impact candidates の LLM 入力表現。
type ReviewEvidenceGenericImpactInput struct {
	Tokens     []string                                    `json:"tokens"`
	Candidates []ReviewEvidenceGenericImpactCandidateInput `json:"candidates"`
	Truncated  bool                                        `json:"truncated"`
}

// ReviewEvidenceGenericImpactCandidateInput は generic impact candidate の LLM 入力表現。
type ReviewEvidenceGenericImpactCandidateInput struct {
	Path    string `json:"path"`
	Role    string `json:"role"`
	Reason  string `json:"reason"`
	Token   string `json:"token"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

// ReviewEvidenceChangeInventoryInput は変更 surface の一覧を LLM 入力向けに表す。
type ReviewEvidenceChangeInventoryInput struct {
	Generated    []string `json:"generated"`
	Tests        []string `json:"tests"`
	Docs         []string `json:"docs"`
	Config       []string `json:"config"`
	Production   []string `json:"production"`
	NewFiles     []string `json:"new_files"`
	DeletedFiles []string `json:"deleted_files"`
	RenamedFiles []string `json:"renamed_files"`
	Untracked    []string `json:"untracked"`
}

// ReviewEvidenceRuleFileInput は rule file evidence を LLM 入力向けに表す。
type ReviewEvidenceRuleFileInput struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
	SizeBytes int64  `json:"size_bytes"`
}

// ReviewEvidenceDiffInput は diff evidence を LLM 入力向けに表す。
type ReviewEvidenceDiffInput struct {
	Source     string                  `json:"source"`
	Stat       ReviewEvidenceTextBlock `json:"stat"`
	NameStatus ReviewEvidenceTextBlock `json:"name_status"`
	Diff       ReviewEvidenceTextBlock `json:"diff"`
}

// ReviewEvidenceUntrackedFileInput は untracked snapshot を LLM 入力向けに表す。
type ReviewEvidenceUntrackedFileInput struct {
	Path       string `json:"path"`
	Symlink    bool   `json:"symlink"`
	LinkTarget string `json:"link_target"`
	Snapshot   string `json:"snapshot"`
	Binary     bool   `json:"binary"`
	Truncated  bool   `json:"truncated"`
	SizeBytes  int64  `json:"size_bytes"`
	ReadBytes  int64  `json:"read_bytes"`
}

// ReviewEvidenceLimitsInput は resource budget を JSON 安定表現にした DTO。
type ReviewEvidenceLimitsInput struct {
	MaxCommandOutputBytes      int64 `json:"max_command_output_bytes"`
	MaxUntrackedFileBytes      int64 `json:"max_untracked_file_bytes"`
	MaxRuleFileBytes           int64 `json:"max_rule_file_bytes"`
	MaxTotalUntrackedBytes     int64 `json:"max_total_untracked_bytes"`
	MaxUntrackedFiles          int   `json:"max_untracked_files"`
	MaxContextFileBytes        int64 `json:"max_context_file_bytes"`
	MaxTotalContextBytes       int64 `json:"max_total_context_bytes"`
	MaxContextFiles            int   `json:"max_context_files"`
	MaxRelatedSearchTerms      int   `json:"max_related_search_terms"`
	MaxRelatedSearchFiles      int   `json:"max_related_search_files"`
	MaxTotalRelatedSearchBytes int64 `json:"max_total_related_search_bytes"`
	MaxRelatedSearchFileBytes  int64 `json:"max_related_search_file_bytes"`
	MaxRelatedSearchHits       int   `json:"max_related_search_hits"`
	MaxSearchSnippetBytes      int64 `json:"max_search_snippet_bytes"`
	CommandTimeoutMS           int64 `json:"command_timeout_ms"`
}

// ReviewEvidenceTruncationFlagsInput は bundle 全体の truncation 状態を固定順序で表す。
type ReviewEvidenceTruncationFlagsInput struct {
	StatusShort         bool                                `json:"status_short"`
	Diffs               []ReviewEvidenceDiffTruncationInput `json:"diffs"`
	UntrackedList       bool                                `json:"untracked_list"`
	RelatedCandidates   bool                                `json:"related_candidates"`
	RelatedSearch       bool                                `json:"related_search"`
	WebSearchEvidence   bool                                `json:"web_search_evidence"`
	UntrackedSnapshots  bool                                `json:"untracked_snapshots"`
	UntrackedFiles      []ReviewEvidencePathTruncationInput `json:"untracked_files"`
	RuleFiles           []ReviewEvidencePathTruncationInput `json:"rule_files"`
	ChangedFileContext  []ReviewEvidencePathTruncationInput `json:"changed_file_context"`
	RelatedContextFiles []ReviewEvidencePathTruncationInput `json:"related_context_files"`
}

// ReviewEvidenceDiffTruncationInput は diff ごとの truncation 状態を表す。
type ReviewEvidenceDiffTruncationInput struct {
	Source     string `json:"source"`
	Stat       bool   `json:"stat"`
	NameStatus bool   `json:"name_status"`
	Diff       bool   `json:"diff"`
}

// ReviewEvidencePathTruncationInput は path 付き evidence の truncation 状態を表す。
type ReviewEvidencePathTruncationInput struct {
	Path      string `json:"path"`
	Truncated bool   `json:"truncated"`
}
