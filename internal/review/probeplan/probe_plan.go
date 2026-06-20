package probeplan

import (
	"github.com/susugadx/xelyon-cli/internal/review/domain"
	"github.com/susugadx/xelyon-cli/internal/review/report"
)

const (
	// ReviewProbePlanSchemaVersionV1 は旧 probe plan schema v1 の識別子。
	ReviewProbePlanSchemaVersionV1 = "review_probe_plan.v1"
	// ReviewProbePlanSchemaVersionV2 は LLM が返す probe plan schema v2 の識別子。
	ReviewProbePlanSchemaVersionV2 = "review_probe_plan.v2"

	// MaxReviewProbePlanProbes は 1 plan が持てる probe 数の上限。
	MaxReviewProbePlanProbes = 8
	// MaxReviewProbePlanCommands は 1 probe が持てる command 数の上限。
	MaxReviewProbePlanCommands = 10
	// MaxReviewProbePlanFiles は 1 probe が持てる generated file 数の上限。
	MaxReviewProbePlanFiles = 20
	// MaxReviewProbePlanFileContentBytes は plan schema 上の generated file content の byte 長上限。
	MaxReviewProbePlanFileContentBytes = 64 * 1024
	// MaxReviewProbePlanTotalFileContentBytes は 1 probe が持てる generated file content 合計の byte 長上限。
	MaxReviewProbePlanTotalFileContentBytes = 256 * 1024
	// MaxReviewProbePlanPurposeBytes は probe purpose の byte 長上限。
	MaxReviewProbePlanPurposeBytes = 512
	// MaxReviewProbePlanTimeoutSeconds は timeout_seconds の上限。
	MaxReviewProbePlanTimeoutSeconds = 300
	// MaxReviewProbePlanMaxOutputBytes は max_output_bytes の上限。
	MaxReviewProbePlanMaxOutputBytes = 1024 * 1024
)

// ReviewProbePlan は LLM 出力 JSON として decode する probe 計画 DTO。
// ProbeRunner が直接扱う runtime 契約ではなく、検証後に ReviewProbeRequest へ変換する。
type ReviewProbePlan struct {
	SchemaVersion         string                     `json:"schema_version"`
	TargetKind            domain.TargetKind          `json:"target_kind"`
	Summary               string                     `json:"summary,omitempty"`
	ImpactSurfaces        []ReviewProbeImpactSurface `json:"impact_surfaces"`
	CandidateRisks        []ReviewProbeCandidateRisk `json:"candidate_risks"`
	Probes                []ReviewPlannedProbe       `json:"probes"`
	NoCandidateRiskReason string                     `json:"no_candidate_risk_reason,omitempty"`
	NoProbeReason         string                     `json:"no_probe_reason,omitempty"`
}

// ReviewProbeImpactSurface は Pass1 で material と判断した影響面を表す。
type ReviewProbeImpactSurface struct {
	ID              string                           `json:"id"`
	Summary         string                           `json:"summary"`
	Category        ReviewProbeImpactSurfaceCategory `json:"category"`
	EvidenceSummary string                           `json:"evidence_summary,omitempty"`
	EvidenceRefs    []report.ReviewEvidenceRef       `json:"evidence_refs,omitempty"`
	Status          ReviewProbeImpactSurfaceStatus   `json:"status"`
	Reason          string                           `json:"reason"`
}

// ReviewProbeCandidateRisk は impact surface に紐づく候補リスクを表す。
type ReviewProbeCandidateRisk struct {
	ID                   string                         `json:"id"`
	Summary              string                         `json:"summary"`
	Severity             report.ReviewGroupSeverity     `json:"severity"`
	SurfaceIDs           []string                       `json:"surface_ids"`
	EvidenceSummary      string                         `json:"evidence_summary,omitempty"`
	EvidenceRefs         []report.ReviewEvidenceRef     `json:"evidence_refs,omitempty"`
	VerificationStrategy string                         `json:"verification_strategy"`
	Status               ReviewProbeCandidateRiskStatus `json:"status"`
}

// ReviewProbeImpactSurfaceCategory は Pass1 scope analysis の surface 種別。
type ReviewProbeImpactSurfaceCategory string

const (
	ReviewProbeImpactSurfaceChangedFile    ReviewProbeImpactSurfaceCategory = "changed_file"
	ReviewProbeImpactSurfaceCaller         ReviewProbeImpactSurfaceCategory = "caller"
	ReviewProbeImpactSurfaceTest           ReviewProbeImpactSurfaceCategory = "test"
	ReviewProbeImpactSurfaceCLI            ReviewProbeImpactSurfaceCategory = "cli"
	ReviewProbeImpactSurfaceTUI            ReviewProbeImpactSurfaceCategory = "tui"
	ReviewProbeImpactSurfaceConfig         ReviewProbeImpactSurfaceCategory = "config"
	ReviewProbeImpactSurfaceValidator      ReviewProbeImpactSurfaceCategory = "validator"
	ReviewProbeImpactSurfacePromptContract ReviewProbeImpactSurfaceCategory = "prompt_contract"
	ReviewProbeImpactSurfaceJSONSchema     ReviewProbeImpactSurfaceCategory = "json_schema"
	ReviewProbeImpactSurfaceSandbox        ReviewProbeImpactSurfaceCategory = "sandbox"
	ReviewProbeImpactSurfaceTimeout        ReviewProbeImpactSurfaceCategory = "timeout"
	ReviewProbeImpactSurfacePathValidation ReviewProbeImpactSurfaceCategory = "path_validation"
	ReviewProbeImpactSurfaceErrorHandling  ReviewProbeImpactSurfaceCategory = "error_handling"
	ReviewProbeImpactSurfacePersistence    ReviewProbeImpactSurfaceCategory = "persistence"
	ReviewProbeImpactSurfaceCompatibility  ReviewProbeImpactSurfaceCategory = "compatibility"
)

