package modeloutput_test

import (
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestFinalizeReportModelOutputConvertsProviderDTOFinding(t *testing.T) {
	model := newReviewReportModelOutputForModelOutputTest(reviewreport.ReviewVerdictHasFindings, reviewreport.ReviewVerificationVerified)
	model.Summary = "One finding."
	model.SuggestedFindings = []reviewreport.ReviewReportModelSuggestedFinding{
		{
			ID:                   "finding-1",
			Severity:             reviewreport.ReviewReportModelSeverityP1,
			Status:               reviewreport.ReviewReportModelFindingConfirmed,
			Confidence:           reviewreport.ReviewReportModelConfidenceHigh,
			Title:                "Runner drops scope coverage",
			AffectedBehavior:     "Final review can omit a material risk.",
			CausalChain:          "The conversion path failed to connect risk-1 to finding-1.",
			EvidenceRefs:         []reviewreport.ReviewEvidenceRef{{Kind: reviewreport.ReviewEvidenceKindFile, Path: "internal/review/modeloutput/report.go", Line: 1}},
			RemediationDirection: "Convert suggested finding IDs into scope coverage finding links.",
		},
	}
	model.ScopeCoverage = &reviewreport.ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
			{SurfaceID: "surface-1", Status: reviewreport.ReviewReportImpactSurfaceChecked, Summary: "surface-1 checked."},
		},
		ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
			{RiskID: "risk-1", Status: reviewreport.ReviewReportCandidateRiskFinding, Summary: "risk-1 became finding-1.", FindingIDs: []string{"finding-1"}},
		},
	}
	data := mustMarshalJSONForModelOutputTest(t, model)

	got, err := reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content: string(data),
		Plan:    newNoProbePlanForModelOutputTest(),
	})
	if err != nil {
		t.Fatalf("FinalizeReportModelOutput() error = %v, want nil", err)
	}
	if got.SchemaVersion != reviewreport.ReviewReportSchemaVersionV2 {
		t.Fatalf("SchemaVersion = %q, want final review report schema", got.SchemaVersion)
	}
	if len(got.RootCauseGroups) != 1 || len(got.RootCauseGroups[0].Findings) != 1 {
		t.Fatalf("RootCauseGroups = %#v, want one converted group and finding", got.RootCauseGroups)
	}
	if got.RootCauseGroups[0].Severity != reviewreport.ReviewGroupSeverityHigh {
		t.Fatalf("group severity = %q, want high", got.RootCauseGroups[0].Severity)
	}
	if got.RootCauseGroups[0].Findings[0].ID != "finding-1" {
		t.Fatalf("finding ID = %q, want finding-1", got.RootCauseGroups[0].Findings[0].ID)
	}
}

func TestFinalizeReportModelOutputKeepsCoverageGapsOutOfFindings(t *testing.T) {
	model := newReviewReportModelOutputForModelOutputTest(reviewreport.ReviewVerdictBlocked, reviewreport.ReviewVerificationBlockedOrInconclusive)
	model.Summary = "Review blocked by missing evidence."
	model.CoverageGaps = []reviewreport.ReviewReportModelCoverageGap{
		{
			Surface:          "surface-1",
			Reason:           reviewreport.ReviewReportModelGapMissingEvidence,
			RecommendedCheck: "Run focused report validation tests.",
		},
	}
	model.ScopeCoverage = &reviewreport.ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
			{SurfaceID: "surface-1", Status: reviewreport.ReviewReportImpactSurfaceUnverified, Summary: "surface-1 lacks evidence."},
		},
		ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
			{RiskID: "risk-1", Status: reviewreport.ReviewReportCandidateRiskUnverified, Summary: "risk-1 lacks evidence."},
		},
	}
	data := mustMarshalJSONForModelOutputTest(t, model)

	got, err := reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content: string(data),
		Plan:    newNoProbePlanForModelOutputTest(),
	})
	if err != nil {
		t.Fatalf("FinalizeReportModelOutput() error = %v, want nil", err)
	}
	if len(got.RootCauseGroups) != 0 {
		t.Fatalf("RootCauseGroups = %#v, want no findings from coverage gap", got.RootCauseGroups)
	}
	if len(got.UnverifiedSurfaces) != 1 || got.UnverifiedSurfaces[0].SurfaceID != "surface-1" {
		t.Fatalf("UnverifiedSurfaces = %#v, want coverage gap projected as unverified surface", got.UnverifiedSurfaces)
	}
	if got.ComputedSummary == nil || got.ComputedSummary.UnverifiedSurfaceCount != 1 || got.ComputedSummary.UnverifiedRiskCount != 1 {
		t.Fatalf("ComputedSummary = %#v, want unverified surface and risk counts from scope_coverage", got.ComputedSummary)
	}
}

