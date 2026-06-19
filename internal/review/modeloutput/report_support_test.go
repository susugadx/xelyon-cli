package modeloutput_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func newProbePlanForModelOutputTest(ids ...string) reviewprobe.ReviewProbePlan {
	probes := make([]reviewprobe.ReviewPlannedProbe, 0, len(ids))
	for _, id := range ids {
		probes = append(probes, reviewprobe.ReviewPlannedProbe{
			ID:         id,
			SurfaceIDs: []string{"surface-1"},
			RiskIDs:    []string{"risk-1"},
			Purpose:    "Confirm or falsify risk-1 for surface-1 with focused review checks.",
			Mode:       reviewprobe.ReviewProbeHostReadOnly,
			Commands: []reviewprobe.ReviewPlannedProbeCommand{
				{Command: "go", Args: []string{"test", "./internal/review"}, WorkDir: "."},
			},
			TimeoutSeconds: 30,
			MaxOutputBytes: 4096,
		})
	}
	return reviewprobe.ReviewProbePlan{
		SchemaVersion: reviewprobe.ReviewProbePlanSchemaVersionV2,
		TargetKind:    reviewprobe.TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []reviewprobe.ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Runner orchestration may need verification.",
				Category:        reviewprobe.ReviewProbeImpactSurfaceChangedFile,
				EvidenceSummary: "Evidence references current review changes at internal/review/runner.go.",
				Status:          reviewprobe.ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Run the planned probes in order.",
			},
		},
		CandidateRisks: []reviewprobe.ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "A runner contract could regress.",
				Severity:             reviewprobe.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Runner tests cover probe orchestration.",
				VerificationStrategy: "Execute the focused runner probe.",
				Status:               reviewprobe.ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: probes,
	}
}

func newNoProbePlanForModelOutputTest() reviewprobe.ReviewProbePlan {
	plan := newProbePlanForModelOutputTest()
	plan.ImpactSurfaces[0].Status = reviewprobe.ReviewProbeImpactSurfaceChecked
	plan.ImpactSurfaces[0].Reason = "Existing evidence covers surface-1."
	plan.CandidateRisks[0].Status = reviewprobe.ReviewProbeCandidateRiskCheckedByEvidence
	plan.CandidateRisks[0].VerificationStrategy = "No probe is needed."
	plan.Probes = []reviewprobe.ReviewPlannedProbe{}
	plan.NoProbeReason = "surface-1 and risk-1 are checked by existing evidence."
	return plan
}

func newCleanReportForModelOutputTest() reviewreport.ReviewReport {
	return reviewreport.ReviewReport{
		SchemaVersion:             reviewreport.ReviewReportSchemaVersionV2,
		TargetKind:                reviewreport.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: reviewreport.ReviewVerificationVerified,
		Verdict:                   reviewreport.ReviewVerdictClean,
		ScopeCoverage:             newCleanScopeCoverageForModelOutputTest(),
	}
}

func newBlockedReportForModelOutputTest() reviewreport.ReviewReport {
	report := newCleanReportForModelOutputTest()
	report.OverallVerificationStatus = reviewreport.ReviewVerificationBlockedOrInconclusive
	report.Verdict = reviewreport.ReviewVerdictBlocked
	report.Summary = "Review blocked by probe execution."
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = reviewreport.ReviewReportImpactSurfaceUnverified
	report.ScopeCoverage.ReviewedCandidateRisks[0].Status = reviewreport.ReviewReportCandidateRiskUnverified
	return report
}

func newHasFindingsReportForModelOutputTest() reviewreport.ReviewReport {
	report := newCleanReportForModelOutputTest()
	report.Verdict = reviewreport.ReviewVerdictHasFindings
	report.OverallVerificationStatus = reviewreport.ReviewVerificationVerified
	report.RootCauseGroups = []reviewreport.ReviewRootCauseGroup{
		{
			ID:                 "rc-1",
			Title:              "test group",
			Severity:           reviewreport.ReviewGroupSeverityLow,
			VerificationStatus: reviewreport.ReviewVerificationVerified,
			FixStrategy:        "fix root cause",
			VerificationPlan:   []string{"run focused validation"},
			Findings: []reviewreport.ReviewFinding{
				{
					ID:    "finding-1",
					Title: "finding",
					EvidenceRefs: []reviewreport.ReviewEvidenceRef{
						{Kind: reviewreport.ReviewEvidenceKindFile, Path: "internal/review/report_validation.go", Line: 1},
					},
				},
			},
		},
	}
	return report
}

