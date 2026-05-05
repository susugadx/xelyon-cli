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

	ChangedFiles           []ReviewChangedFile
	UntrackedFiles         []ReviewUntrackedFile
	UntrackedListTruncated bool
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
	MaxCommandOutputBytes  int64
	MaxUntrackedFileBytes  int64
	MaxRuleFileBytes       int64
	MaxTotalUntrackedBytes int64
	MaxUntrackedFiles      int
	CommandTimeout         time.Duration
}