func TestFinalizeReportModelOutputRejectsProviderCoverageGapForCheckedScope(t *testing.T) {
	model := newReviewReportModelOutputForModelOutputTest(reviewreport.ReviewVerdictBlocked, reviewreport.ReviewVerificationBlockedOrInconclusive)
	model.Summary = "Review blocked by missing evidence."
	model.CoverageGaps = []reviewreport.ReviewReportModelCoverageGap{
		{
			Surface:          "surface-1",
			Reason:           reviewreport.ReviewReportModelGapMissingEvidence,
			RecommendedCheck: "Run focused report validation tests.",
		},
	}
	model.ScopeCoverage = &reviewreport.ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
			{SurfaceID: "surface-1", Status: reviewreport.ReviewReportImpactSurfaceChecked, Summary: "surface-1 checked."},
		},
		ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
			{RiskID: "risk-1", Status: reviewreport.ReviewReportCandidateRiskUnverified, Summary: "risk-1 lacks evidence."},
		},
	}
	data := mustMarshalJSONForModelOutputTest(t, model)

	_, err := reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content: string(data),
		Plan:    newNoProbePlanForModelOutputTest(),
	})
	if err == nil || !strings.Contains(err.Error(), "coverage_gaps") {
		t.Fatalf("FinalizeReportModelOutput() error = %v, want coverage gap checked scope rejection", err)
	}
}

func TestFinalizeReportModelOutputRejectsCleanProviderCoverageGaps(t *testing.T) {
	model := newReviewReportModelOutputForModelOutputTest(reviewreport.ReviewVerdictClean, reviewreport.ReviewVerificationVerified)
	model.Summary = "No findings."
	model.CoverageGaps = []reviewreport.ReviewReportModelCoverageGap{
		{
			Surface:          "surface-1",
			Reason:           reviewreport.ReviewReportModelGapMissingEvidence,
			RecommendedCheck: "Run focused report validation tests.",
		},
	}
	data := mustMarshalJSONForModelOutputTest(t, model)

	_, err := reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content: string(data),
		Plan:    newNoProbePlanForModelOutputTest(),
	})
	if err == nil || !strings.Contains(err.Error(), "coverage_gaps") {
		t.Fatalf("FinalizeReportModelOutput() error = %v, want clean coverage gap rejection", err)
	}
}

