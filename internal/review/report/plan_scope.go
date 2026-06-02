package report

// PlanScope は Pass1 probe plan から report validation が参照する最小 scope。
// probe 実行 policy や command/file DTO は持たず、report 側の cross validation に必要な
// ID、scope status、probe linkage だけを保持する。
type PlanScope struct {
	ImpactSurfaces []PlanImpactSurface
	CandidateRisks []PlanCandidateRisk
	Probes         []PlanProbe
}

// PlanImpactSurface は report validation が参照する Pass1 impact surface の最小形。
type PlanImpactSurface struct {
	ID     string
	Status PlanImpactSurfaceStatus
}

// PlanImpactSurfaceStatus は report validation が参照する Pass1 impact surface status。
type PlanImpactSurfaceStatus string

const (
	// PlanImpactSurfaceChecked は Pass1 で evidence のみで確認済みだった surface を表す。
	PlanImpactSurfaceChecked PlanImpactSurfaceStatus = "checked"
	// PlanImpactSurfaceNeedsProbe は Pass1 で probe が必要だった surface を表す。
	PlanImpactSurfaceNeedsProbe PlanImpactSurfaceStatus = "needs_probe"
	// PlanImpactSurfaceUnverified は Pass1 で未検証だった surface を表す。
	PlanImpactSurfaceUnverified PlanImpactSurfaceStatus = "unverified"
)

// PlanCandidateRisk は report validation が参照する Pass1 candidate risk の最小形。
type PlanCandidateRisk struct {
	ID     string
	Status PlanCandidateRiskStatus
	// Severity は deterministic coverage audit の優先度判定だけで使う内部 metadata。
	Severity ReviewGroupSeverity
}

// PlanCandidateRiskStatus は report validation が参照する Pass1 candidate risk status。
type PlanCandidateRiskStatus string

const (
	// PlanCandidateRiskNeedsProbe は Pass1 で probe が必要だった candidate risk を表す。
	PlanCandidateRiskNeedsProbe PlanCandidateRiskStatus = "needs_probe"
	// PlanCandidateRiskCheckedByEvidence は Pass1 で evidence のみで確認済みだった candidate risk を表す。
	PlanCandidateRiskCheckedByEvidence PlanCandidateRiskStatus = "checked_by_evidence"
	// PlanCandidateRiskUnverified は Pass1 で未検証だった candidate risk を表す。
	PlanCandidateRiskUnverified PlanCandidateRiskStatus = "unverified"
)

// PlanProbe は report validation が参照する Pass1 probe linkage の最小形。
type PlanProbe struct {
	ID         string
	SurfaceIDs []string
	RiskIDs    []string
}
