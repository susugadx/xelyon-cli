package report

const (
	// ReviewSaturationCheckSchemaVersionV1 は runner 内部の final report saturation check schema。
	ReviewSaturationCheckSchemaVersionV1 = "review_saturation_check.v1"
)

// ReviewSaturationStatus は final report が Pass1 scope を十分に処理したかを表す。
type ReviewSaturationStatus string

const (
	ReviewSaturationStatusSaturated     ReviewSaturationStatus = "saturated"
	ReviewSaturationStatusNeedsRevision ReviewSaturationStatus = "needs_revision"
	ReviewSaturationStatusBlocked       ReviewSaturationStatus = "blocked"
)

// ReviewSaturationCheck は runner 内部で使う final report 漏れ検査 DTO。
type ReviewSaturationCheck struct {
	SchemaVersion               string                                       `json:"schema_version"`
	Status                      ReviewSaturationStatus                       `json:"status"`
	CheckedSummary              string                                       `json:"checked_summary"`
	MissingSurfaceIDs           []string                                     `json:"missing_surface_ids,omitempty"`
	MissingRiskIDs              []string                                     `json:"missing_risk_ids,omitempty"`
	AdditionalFindingCandidates []ReviewSaturationAdditionalFindingCandidate `json:"additional_finding_candidates,omitempty"`
	RevisionInstructions        string                                       `json:"revision_instructions,omitempty"`
}

// ReviewSaturationAdditionalFindingCandidate は revision に渡す追加 finding 候補。
type ReviewSaturationAdditionalFindingCandidate struct {
	Summary      string              `json:"summary"`
	EvidenceRefs []ReviewEvidenceRef `json:"evidence_refs"`
	Reason       string              `json:"reason"`
}
