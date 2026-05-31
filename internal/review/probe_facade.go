package review

import reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"

const (
	// ReviewProbePlanSchemaVersionV1 は旧 probe plan schema v1 の識別子。
	ReviewProbePlanSchemaVersionV1 = reviewprobe.ReviewProbePlanSchemaVersionV1
	// ReviewProbePlanSchemaVersionV2 は LLM が返す probe plan schema v2 の識別子。
	ReviewProbePlanSchemaVersionV2 = reviewprobe.ReviewProbePlanSchemaVersionV2

	// MaxReviewProbePlanProbes は 1 plan が持てる probe 数の上限。
	MaxReviewProbePlanProbes = reviewprobe.MaxReviewProbePlanProbes
	// MaxReviewProbePlanCommands は 1 probe が持てる command 数の上限。
	MaxReviewProbePlanCommands = reviewprobe.MaxReviewProbePlanCommands
	// MaxReviewProbePlanFiles は 1 probe が持てる generated file 数の上限。
	MaxReviewProbePlanFiles = reviewprobe.MaxReviewProbePlanFiles
	// MaxReviewProbePlanFileContentBytes は plan schema 上の generated file content の byte 長上限。
	MaxReviewProbePlanFileContentBytes = reviewprobe.MaxReviewProbePlanFileContentBytes
	// MaxReviewProbePlanTotalFileContentBytes は 1 probe が持てる generated file content 合計の byte 長上限。
	MaxReviewProbePlanTotalFileContentBytes = reviewprobe.MaxReviewProbePlanTotalFileContentBytes
	// MaxReviewProbePlanPurposeBytes は probe purpose の byte 長上限。
	MaxReviewProbePlanPurposeBytes = reviewprobe.MaxReviewProbePlanPurposeBytes
	// MaxReviewProbePlanTimeoutSeconds は timeout_seconds の上限。
	MaxReviewProbePlanTimeoutSeconds = reviewprobe.MaxReviewProbePlanTimeoutSeconds
	// MaxReviewProbePlanMaxOutputBytes は max_output_bytes の上限。
	MaxReviewProbePlanMaxOutputBytes = reviewprobe.MaxReviewProbePlanMaxOutputBytes

	reviewProbeHostReadOnlyTempPrefix = reviewprobe.ReviewProbeHostReadOnlyTempPrefix
	reviewProbeScratchTempPrefix      = reviewprobe.ReviewProbeScratchTempPrefix
	reviewProbeSandboxTempPrefix      = reviewprobe.ReviewProbeSandboxTempPrefix
)

// ReviewProbePlan は LLM 出力 JSON として decode する probe 計画 DTO。
type ReviewProbePlan = reviewprobe.ReviewProbePlan

// ReviewProbeImpactSurface は Pass1 で material と判断した影響面を表す。
type ReviewProbeImpactSurface = reviewprobe.ReviewProbeImpactSurface

// ReviewProbeCandidateRisk は impact surface に紐づく候補リスクを表す。
type ReviewProbeCandidateRisk = reviewprobe.ReviewProbeCandidateRisk

// ReviewProbeImpactSurfaceCategory は Pass1 scope analysis の surface 種別。
type ReviewProbeImpactSurfaceCategory = reviewprobe.ReviewProbeImpactSurfaceCategory

const (
	ReviewProbeImpactSurfaceChangedFile    = reviewprobe.ReviewProbeImpactSurfaceChangedFile
	ReviewProbeImpactSurfaceCaller         = reviewprobe.ReviewProbeImpactSurfaceCaller
	ReviewProbeImpactSurfaceTest           = reviewprobe.ReviewProbeImpactSurfaceTest
	ReviewProbeImpactSurfaceCLI            = reviewprobe.ReviewProbeImpactSurfaceCLI
	ReviewProbeImpactSurfaceTUI            = reviewprobe.ReviewProbeImpactSurfaceTUI
	ReviewProbeImpactSurfaceConfig         = reviewprobe.ReviewProbeImpactSurfaceConfig
	ReviewProbeImpactSurfaceValidator      = reviewprobe.ReviewProbeImpactSurfaceValidator
	ReviewProbeImpactSurfacePromptContract = reviewprobe.ReviewProbeImpactSurfacePromptContract
	ReviewProbeImpactSurfaceJSONSchema     = reviewprobe.ReviewProbeImpactSurfaceJSONSchema
	ReviewProbeImpactSurfaceSandbox        = reviewprobe.ReviewProbeImpactSurfaceSandbox
	ReviewProbeImpactSurfaceTimeout        = reviewprobe.ReviewProbeImpactSurfaceTimeout
	ReviewProbeImpactSurfacePathValidation = reviewprobe.ReviewProbeImpactSurfacePathValidation
	ReviewProbeImpactSurfaceErrorHandling  = reviewprobe.ReviewProbeImpactSurfaceErrorHandling
	ReviewProbeImpactSurfacePersistence    = reviewprobe.ReviewProbeImpactSurfacePersistence
	ReviewProbeImpactSurfaceCompatibility  = reviewprobe.ReviewProbeImpactSurfaceCompatibility
)

