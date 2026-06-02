package report

// CoverageIssueKind は deterministic coverage audit が検出した漏れの種類。
type CoverageIssueKind string

const (
	// CoverageIssueKindMissingImpactSurfaceCoverage は Pass1 impact surface が scope_coverage に無い状態を表す。
	CoverageIssueKindMissingImpactSurfaceCoverage CoverageIssueKind = "missing_impact_surface_coverage"
	// CoverageIssueKindMissingCandidateRiskCoverage は Pass1 candidate risk が scope_coverage に無い状態を表す。
	CoverageIssueKindMissingCandidateRiskCoverage CoverageIssueKind = "missing_candidate_risk_coverage"
	// CoverageIssueKindUnreflectedProbeOutcome は linked probe の非 pass outcome が report に反映されていない状態を表す。
	CoverageIssueKindUnreflectedProbeOutcome CoverageIssueKind = "unreflected_probe_outcome"
	// CoverageIssueKindUnreflectedExternalEvidence は Post-Pass1 external evidence が report に反映されていない状態を表す。
	CoverageIssueKindUnreflectedExternalEvidence CoverageIssueKind = "unreflected_external_evidence"
	// CoverageIssueKindUnsupportedExternalConfirmation は external support が弱いのに report が公式確認を断定している状態を表す。
	CoverageIssueKindUnsupportedExternalConfirmation CoverageIssueKind = "unsupported_external_confirmation"
)

// CoverageIssueSeverity は deterministic coverage audit issue の重要度。
type CoverageIssueSeverity string

const (
	// CoverageIssueSeverityHigh は report revision を強く要求する coverage issue。
	CoverageIssueSeverityHigh CoverageIssueSeverity = "high"
	// CoverageIssueSeverityMedium は report の再確認と明示的な分類を要求する coverage issue。
	CoverageIssueSeverityMedium CoverageIssueSeverity = "medium"
)

// CoverageExternalSupport は report audit に渡す外部 evidence の薄い品質 summary。
type CoverageExternalSupport struct {
	Level                                string
	DocCount                             int
	CitationCapableDocCount              int
	CitationCapableSnippetCount          int
	OfficialCandidateDocCount            int
	OfficialCandidateCitationCapableDocs int
	OfficialConfirmation                 bool
	Warnings                             []string
	Reasons                              []string
}

// CoverageExternalEvidenceDelta は Post-Pass1 で増えた外部 evidence 差分。
// zero value は Post-Pass1 で反映すべき差分がない状態として扱う。
type CoverageExternalEvidenceDelta struct {
	AddedQueryCount       int
	AddedFailedQueryCount int
	AddedNoResultCount    int
	AddedQueries          []string
	AddedFailedQueries    []string
	AddedNoResultQueries  []string
	AddedDocIDs           []string
	AddedDocURLs          []string
	EvidenceError         bool
	Inconclusive          bool
	Truncated             bool
	Warnings              []string
	Reasons               []string
}

// CoverageAuditInput は final report coverage audit の入力。
type CoverageAuditInput struct {
	Plan                      PlanScope
	Report                    ReviewReport
	TrustedProbeSummaries     []ReviewProbeSummary
	PostPass1ExternalEvidence CoverageExternalEvidenceDelta
	ExternalSupport           CoverageExternalSupport
}

// CoverageIssue は deterministic coverage audit が revision feedback へ渡す 1 件の issue。
type CoverageIssue struct {
	Kind                CoverageIssueKind
	Severity            CoverageIssueSeverity
	SurfaceIDs          []string
	RiskIDs             []string
	ProbeID             string
	EvidenceRefs        []ReviewEvidenceRef
	Summary             string
	RevisionInstruction string
}

// AuditReviewReportCoverage は Pass1 scope/probe/external evidence が final report に意味的に反映されたかを検査する。
func AuditReviewReportCoverage(input CoverageAuditInput) []CoverageIssue {
	var issues []CoverageIssue
	issues = append(issues, auditMissingScopeCoverage(input)...)
	issues = append(issues, auditNonPassingProbeCoverage(input)...)
	issues = append(issues, auditExternalEvidenceCoverage(input)...)
	return issues
}
