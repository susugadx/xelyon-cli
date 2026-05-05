package review

const reviewEvidenceRepoRootPathDisplay = "<repo_root>"

// ReviewEvidenceModelInput は ReviewEvidenceBundle を LLM 入力向けに正規化した DTO。
type ReviewEvidenceModelInput struct {
	TargetKind      TargetKind                         `json:"target_kind"`
	RepoRoot        string                             `json:"repo_root"`
	CWDDisplay      string                             `json:"cwd_display"`
	GitStatusShort  ReviewEvidenceTextBlock            `json:"git_status_short"`
	ChangeInventory ReviewEvidenceChangeInventoryInput `json:"change_inventory"`
	ChangedFiles    []ReviewEvidenceChangedFileInput   `json:"changed_files"`
	RuleFiles       []ReviewEvidenceRuleFileInput      `json:"rule_files"`
	Diffs           []ReviewEvidenceDiffInput          `json:"diffs"`
	UntrackedFiles  []ReviewEvidenceUntrackedFileInput `json:"untracked_files"`
	Limits          ReviewEvidenceLimitsInput          `json:"limits"`
	TruncationFlags ReviewEvidenceTruncationFlagsInput `json:"truncation_flags"`
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
	MaxCommandOutputBytes  int64 `json:"max_command_output_bytes"`
	MaxUntrackedFileBytes  int64 `json:"max_untracked_file_bytes"`
	MaxRuleFileBytes       int64 `json:"max_rule_file_bytes"`
	MaxTotalUntrackedBytes int64 `json:"max_total_untracked_bytes"`
	MaxUntrackedFiles      int   `json:"max_untracked_files"`
	CommandTimeoutMS       int64 `json:"command_timeout_ms"`
}