// ReviewProbeImpactSurfaceStatus は impact surface の Pass1 確認状態。
type ReviewProbeImpactSurfaceStatus = reviewprobe.ReviewProbeImpactSurfaceStatus

const (
	ReviewProbeImpactSurfaceChecked    = reviewprobe.ReviewProbeImpactSurfaceChecked
	ReviewProbeImpactSurfaceNeedsProbe = reviewprobe.ReviewProbeImpactSurfaceNeedsProbe
	ReviewProbeImpactSurfaceUnverified = reviewprobe.ReviewProbeImpactSurfaceUnverified
)

// ReviewProbeCandidateRiskStatus は candidate risk の Pass1 確認状態。
type ReviewProbeCandidateRiskStatus = reviewprobe.ReviewProbeCandidateRiskStatus

const (
	ReviewProbeCandidateRiskNeedsProbe        = reviewprobe.ReviewProbeCandidateRiskNeedsProbe
	ReviewProbeCandidateRiskCheckedByEvidence = reviewprobe.ReviewProbeCandidateRiskCheckedByEvidence
	ReviewProbeCandidateRiskUnverified        = reviewprobe.ReviewProbeCandidateRiskUnverified
)

// ReviewPlannedProbe は LLM plan 内の 1 probe 定義を表す。
type ReviewPlannedProbe = reviewprobe.ReviewPlannedProbe

// ReviewPlannedProbeCommand は plan schema 上の command DTO。
type ReviewPlannedProbeCommand = reviewprobe.ReviewPlannedProbeCommand

// ReviewPlannedProbeFile は plan schema 上の generated file DTO。
type ReviewPlannedProbeFile = reviewprobe.ReviewPlannedProbeFile

// ReviewProbeRequest は ProbeRunner.Run に渡す runtime 内部の検証実行要求を表す。
type ReviewProbeRequest = reviewprobe.ReviewProbeRequest

// ReviewProbeFile は probe 実行時に必要な一時ファイル定義を表す。
type ReviewProbeFile = reviewprobe.ReviewProbeFile

// ReviewProbeCommand は probe 内で実行する 1 コマンドを表す。
type ReviewProbeCommand = reviewprobe.ReviewProbeCommand

// ReviewProbeResult は probe 実行結果を表す。
type ReviewProbeResult = reviewprobe.ReviewProbeResult

// ReviewProbeCommandResult は単一コマンドの実行結果を表す。
type ReviewProbeCommandResult = reviewprobe.ReviewProbeCommandResult

// ProbeRunner は review probe 実行を担当する。
type ProbeRunner = reviewprobe.ProbeRunner

var (
	reviewProbeIsolatedTempRootPrefixes = reviewprobe.ReviewProbeIsolatedTempRootPrefixes()
)

var (
	// ErrUnsupportedReviewProbeMode は未知または未対応 mode の実行時に返す。
	ErrUnsupportedReviewProbeMode = reviewprobe.ErrUnsupportedReviewProbeMode
	// ErrHostReadOnlyBlocked は host_readonly policy で command 実行が拒否されたことを示す。
	ErrHostReadOnlyBlocked = reviewprobe.ErrHostReadOnlyBlocked
	// ErrHostReadOnlyOutsideRepoPath は repo root 外 path による拒否を示す。
	ErrHostReadOnlyOutsideRepoPath = reviewprobe.ErrHostReadOnlyOutsideRepoPath
)

// NewProbeRunner は repo root を基準に probe runner を構築する。
func NewProbeRunner(repoRoot string) *ProbeRunner {
	return reviewprobe.NewProbeRunner(repoRoot)
}

// DecodeReviewProbePlanJSON は strict JSON として ReviewProbePlan を decode して検証する。
func DecodeReviewProbePlanJSON(data []byte) (ReviewProbePlan, error) {
	return reviewprobe.DecodeReviewProbePlanJSON(data)
}

// ValidateReviewProbePlan は LLM probe plan schema v2 の構造契約を検証する。
func ValidateReviewProbePlan(plan ReviewProbePlan) error {
	return reviewprobe.ValidateReviewProbePlan(plan)
}

// BuildReviewProbeRequestsFromPlan は検証済み plan DTO を ProbeRunner 用 runtime request へ変換する。
func BuildReviewProbeRequestsFromPlan(plan ReviewProbePlan) ([]ReviewProbeRequest, error) {
	return reviewprobe.BuildReviewProbeRequestsFromPlan(plan)
}

func formatProbeCommand(command string, args []string) string {
	return reviewprobe.FormatProbeCommand(command, args)
}