func newPlanAwareHasFindingsReportForModelOutputTest() reviewreport.ReviewReport {
	report := newHasFindingsReportForModelOutputTest()
	report.ScopeCoverage = &reviewreport.ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
			{SurfaceID: "surface-1", Status: reviewreport.ReviewReportImpactSurfaceChecked, Summary: "surface-1 was checked."},
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
	return report
}

func newCleanScopeCoverageForModelOutputTest() *reviewreport.ReviewReportScopeCoverage {
	return &reviewreport.ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
			{
				SurfaceID: "surface-1",
				Status:    reviewreport.ReviewReportImpactSurfaceChecked,
				Summary:   "surface-1 was checked.",
			},
		},
		ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
			{
				RiskID:  "risk-1",
				Status:  reviewreport.ReviewReportCandidateRiskDismissed,
				Summary: "risk-1 was dismissed.",
			},
		},
	}
}

func newExternalDocsForModelOutputTest() []externaldoc.Evidence {
	fetchedAt := time.Date(2026, time.May, 31, 12, 0, 0, 0, time.UTC)
	return []externaldoc.Evidence{
		{
			DocID:                   "external-doc-1",
			URL:                     "https://docs.example.test/spec",
			SourceDomain:            "docs.example.test",
			SourceCredibility:       externaldoc.SourceCredibilityUnknown,
			SourceCredibilityReason: "unknown: test fixture",
			FetchedAt:               fetchedAt,
			StatusCode:              200,
			ContentType:             "text/html",
			ContentHash:             "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			Snippets: []externaldoc.SnippetEvidence{
				{
					SnippetID:   "external-doc-1-snippet-1",
					Content:     "External spec text.",
					ContentHash: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
					FocusTerm:   "External",
					FocusReason: "query focus",
				},
			},
		},
	}
}

func newExternalDocEvidenceRefForModelOutputTest(docs []externaldoc.Evidence) reviewreport.ReviewEvidenceRef {
	doc := docs[0]
	snippet := doc.Snippets[0]
	return reviewreport.ReviewEvidenceRef{
		Kind:        reviewreport.ReviewEvidenceKindExternalDoc,
		DocID:       doc.DocID,
		SnippetID:   snippet.SnippetID,
		URL:         doc.URL,
		FetchedAt:   doc.FetchedAt.Format(time.RFC3339Nano),
		ContentHash: snippet.ContentHash,
	}
}

func mustMarshalJSONForModelOutputTest(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}

func assertComputedSummaryForModelOutputTest(t *testing.T, got *reviewreport.ReviewReportComputedSummary, want reviewreport.ReviewReportComputedSummary) {
	t.Helper()

	if got == nil {
		t.Fatal("ComputedSummary = nil, want runner computed summary")
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("computed summary mismatch:\n got  = %#v\n want = %#v", *got, want)
	}
}

func cloneProbeSummariesForModelOutputTest(summaries []reviewreport.ReviewProbeSummary) []reviewreport.ReviewProbeSummary {
	cloned := make([]reviewreport.ReviewProbeSummary, len(summaries))
	for i, summary := range summaries {
		cloned[i] = summary
		cloned[i].MutatedFiles = append([]string(nil), summary.MutatedFiles...)
		cloned[i].Commands = make([]reviewreport.ReviewProbeCommandSummary, len(summary.Commands))
		for j, command := range summary.Commands {
			cloned[i].Commands[j] = command
			cloned[i].Commands[j].Args = append([]string(nil), command.Args...)
		}
	}
	return cloned
}

type replacementForModelOutputTest struct {
	old string
	new string
}

type replacingRedactorForModelOutputTest struct {
	replacements []replacementForModelOutputTest
}

func newReplacingRedactorForModelOutputTest(replacements ...replacementForModelOutputTest) replacingRedactorForModelOutputTest {
	sort.SliceStable(replacements, func(i, j int) bool {
		return len(replacements[i].old) > len(replacements[j].old)
	})
	return replacingRedactorForModelOutputTest{replacements: replacements}
}

func (r replacingRedactorForModelOutputTest) RedactText(text string) string {
	redacted := text
	for _, replacement := range r.replacements {
		redacted = strings.ReplaceAll(redacted, replacement.old, replacement.new)
	}
	return redacted
}

func (r replacingRedactorForModelOutputTest) RedactTexts(values []string) []string {
	redacted := make([]string, 0, len(values))
	for _, value := range values {
		redacted = append(redacted, r.RedactText(value))
	}
	return redacted
}

func (r replacingRedactorForModelOutputTest) RedactPath(path string) string {
	return r.RedactText(path)
}

func (r replacingRedactorForModelOutputTest) RedactPaths(paths []string) []string {
	redacted := make([]string, 0, len(paths))
	for _, path := range paths {
		redacted = append(redacted, r.RedactPath(path))
	}
	return redacted
}