var reviewProbeImpactSurfaceCategories = []ReviewProbeImpactSurfaceCategory{
	ReviewProbeImpactSurfaceChangedFile,
	ReviewProbeImpactSurfaceCaller,
	ReviewProbeImpactSurfaceTest,
	ReviewProbeImpactSurfaceCLI,
	ReviewProbeImpactSurfaceTUI,
	ReviewProbeImpactSurfaceConfig,
	ReviewProbeImpactSurfaceValidator,
	ReviewProbeImpactSurfacePromptContract,
	ReviewProbeImpactSurfaceJSONSchema,
	ReviewProbeImpactSurfaceSandbox,
	ReviewProbeImpactSurfaceTimeout,
	ReviewProbeImpactSurfacePathValidation,
	ReviewProbeImpactSurfaceErrorHandling,
	ReviewProbeImpactSurfacePersistence,
	ReviewProbeImpactSurfaceCompatibility,
}

// KnownReviewProbeImpactSurfaceCategories は probe plan で許可する impact surface category 一覧を返す。
func KnownReviewProbeImpactSurfaceCategories() []ReviewProbeImpactSurfaceCategory {
	return append([]ReviewProbeImpactSurfaceCategory(nil), reviewProbeImpactSurfaceCategories...)
}

// ReviewProbeImpactSurfaceStatus は impact surface の Pass1 確認状態。
type ReviewProbeImpactSurfaceStatus string

const (
	ReviewProbeImpactSurfaceChecked    ReviewProbeImpactSurfaceStatus = "checked"
	ReviewProbeImpactSurfaceNeedsProbe ReviewProbeImpactSurfaceStatus = "needs_probe"
	ReviewProbeImpactSurfaceUnverified ReviewProbeImpactSurfaceStatus = "unverified"
)

var reviewProbeImpactSurfaceStatuses = []ReviewProbeImpactSurfaceStatus{
	ReviewProbeImpactSurfaceChecked,
	ReviewProbeImpactSurfaceNeedsProbe,
	ReviewProbeImpactSurfaceUnverified,
}

// KnownReviewProbeImpactSurfaceStatuses は probe plan で許可する impact surface status 一覧を返す。
func KnownReviewProbeImpactSurfaceStatuses() []ReviewProbeImpactSurfaceStatus {
	return append([]ReviewProbeImpactSurfaceStatus(nil), reviewProbeImpactSurfaceStatuses...)
}

// ReviewProbeCandidateRiskStatus は candidate risk の Pass1 確認状態。
type ReviewProbeCandidateRiskStatus string

const (
	ReviewProbeCandidateRiskNeedsProbe        ReviewProbeCandidateRiskStatus = "needs_probe"
	ReviewProbeCandidateRiskCheckedByEvidence ReviewProbeCandidateRiskStatus = "checked_by_evidence"
	ReviewProbeCandidateRiskUnverified        ReviewProbeCandidateRiskStatus = "unverified"
)

var reviewProbeCandidateRiskStatuses = []ReviewProbeCandidateRiskStatus{
	ReviewProbeCandidateRiskNeedsProbe,
	ReviewProbeCandidateRiskCheckedByEvidence,
	ReviewProbeCandidateRiskUnverified,
}

// KnownReviewProbeCandidateRiskStatuses は probe plan で許可する candidate risk status 一覧を返す。
func KnownReviewProbeCandidateRiskStatuses() []ReviewProbeCandidateRiskStatus {
	return append([]ReviewProbeCandidateRiskStatus(nil), reviewProbeCandidateRiskStatuses...)
}

// ReviewPlannedProbe は LLM plan 内の 1 probe 定義を表す。
type ReviewPlannedProbe struct {
	ID             string                      `json:"id"`
	SurfaceIDs     []string                    `json:"surface_ids,omitempty"`
	RiskIDs        []string                    `json:"risk_ids,omitempty"`
	Purpose        string                      `json:"purpose"`
	Mode           domain.ReviewProbeMode      `json:"mode"`
	Commands       []ReviewPlannedProbeCommand `json:"commands,omitempty"`
	Files          []ReviewPlannedProbeFile    `json:"files,omitempty"`
	TimeoutSeconds int                         `json:"timeout_seconds,omitempty"`
	MaxOutputBytes int64                       `json:"max_output_bytes,omitempty"`
}

// ReviewPlannedProbeCommand は plan schema 上の command DTO。
type ReviewPlannedProbeCommand struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	WorkDir string   `json:"work_dir,omitempty"`
}

// ReviewPlannedProbeFile は plan schema 上の generated file DTO。
type ReviewPlannedProbeFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
