package review

import "time"

const (
	// ReviewReportSchemaVersionV1 は `/review` report schema v1 の識別子。
	ReviewReportSchemaVersionV1 = "review_report.v1"
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
	CommandIndex int    `json:"command_index,omitempty"`
	Path         string `json:"path,omitempty"`
	Line         int    `json:"line,omitempty"`
	Snippet      string `json:"snippet,omitempty"`
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

// NewReviewReportSkeleton は schema v1 の最小 report 枠を構築する。
func NewReviewReportSkeleton(req ReviewRequest, generatedAt time.Time) ReviewReport {
	return ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV1,
		TargetKind:                req.TargetKind,
		CustomInstructions:        req.CustomInstructions,
		GeneratedAt:               generatedAt,
		OverallVerificationStatus: ReviewVerificationUnverified,
		RootCauseGroups:           make([]ReviewRootCauseGroup, 0),
		ProbeSummaries:            make([]ReviewProbeSummary, 0),
		CheckedSurfaces:           make([]ReviewSurfaceCoverage, 0),
		UnverifiedSurfaces:        make([]ReviewSurfaceCoverage, 0),
		ResidualRisks:             make([]ReviewResidualRisk, 0),
	}
}

// BuildReviewProbeSummary は probe 結果を report 用要約へ変換する。
func BuildReviewProbeSummary(result ReviewProbeResult) ReviewProbeSummary {
	summary := ReviewProbeSummary{
		ProbeID:         result.ID,
		Mode:            result.Mode,
		Status:          result.Status,
		MutatedWorktree: result.MutatedWorktree,
		MutatedFiles:    append([]string(nil), result.MutatedFiles...),
		OutputTruncated: result.OutputTruncated,
		Error:           result.Error,
		Commands:        make([]ReviewProbeCommandSummary, 0, len(result.CommandResults)),
	}
	for _, commandResult := range result.CommandResults {
		summary.Commands = append(summary.Commands, ReviewProbeCommandSummary{
			Command:         commandResult.Command,
			Args:            append([]string(nil), commandResult.Args...),
			WorkDir:         commandResult.WorkDir,
			Status:          commandResult.Status,
			ExitCode:        commandResult.ExitCode,
			OutputTruncated: commandResult.OutputTruncated,
			Error:           commandResult.Error,
			DurationMs:      commandResult.Duration.Milliseconds(),
		})
	}
	return summary
}

// BuildReviewProbeSummaries は複数 probe 結果をまとめて要約する。
func BuildReviewProbeSummaries(results []ReviewProbeResult) []ReviewProbeSummary {
	summaries := make([]ReviewProbeSummary, 0, len(results))
	for _, result := range results {
		summaries = append(summaries, BuildReviewProbeSummary(result))
	}
	return summaries
}
