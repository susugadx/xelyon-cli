package review

import "time"

const (
	// ReviewReportSchemaVersionV1 は `/review` report schema v1 の識別子。
	ReviewReportSchemaVersionV1 = "review_report.v1"
	// ReviewReportSkeletonBlockedSummary は skeleton report の blocked reason 既定文。
	ReviewReportSkeletonBlockedSummary = "Review report has not been finalized."
)

// ReviewVerdict は review report の最終判定。
type ReviewVerdict string

const (
	ReviewVerdictClean       ReviewVerdict = "clean"
	ReviewVerdictHasFindings ReviewVerdict = "has_findings"
	ReviewVerdictBlocked     ReviewVerdict = "blocked"
)

// ReviewVerificationStatus は検証状態の明示表現。
type ReviewVerificationStatus string

const (
	ReviewVerificationVerified              ReviewVerificationStatus = "verified"
	ReviewVerificationPartiallyVerified     ReviewVerificationStatus = "partially_verified"
	ReviewVerificationUnverified            ReviewVerificationStatus = "unverified"
	ReviewVerificationNotApplicable         ReviewVerificationStatus = "not_applicable"
	ReviewVerificationBlockedOrInconclusive ReviewVerificationStatus = "blocked_or_inconclusive"
)

// ReviewGroupSeverity は root-cause group 単位の重要度。
type ReviewGroupSeverity string

const (
	ReviewGroupSeverityCritical ReviewGroupSeverity = "critical"
	ReviewGroupSeverityHigh     ReviewGroupSeverity = "high"
	ReviewGroupSeverityMedium   ReviewGroupSeverity = "medium"
	ReviewGroupSeverityLow      ReviewGroupSeverity = "low"
	ReviewGroupSeverityInfo     ReviewGroupSeverity = "info"
)

// ReviewReport は `/review` の構造化結果を表す。
type ReviewReport struct {
	SchemaVersion             string                   `json:"schema_version"`
	TargetKind                TargetKind               `json:"target_kind"`
	CustomInstructions        string                   `json:"custom_instructions,omitempty"`
	GeneratedAt               time.Time                `json:"generated_at"`
	OverallVerificationStatus ReviewVerificationStatus `json:"overall_verification_status"`
	Verdict                   ReviewVerdict            `json:"verdict"`
	Summary                   string                   `json:"summary,omitempty"`
	RootCauseGroups           []ReviewRootCauseGroup   `json:"root_cause_groups,omitempty"`
	ProbeSummaries            []ReviewProbeSummary     `json:"probe_summaries,omitempty"`
	CheckedSurfaces           []ReviewSurfaceCoverage  `json:"checked_surfaces,omitempty"`
	UnverifiedSurfaces        []ReviewSurfaceCoverage  `json:"unverified_surfaces,omitempty"`
	ResidualRisks             []ReviewResidualRisk     `json:"residual_risks,omitempty"`
}

// ReviewRootCauseGroup は同一根本原因に紐づく finding 群をまとめる。
type ReviewRootCauseGroup struct {
	ID                 string                   `json:"id"`
	Title              string                   `json:"title"`
	Summary            string                   `json:"summary,omitempty"`
	Severity           ReviewGroupSeverity      `json:"severity"`
	VerificationStatus ReviewVerificationStatus `json:"verification_status"`
	FixStrategy        string                   `json:"fix_strategy,omitempty"`
	DoNotFixBy         []string                 `json:"do_not_fix_by,omitempty"`
	VerificationPlan   []string                 `json:"verification_plan,omitempty"`
	Findings           []ReviewFinding          `json:"findings,omitempty"`
	CheckedSurfaces    []ReviewSurfaceCoverage  `json:"checked_surfaces,omitempty"`
	UnverifiedSurfaces []ReviewSurfaceCoverage  `json:"unverified_surfaces,omitempty"`
	ResidualRisks      []ReviewResidualRisk     `json:"residual_risks,omitempty"`
}

// ReviewFinding は root-cause group 内の個別所見を表す。
type ReviewFinding struct {
	ID                 string                  `json:"id,omitempty"`
	Title              string                  `json:"title"`
	Summary            string                  `json:"summary,omitempty"`
	EvidenceRefs       []ReviewEvidenceRef     `json:"evidence_refs,omitempty"`
	CheckedSurfaces    []ReviewSurfaceCoverage `json:"checked_surfaces,omitempty"`
	UnverifiedSurfaces []ReviewSurfaceCoverage `json:"unverified_surfaces,omitempty"`
	ResidualRisks      []ReviewResidualRisk    `json:"residual_risks,omitempty"`
}

// ReviewEvidenceRef は finding/surface/risk を支える根拠参照を表す。
type ReviewEvidenceRef struct {
	Kind         string `json:"kind"`
	Summary      string `json:"summary,omitempty"`
	ProbeID      string `json:"probe_id,omitempty"`
	CommandIndex *int   `json:"command_index,omitempty"`
	Path         string `json:"path,omitempty"`
	Line         int    `json:"line,omitempty"`
	Snippet      string `json:"snippet,omitempty"`
}

const (
	ReviewEvidenceKindProbeCommand = "probe_command"
	ReviewEvidenceKindProbe        = "probe"
	ReviewEvidenceKindFile         = "file"
	ReviewEvidenceKindDiff         = "diff"
	ReviewEvidenceKindGitStatus    = "git_status"
	ReviewEvidenceKindRuleFile     = "rule_file"
)

// ReviewCommandIndex は command index の明示値を返す。
func ReviewCommandIndex(i int) *int {
	return &i
}

// ReviewSurfaceCoverage は確認済み/未確認の surface を構造化して保持する。
type ReviewSurfaceCoverage struct {
	SurfaceID    string              `json:"surface_id"`
	Summary      string              `json:"summary,omitempty"`
	EvidenceRefs []ReviewEvidenceRef `json:"evidence_refs,omitempty"`
}

// ReviewResidualRisk は未解消リスクを表す。
type ReviewResidualRisk struct {
	ID                  string              `json:"id,omitempty"`
	Summary             string              `json:"summary"`
	SuggestedMitigation string              `json:"suggested_mitigation,omitempty"`
	EvidenceRefs        []ReviewEvidenceRef `json:"evidence_refs,omitempty"`
}

// ReviewProbeSummary は ReviewProbeResult から report 用に切り出した要約。
type ReviewProbeSummary struct {
	ProbeID         string                      `json:"probe_id"`
	Mode            ReviewProbeMode             `json:"mode"`
	Status          ReviewProbeStatus           `json:"status"`
	MutatedWorktree bool                        `json:"mutated_worktree,omitempty"`
	MutatedFiles    []string                    `json:"mutated_files,omitempty"`
	OutputTruncated bool                        `json:"output_truncated,omitempty"`
	Error           string                      `json:"error,omitempty"`
	Commands        []ReviewProbeCommandSummary `json:"commands,omitempty"`
}

// ReviewProbeCommandSummary は report 用の probe command 要約。
type ReviewProbeCommandSummary struct {
	Command         string            `json:"command"`
	Args            []string          `json:"args,omitempty"`
	WorkDir         string            `json:"work_dir,omitempty"`
	Status          ReviewProbeStatus `json:"status"`
	ExitCode        int               `json:"exit_code,omitempty"`
	OutputTruncated bool              `json:"output_truncated,omitempty"`
	Error           string            `json:"error,omitempty"`
	DurationMs      int64             `json:"duration_ms,omitempty"`
}