func TestFinalizeReportModelOutputValidatesProbeCommandRefsAfterTrustedSummaryInjection(t *testing.T) {
	probeRef := reviewreport.ReviewEvidenceRef{
		Kind:         reviewreport.ReviewEvidenceKindProbeCommand,
		ProbeID:      "probe-1",
		CommandIndex: reviewreport.ReviewCommandIndex(0),
	}
	model := newReviewReportModelOutputForModelOutputTest(reviewreport.ReviewVerdictHasFindings, reviewreport.ReviewVerificationVerified)
	model.Summary = "Probe-backed finding."
	model.SuggestedFindings = []reviewreport.ReviewReportModelSuggestedFinding{
		{
			ID:                   "finding-1",
			Severity:             reviewreport.ReviewReportModelSeverityP1,
			Status:               reviewreport.ReviewReportModelFindingConfirmed,
			Confidence:           reviewreport.ReviewReportModelConfidenceHigh,
			Title:                "Probe-backed regression",
			AffectedBehavior:     "The runner would reject probe-backed findings before trusted summaries are injected.",
			CausalChain:          "Provider DTO validation used model-supplied probe_summaries as the probe ref source of truth.",
			EvidenceRefs:         []reviewreport.ReviewEvidenceRef{probeRef},
			RemediationDirection: "Validate provider DTO evidence ref shape before finalizing against runner-owned trusted probes.",
		},
	}
	model.ScopeCoverage = &reviewreport.ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
			{
				SurfaceID:    "surface-1",
				Status:       reviewreport.ReviewReportImpactSurfaceChecked,
				Summary:      "surface-1 checked by probe-1.",
				EvidenceRefs: []reviewreport.ReviewEvidenceRef{probeRef},
			},
		},
		ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
			{
				RiskID:     "risk-1",
				Status:     reviewreport.ReviewReportCandidateRiskFinding,
				Summary:    "risk-1 became finding-1.",
				FindingIDs: []string{"finding-1"},
			},
		},
	}
	data := mustMarshalJSONForModelOutputTest(t, model)

	got, err := reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content:               string(data),
		Plan:                  newProbePlanForModelOutputTest("probe-1"),
		TrustedProbeSummaries: newPassedTrustedProbeSummariesForModelOutputTest("probe-1"),
	})
	if err != nil {
		t.Fatalf("FinalizeReportModelOutput() error = %v, want nil", err)
	}
	if len(got.ProbeSummaries) != 1 || got.ProbeSummaries[0].ProbeID != "probe-1" {
		t.Fatalf("ProbeSummaries = %#v, want runner-owned trusted probe summary", got.ProbeSummaries)
	}
	if len(got.RootCauseGroups) != 1 || len(got.RootCauseGroups[0].Findings) != 1 {
		t.Fatalf("RootCauseGroups = %#v, want converted probe-backed finding", got.RootCauseGroups)
	}
	gotRef := got.RootCauseGroups[0].Findings[0].EvidenceRefs[0]
	if gotRef.Kind != reviewreport.ReviewEvidenceKindProbeCommand || gotRef.ProbeID != "probe-1" || gotRef.CommandIndex == nil || *gotRef.CommandIndex != 0 {
		t.Fatalf("converted finding evidence ref = %#v, want probe_command probe-1 command 0", gotRef)
	}
}

func newReviewReportModelOutputForModelOutputTest(verdict reviewreport.ReviewVerdict, status reviewreport.ReviewVerificationStatus) reviewreport.ReviewReportModelOutput {
	return reviewreport.ReviewReportModelOutput{
		SchemaVersion:             reviewreport.ReviewReportModelSchemaVersionV2,
		TargetKind:                domain.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: status,
		Verdict:                   verdict,
		Summary:                   "Review summary.",
		SuggestedFindings:         []reviewreport.ReviewReportModelSuggestedFinding{},
		CoverageGaps:              []reviewreport.ReviewReportModelCoverageGap{},
		ProbeSummaries:            []reviewreport.ReviewProbeSummary{},
		ScopeCoverage: &reviewreport.ReviewReportScopeCoverage{
			ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
				{SurfaceID: "surface-1", Status: reviewreport.ReviewReportImpactSurfaceChecked, Summary: "surface-1 checked."},
			},
			ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
				{RiskID: "risk-1", Status: reviewreport.ReviewReportCandidateRiskDismissed, Summary: "risk-1 dismissed."},
			},
		},
	}
}

func newPassedTrustedProbeSummariesForModelOutputTest(probeID string) []reviewreport.ReviewProbeSummary {
	return []reviewreport.ReviewProbeSummary{
		{
			ProbeID: probeID,
			Mode:    domain.ReviewProbeHostReadOnly,
			Status:  domain.ReviewProbePassed,
			Commands: []reviewreport.ReviewProbeCommandSummary{
				{Command: "go test ./internal/review", Status: domain.ReviewProbePassed},
			},
		},
	}
}