// ReviewEvidenceTruncationFlagsInput は bundle 全体の truncation 状態を固定順序で表す。
type ReviewEvidenceTruncationFlagsInput struct {
	StatusShort        bool                                `json:"status_short"`
	Diffs              []ReviewEvidenceDiffTruncationInput `json:"diffs"`
	UntrackedList      bool                                `json:"untracked_list"`
	UntrackedSnapshots bool                                `json:"untracked_snapshots"`
	UntrackedFiles     []ReviewEvidencePathTruncationInput `json:"untracked_files"`
	RuleFiles          []ReviewEvidencePathTruncationInput `json:"rule_files"`
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

// BuildReviewEvidenceModelInput は ReviewEvidenceBundle を LLM 入力 DTO に変換する。
func BuildReviewEvidenceModelInput(bundle ReviewEvidenceBundle) ReviewEvidenceModelInput {
	repoRoot := bundle.RepoRoot

	return ReviewEvidenceModelInput{
		TargetKind: bundle.TargetKind,
		RepoRoot:   reviewEvidenceRepoRootPathDisplay,
		CWDDisplay: formatReviewEvidencePathDisplay(repoRoot, bundle.CWD),
		GitStatusShort: ReviewEvidenceTextBlock{
			Content:   bundle.StatusShort,
			Truncated: bundle.StatusShortTruncated,
		},
		ChangeInventory: buildReviewEvidenceChangeInventoryInput(repoRoot, bundle.Inventory),
		ChangedFiles:    buildReviewEvidenceChangedFileInputs(repoRoot, bundle.ChangedFiles),
		RuleFiles:       buildReviewEvidenceRuleFileInputs(repoRoot, bundle.RuleFiles),
		Diffs:           buildReviewEvidenceDiffInputs(bundle.Diffs),
		UntrackedFiles:  buildReviewEvidenceUntrackedFileInputs(repoRoot, bundle.UntrackedFiles),
		Limits:          buildReviewEvidenceLimitsInput(bundle.Limits),
		TruncationFlags: buildReviewEvidenceTruncationFlagsInput(repoRoot, bundle),
	}
}

func buildReviewEvidenceChangeInventoryInput(repoRoot string, inventory ReviewChangeInventory) ReviewEvidenceChangeInventoryInput {
	return ReviewEvidenceChangeInventoryInput{
		Generated:    formatReviewEvidencePathDisplays(repoRoot, inventory.Generated),
		Tests:        formatReviewEvidencePathDisplays(repoRoot, inventory.Tests),
		Docs:         formatReviewEvidencePathDisplays(repoRoot, inventory.Docs),
		Config:       formatReviewEvidencePathDisplays(repoRoot, inventory.Config),
		Production:   formatReviewEvidencePathDisplays(repoRoot, inventory.Production),
		NewFiles:     formatReviewEvidencePathDisplays(repoRoot, inventory.NewFiles),
		DeletedFiles: formatReviewEvidencePathDisplays(repoRoot, inventory.DeletedFiles),
		RenamedFiles: formatReviewEvidencePathDisplays(repoRoot, inventory.RenamedFiles),
		Untracked:    formatReviewEvidencePathDisplays(repoRoot, inventory.Untracked),
	}
}

func buildReviewEvidenceChangedFileInputs(repoRoot string, files []ReviewChangedFile) []ReviewEvidenceChangedFileInput {
	result := make([]ReviewEvidenceChangedFileInput, 0, len(files))
	for _, file := range files {
		result = append(result, ReviewEvidenceChangedFileInput{
			Path:     formatReviewEvidencePathDisplay(repoRoot, file.Path),
			OldPath:  formatReviewEvidenceOptionalPathDisplay(repoRoot, file.OldPath),
			Status:   file.Status,
			Staged:   file.Staged,
			Unstaged: file.Unstaged,
		})
	}
	return result
}

func buildReviewEvidenceRuleFileInputs(repoRoot string, files []ReviewRuleFileEvidence) []ReviewEvidenceRuleFileInput {
	result := make([]ReviewEvidenceRuleFileInput, 0, len(files))
	for _, file := range files {
		result = append(result, ReviewEvidenceRuleFileInput{
			Path:      formatReviewEvidencePathDisplay(repoRoot, file.Path),
			Content:   file.Content,
			Truncated: file.Truncated,
			SizeBytes: file.SizeBytes,
		})
	}
	return result
}

func buildReviewEvidenceDiffInputs(diffs []ReviewDiffEvidence) []ReviewEvidenceDiffInput {
	result := make([]ReviewEvidenceDiffInput, 0, len(diffs))
	for _, diff := range diffs {
		result = append(result, ReviewEvidenceDiffInput{
			Source: diff.Source,
			Stat: ReviewEvidenceTextBlock{
				Content:   diff.Stat,
				Truncated: diff.StatTruncated,
			},
			NameStatus: ReviewEvidenceTextBlock{
				Content:   diff.NameStatus,
				Truncated: diff.NameStatusTruncated,
			},
			Diff: ReviewEvidenceTextBlock{
				Content:   diff.Diff,
				Truncated: diff.DiffTruncated,
			},
		})
	}
	return result
}

func buildReviewEvidenceUntrackedFileInputs(repoRoot string, files []ReviewUntrackedFile) []ReviewEvidenceUntrackedFileInput {
	result := make([]ReviewEvidenceUntrackedFileInput, 0, len(files))
	for _, file := range files {
		result = append(result, ReviewEvidenceUntrackedFileInput{
			Path:       formatReviewEvidencePathDisplay(repoRoot, file.Path),
			Symlink:    file.Symlink,
			LinkTarget: formatReviewEvidenceSymlinkTargetDisplay(repoRoot, file),
			Snapshot:   file.Snapshot,
			Binary:     file.Binary,
			Truncated:  file.Truncated,
			SizeBytes:  file.SizeBytes,
			ReadBytes:  file.ReadBytes,
		})
	}
	return result
}

func buildReviewEvidenceLimitsInput(limits ReviewEvidenceLimits) ReviewEvidenceLimitsInput {
	return ReviewEvidenceLimitsInput{
		MaxCommandOutputBytes:  limits.MaxCommandOutputBytes,
		MaxUntrackedFileBytes:  limits.MaxUntrackedFileBytes,
		MaxRuleFileBytes:       limits.MaxRuleFileBytes,
		MaxTotalUntrackedBytes: limits.MaxTotalUntrackedBytes,
		MaxUntrackedFiles:      limits.MaxUntrackedFiles,
		CommandTimeoutMS:       limits.CommandTimeout.Milliseconds(),
	}
}

func buildReviewEvidenceTruncationFlagsInput(repoRoot string, bundle ReviewEvidenceBundle) ReviewEvidenceTruncationFlagsInput {
	return ReviewEvidenceTruncationFlagsInput{
		StatusShort:        bundle.StatusShortTruncated,
		Diffs:              buildReviewEvidenceDiffTruncationInputs(bundle.Diffs),
		UntrackedList:      bundle.UntrackedListTruncated,
		UntrackedSnapshots: bundle.UntrackedSnapshotsTruncated,
		UntrackedFiles:     buildReviewEvidenceUntrackedTruncationInputs(repoRoot, bundle.UntrackedFiles),
		RuleFiles:          buildReviewEvidenceRuleFileTruncationInputs(repoRoot, bundle.RuleFiles),
	}
}

func buildReviewEvidenceDiffTruncationInputs(diffs []ReviewDiffEvidence) []ReviewEvidenceDiffTruncationInput {
	result := make([]ReviewEvidenceDiffTruncationInput, 0, len(diffs))
	for _, diff := range diffs {
		result = append(result, ReviewEvidenceDiffTruncationInput{
			Source:     diff.Source,
			Stat:       diff.StatTruncated,
			NameStatus: diff.NameStatusTruncated,
			Diff:       diff.DiffTruncated,
		})
	}
	return result
}

func buildReviewEvidenceUntrackedTruncationInputs(repoRoot string, files []ReviewUntrackedFile) []ReviewEvidencePathTruncationInput {
	result := make([]ReviewEvidencePathTruncationInput, 0, len(files))
	for _, file := range files {
		result = append(result, ReviewEvidencePathTruncationInput{
			Path:      formatReviewEvidencePathDisplay(repoRoot, file.Path),
			Truncated: file.Truncated,
		})
	}
	return result
}

func buildReviewEvidenceRuleFileTruncationInputs(repoRoot string, files []ReviewRuleFileEvidence) []ReviewEvidencePathTruncationInput {
	result := make([]ReviewEvidencePathTruncationInput, 0, len(files))
	for _, file := range files {
		result = append(result, ReviewEvidencePathTruncationInput{
			Path:      formatReviewEvidencePathDisplay(repoRoot, file.Path),
			Truncated: file.Truncated,
		})
	}
	return result
}
