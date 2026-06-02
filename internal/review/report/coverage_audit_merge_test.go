package report

import (
	"strings"
	"testing"
)

func TestMergeCoverageIssuesIntoSaturationCheckDoesNotPromoteWeakExternalEvidenceToFindingCandidate(t *testing.T) {
	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: newPlanAwareCleanReportForValidationTest(),
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-post"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                       "weak",
			DocCount:                    1,
			CitationCapableDocCount:     1,
			CitationCapableSnippetCount: 1,
			OfficialConfirmation:        false,
		},
	})
	if len(issues) == 0 {
		t.Fatal("AuditReviewReportCoverage() issues = nil, want weak external evidence feedback issue")
	}

	merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)
	if merged.Status != ReviewSaturationStatusSaturated {
		t.Fatalf("merged Status = %q, want %q", merged.Status, ReviewSaturationStatusSaturated)
	}
	if len(merged.AdditionalFindingCandidates) != 0 {
		t.Fatalf("AdditionalFindingCandidates = %#v, want none for weak external evidence alone", merged.AdditionalFindingCandidates)
	}
	if strings.TrimSpace(merged.RevisionInstructions) != "" {
		t.Fatalf("RevisionInstructions = %q, want none for medium-only saturated check", merged.RevisionInstructions)
	}
}

func TestMergeCoverageIssuesIntoSaturationCheckAddsDeterministicRevisionFeedback(t *testing.T) {
	issues := []CoverageIssue{
		{
			Kind:                CoverageIssueKindMissingImpactSurfaceCoverage,
			Severity:            CoverageIssueSeverityHigh,
			SurfaceIDs:          []string{"surface-1"},
			Summary:             "surface missing",
			RevisionInstruction: "Add surface-1 to scope_coverage.",
		},
		{
			Kind:                CoverageIssueKindUnreflectedProbeOutcome,
			Severity:            CoverageIssueSeverityHigh,
			RiskIDs:             []string{"risk-1"},
			ProbeID:             "probe-1",
			EvidenceRefs:        []ReviewEvidenceRef{{Kind: ReviewEvidenceKindProbe, ProbeID: "probe-1"}},
			Summary:             "probe outcome was ignored",
			RevisionInstruction: "Reflect probe-1 outcome.",
		},
	}

	merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)

	if merged.Status != ReviewSaturationStatusNeedsRevision {
		t.Fatalf("Status = %q, want %q", merged.Status, ReviewSaturationStatusNeedsRevision)
	}
	if got, want := strings.Join(merged.MissingSurfaceIDs, ","), "surface-1"; got != want {
		t.Fatalf("MissingSurfaceIDs = %q, want %q", got, want)
	}
	if got, want := strings.Join(merged.MissingRiskIDs, ","), "risk-1"; got != want {
		t.Fatalf("MissingRiskIDs = %q, want %q", got, want)
	}
	if len(merged.AdditionalFindingCandidates) != 1 {
		t.Fatalf("AdditionalFindingCandidates = %#v, want one probe-backed candidate", merged.AdditionalFindingCandidates)
	}
	for _, want := range []string{
		"Deterministic coverage audit requires revision",
		string(CoverageIssueKindMissingImpactSurfaceCoverage),
		string(CoverageIssueKindUnreflectedProbeOutcome),
		"Reflect probe-1 outcome",
	} {
		if !strings.Contains(merged.RevisionInstructions, want) {
			t.Fatalf("RevisionInstructions missing %q:\n%s", want, merged.RevisionInstructions)
		}
	}
}

