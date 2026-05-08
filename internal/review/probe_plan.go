package review

const (
	// ReviewProbePlanSchemaVersionV1 は LLM が返す probe plan schema v1 の識別子。
	ReviewProbePlanSchemaVersionV1 = "review_probe_plan.v1"

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
	SchemaVersion string               `json:"schema_version"`
	TargetKind    TargetKind           `json:"target_kind"`
	Summary       string               `json:"summary,omitempty"`
	Probes        []ReviewPlannedProbe `json:"probes"`
	NoProbeReason string               `json:"no_probe_reason,omitempty"`
}

// ReviewPlannedProbe は LLM plan 内の 1 probe 定義を表す。
type ReviewPlannedProbe struct {
	ID             string                      `json:"id"`
	Purpose        string                      `json:"purpose"`
	Mode           ReviewProbeMode             `json:"mode"`
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
