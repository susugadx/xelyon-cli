package review

import "time"

const (
	// ReviewReportSchemaVersionV1 は旧 `/review` report schema v1 の識別子。
	ReviewReportSchemaVersionV1 = "review_report.v1"
	// ReviewReportSchemaVersionV2 は `/review` report schema v2 の識別子。
	ReviewReportSchemaVersionV2 = "review_report.v2"
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

var reviewGroupSeverities = []ReviewGroupSeverity{
	ReviewGroupSeverityCritical,
	ReviewGroupSeverityHigh,
	ReviewGroupSeverityMedium,
	ReviewGroupSeverityLow,
	ReviewGroupSeverityInfo,
}

// ReviewReport は `/review` の最終 report schema を表す。
// decode/validate の契約は report 側が owner し、probe 実行や evidence 収集は扱わない。
type ReviewReport struct {
	SchemaVersion             string                       `json:"schema_version"`
	TargetKind                TargetKind                   `json:"target_kind"`
	CustomInstructions        string                       `json:"custom_instructions,omitempty"`
	GeneratedAt               time.Time                    `json:"generated_at"`
	OverallVerificationStatus ReviewVerificationStatus     `json:"overall_verification_status"`
	Verdict                   ReviewVerdict                `json:"verdict"`
	Summary                   string                       `json:"summary,omitempty"`
	RootCauseGroups           []ReviewRootCauseGroup       `json:"root_cause_groups,omitempty"`
	ProbeSummaries            []ReviewProbeSummary         `json:"probe_summaries,omitempty"`
	CheckedSurfaces           []ReviewSurfaceCoverage      `json:"checked_surfaces,omitempty"`
	UnverifiedSurfaces        []ReviewSurfaceCoverage      `json:"unverified_surfaces,omitempty"`
	ResidualRisks             []ReviewResidualRisk         `json:"residual_risks,omitempty"`
	ScopeCoverage             *ReviewReportScopeCoverage   `json:"scope_coverage,omitempty"`
	ComputedSummary           *ReviewReportComputedSummary `json:"computed_summary,omitempty"`
}

// ReviewReportComputedSummary は runner が validation 後に算出する派生 count を表す。
// LLM/model 出力の入力 schema ではない。
type ReviewReportComputedSummary struct {
	RootCauseGroupCount       int `json:"root_cause_group_count"`
	FindingCount              int `json:"finding_count"`
	CheckedSurfaceCount       int `json:"checked_surface_count"`
	FindingSurfaceCount       int `json:"finding_surface_count"`
	UnverifiedSurfaceCount    int `json:"unverified_surface_count"`
	ResidualSurfaceCount      int `json:"residual_surface_count"`
	CandidateRiskCount        int `json:"candidate_risk_count"`
	DismissedRiskCount        int `json:"dismissed_risk_count"`
	FindingRiskCount          int `json:"finding_risk_count"`
	UnverifiedRiskCount       int `json:"unverified_risk_count"`
	ResidualRiskCount         int `json:"residual_risk_count"`
	NewReportPassFindingCount int `json:"new_report_pass_finding_count"`
	ProbeCount                int `json:"probe_count"`
	PassedProbeCount          int `json:"passed_probe_count"`
	FailedProbeCount          int `json:"failed_probe_count"`
	TimedOutProbeCount        int `json:"timed_out_probe_count"`
	BlockedProbeCount         int `json:"blocked_probe_count"`
	MutatedWorktreeProbeCount int `json:"mutated_worktree_probe_count"`
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
	DocID        string `json:"doc_id,omitempty"`
	SnippetID    string `json:"snippet_id,omitempty"`
	URL          string `json:"url,omitempty"`
	FetchedAt    string `json:"fetched_at,omitempty"`
	ContentHash  string `json:"content_hash,omitempty"`
}

const (
	ReviewEvidenceKindProbeCommand = "probe_command"
	ReviewEvidenceKindProbe        = "probe"
	ReviewEvidenceKindFile         = "file"
	ReviewEvidenceKindDiff         = "diff"
	ReviewEvidenceKindGitStatus    = "git_status"
	ReviewEvidenceKindRuleFile     = "rule_file"
	ReviewEvidenceKindExternalDoc  = "external_doc"
)

var reviewEvidenceKinds = []string{
	ReviewEvidenceKindProbeCommand,
	ReviewEvidenceKindProbe,
	ReviewEvidenceKindFile,
	ReviewEvidenceKindDiff,
	ReviewEvidenceKindGitStatus,
	ReviewEvidenceKindRuleFile,
	ReviewEvidenceKindExternalDoc,
}

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

// ReviewReportScopeCoverage は Pass1 で列挙した scope を Pass2 がどう処理したかを表す。
type ReviewReportScopeCoverage struct {
	ReviewedImpactSurfaces    []ReviewReportImpactSurfaceCoverage `json:"reviewed_impact_surfaces,omitempty"`
	ReviewedCandidateRisks    []ReviewReportCandidateRiskCoverage `json:"reviewed_candidate_risks,omitempty"`
	NewFindingsFromReportPass []ReviewReportPassFindingCoverage   `json:"new_findings_from_report_pass,omitempty"`
}

// ReviewReportImpactSurfaceStatus は Pass2 での impact surface 処理結果。
type ReviewReportImpactSurfaceStatus string

const (
	ReviewReportImpactSurfaceChecked      ReviewReportImpactSurfaceStatus = "checked"
	ReviewReportImpactSurfaceFinding      ReviewReportImpactSurfaceStatus = "finding"
	ReviewReportImpactSurfaceUnverified   ReviewReportImpactSurfaceStatus = "unverified"
	ReviewReportImpactSurfaceResidualRisk ReviewReportImpactSurfaceStatus = "residual_risk"
)

var reviewReportImpactSurfaceStatuses = []ReviewReportImpactSurfaceStatus{
	ReviewReportImpactSurfaceChecked,
	ReviewReportImpactSurfaceFinding,
	ReviewReportImpactSurfaceUnverified,
	ReviewReportImpactSurfaceResidualRisk,
}

// ReviewReportCandidateRiskStatus は Pass2 での candidate risk 処理結果。
type ReviewReportCandidateRiskStatus string

const (
	ReviewReportCandidateRiskDismissed    ReviewReportCandidateRiskStatus = "dismissed"
	ReviewReportCandidateRiskFinding      ReviewReportCandidateRiskStatus = "finding"
	ReviewReportCandidateRiskUnverified   ReviewReportCandidateRiskStatus = "unverified"
	ReviewReportCandidateRiskResidualRisk ReviewReportCandidateRiskStatus = "residual_risk"
)

var reviewReportCandidateRiskStatuses = []ReviewReportCandidateRiskStatus{
	ReviewReportCandidateRiskDismissed,
	ReviewReportCandidateRiskFinding,
	ReviewReportCandidateRiskUnverified,
	ReviewReportCandidateRiskResidualRisk,
}

// ReviewReportImpactSurfaceCoverage は Pass1 impact surface 1 件の Pass2 処理結果。
type ReviewReportImpactSurfaceCoverage struct {
	SurfaceID    string                          `json:"surface_id"`
	Status       ReviewReportImpactSurfaceStatus `json:"status"`
	Summary      string                          `json:"summary,omitempty"`
	EvidenceRefs []ReviewEvidenceRef             `json:"evidence_refs,omitempty"`
	FindingIDs   []string                        `json:"finding_ids,omitempty"`
}

// ReviewReportCandidateRiskCoverage は Pass1 candidate risk 1 件の Pass2 処理結果。
type ReviewReportCandidateRiskCoverage struct {
	RiskID       string                          `json:"risk_id"`
	Status       ReviewReportCandidateRiskStatus `json:"status"`
	Summary      string                          `json:"summary,omitempty"`
	EvidenceRefs []ReviewEvidenceRef             `json:"evidence_refs,omitempty"`
	FindingIDs   []string                        `json:"finding_ids,omitempty"`
}

// ReviewReportPassFindingCoverage は Pass1 scope 外で Pass2 が新たに見つけた finding 接続。
type ReviewReportPassFindingCoverage struct {
	FindingIDs   []string            `json:"finding_ids,omitempty"`
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