func TestMergeCoverageIssuesIntoSaturationCheckAddsMediumContextOnlyWhenModelAlreadyNeedsRevision(t *testing.T) {
	check := ReviewSaturationCheck{
		SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
		Status:               ReviewSaturationStatusNeedsRevision,
		CheckedSummary:       "model found risk-1 incomplete",
		MissingRiskIDs:       []string{"risk-1"},
		RevisionInstructions: "Revise risk-1.",
	}
	issues := []CoverageIssue{{
		Kind:                CoverageIssueKindUnreflectedExternalEvidence,
		Severity:            CoverageIssueSeverityMedium,
		SurfaceIDs:          []string{"surface-1"},
		RevisionInstruction: "Weak external evidence is advisory context.",
	}}

	merged := MergeCoverageIssuesIntoSaturationCheck(check, issues)

	if merged.Status != ReviewSaturationStatusNeedsRevision {
		t.Fatalf("Status = %q, want %q", merged.Status, ReviewSaturationStatusNeedsRevision)
	}
	if got, want := strings.Join(merged.MissingSurfaceIDs, ","), ""; got != want {
		t.Fatalf("MissingSurfaceIDs = %q, want %q for medium advisory", got, want)
	}
	for _, want := range []string{
		"Revise risk-1.",
		"Deterministic coverage audit advisory context",
		"unreflected_external_evidence (medium)",
	} {
		if !strings.Contains(merged.RevisionInstructions, want) {
			t.Fatalf("RevisionInstructions missing %q:\n%s", want, merged.RevisionInstructions)
		}
	}
}

func TestMergeCoverageIssuesIntoSaturationCheckDedupesAndOrdersHighBeforeMediumContext(t *testing.T) {
	issues := []CoverageIssue{
		{
			Kind:                CoverageIssueKindUnreflectedExternalEvidence,
			Severity:            CoverageIssueSeverityMedium,
			SurfaceIDs:          []string{"surface-1"},
			RevisionInstruction: "Review weak external evidence context.",
		},
		{
			Kind:                CoverageIssueKindMissingCandidateRiskCoverage,
			Severity:            CoverageIssueSeverityHigh,
			RiskIDs:             []string{"risk-1"},
			RevisionInstruction: "Add risk-1 to scope coverage.",
		},
		{
			Kind:                CoverageIssueKindMissingCandidateRiskCoverage,
			Severity:            CoverageIssueSeverityHigh,
			RiskIDs:             []string{"risk-1"},
			RevisionInstruction: "Add risk-1 to scope coverage.",
		},
		{
			Kind:                CoverageIssueKindUnreflectedExternalEvidence,
			Severity:            CoverageIssueSeverityMedium,
			SurfaceIDs:          []string{"surface-1"},
			RevisionInstruction: "Review weak external evidence context.",
		},
	}

	merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)

	if merged.Status != ReviewSaturationStatusNeedsRevision {
		t.Fatalf("Status = %q, want %q", merged.Status, ReviewSaturationStatusNeedsRevision)
	}
	if got, want := strings.Join(merged.MissingRiskIDs, ","), "risk-1"; got != want {
		t.Fatalf("MissingRiskIDs = %q, want %q", got, want)
	}
	if strings.Count(merged.RevisionInstructions, "Add risk-1 to scope coverage.") != 1 {
		t.Fatalf("RevisionInstructions did not dedupe high issue:\n%s", merged.RevisionInstructions)
	}
	if strings.Count(merged.RevisionInstructions, "Review weak external evidence context.") != 1 {
		t.Fatalf("RevisionInstructions did not dedupe medium issue:\n%s", merged.RevisionInstructions)
	}
	highIndex := strings.Index(merged.RevisionInstructions, "missing_candidate_risk_coverage (high)")
	mediumIndex := strings.Index(merged.RevisionInstructions, "unreflected_external_evidence (medium)")
	if highIndex < 0 || mediumIndex < 0 || highIndex > mediumIndex {
		t.Fatalf("RevisionInstructions = %q, want high context before medium context", merged.RevisionInstructions)
	}
}

func TestMergeCoverageIssuesIntoSaturationCheckDoesNotOverrideBlockedCheck(t *testing.T) {
	blocked := ReviewSaturationCheck{
		SchemaVersion:  ReviewSaturationCheckSchemaVersionV1,
		Status:         ReviewSaturationStatusBlocked,
		CheckedSummary: "model could not check coverage",
	}
	issues := []CoverageIssue{{
		Kind:                CoverageIssueKindMissingCandidateRiskCoverage,
		Severity:            CoverageIssueSeverityHigh,
		RiskIDs:             []string{"risk-1"},
		RevisionInstruction: "Add risk-1.",
	}}

	merged := MergeCoverageIssuesIntoSaturationCheck(blocked, issues)
	if merged.Status != ReviewSaturationStatusBlocked || merged.RevisionInstructions != "" || len(merged.MissingRiskIDs) != 0 {
		t.Fatalf("MergeCoverageIssuesIntoSaturationCheck() = %#v, want original blocked check", merged)
	}
}
