package modeloutput_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestFinalizeReportRejectsCleanReportWithBlockedTrustedProbe(t *testing.T) {
	tests := []struct {
		name            string
		status          reviewreport.ReviewProbeStatus
		mutatedWorktree bool
	}{
		{name: "blocked", status: reviewreport.ReviewProbeBlocked},
		{name: "timed out", status: reviewreport.ReviewProbeTimedOut},
		{name: "mutated worktree", status: reviewreport.ReviewProbeMutatedWorktree},
		{name: "mutated worktree flag", status: reviewreport.ReviewProbeFailed, mutatedWorktree: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
				Report: newCleanReportForModelOutputTest(),
				Plan:   newProbePlanForModelOutputTest("probe-1"),
				TrustedProbeSummaries: []reviewreport.ReviewProbeSummary{
					{
						ProbeID:         "probe-1",
						Mode:            reviewreport.ReviewProbeHostReadOnly,
						Status:          tt.status,
						MutatedWorktree: tt.mutatedWorktree,
					},
				},
			})
			if err == nil {
				t.Fatal("FinalizeReport() error = nil, want clean trusted probe rejection")
			}
			for _, want := range []string{"finalize report", `verdict "clean"`} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("FinalizeReport() error = %q, want %q", err.Error(), want)
				}
			}
		})
	}
}

func TestFinalizeReportDowngradesVerifiedFindingsWithBlockedTrustedProbe(t *testing.T) {
	report := newHasFindingsReportForModelOutputTest()
	report.ScopeCoverage = &reviewreport.ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
			{SurfaceID: "surface-1", Status: reviewreport.ReviewReportImpactSurfaceUnverified},
		},
		ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
			{RiskID: "risk-1", Status: reviewreport.ReviewReportCandidateRiskFinding, FindingIDs: []string{"finding-1"}},
		},
	}

	got, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report: report,
		Plan:   newProbePlanForModelOutputTest("probe-1"),
		TrustedProbeSummaries: []reviewreport.ReviewProbeSummary{
			{
				ProbeID: "probe-1",
				Mode:    reviewreport.ReviewProbeHostReadOnly,
				Status:  reviewreport.ReviewProbeBlocked,
			},
		},
	})
	if err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
	if got.Verdict != reviewreport.ReviewVerdictHasFindings {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, reviewreport.ReviewVerdictHasFindings)
	}
	if got.OverallVerificationStatus != reviewreport.ReviewVerificationPartiallyVerified {
		t.Fatalf("OverallVerificationStatus = %q, want %q", got.OverallVerificationStatus, reviewreport.ReviewVerificationPartiallyVerified)
	}
}

func TestFinalizeReportInjectsRedactedTrustedProbeSummaries(t *testing.T) {
	repoFile := "/tmp/review-runner/repo/internal/review/runner.go"
	probeRoot := "/tmp/review-runner/probe-root"
	probeFile := probeRoot + "/raw-output.txt"
	trustedSummaries := []reviewreport.ReviewProbeSummary{
		{
			ProbeID:         "probe-raw",
			Mode:            reviewreport.ReviewProbeHostReadOnly,
			Status:          reviewreport.ReviewProbeFailed,
			MutatedFiles:    []string{repoFile, probeFile},
			OutputTruncated: true,
			Error:           "raw paths " + repoFile + " " + probeFile,
			Commands: []reviewreport.ReviewProbeCommandSummary{
				{
					Command:         "cat " + probeFile,
					Args:            []string{repoFile, probeFile},
					WorkDir:         probeRoot,
					Status:          reviewreport.ReviewProbeFailed,
					ExitCode:        1,
					OutputTruncated: true,
					Error:           "failed at " + probeFile,
					DurationMs:      25,
				},
			},
		},
	}
	original := cloneProbeSummariesForModelOutputTest(trustedSummaries)
	redactor := newReplacingRedactorForModelOutputTest(
		replacementForModelOutputTest{old: repoFile, new: "internal/review/runner.go"},
		replacementForModelOutputTest{old: "/tmp/review-runner/repo", new: "<repo_root>"},
		replacementForModelOutputTest{old: probeRoot, new: "<probe_workdir>"},
	)

	got, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report:                newBlockedReportForModelOutputTest(),
		Plan:                  newProbePlanForModelOutputTest("probe-raw"),
		TrustedProbeSummaries: trustedSummaries,
		Redactor:              redactor,
	})
	if err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(trustedSummaries, original) {
		t.Fatalf("FinalizeReport() mutated trusted summaries:\ngot  %#v\nwant %#v", trustedSummaries, original)
	}
	if strings.Contains(got.ProbeSummaries[0].Error, "/tmp/review-runner") {
		t.Fatalf("ProbeSummaries[0].Error leaked raw path: %q", got.ProbeSummaries[0].Error)
	}
	wantMutatedFiles := []string{"internal/review/runner.go", "<probe_workdir>/raw-output.txt"}
	if !reflect.DeepEqual(got.ProbeSummaries[0].MutatedFiles, wantMutatedFiles) {
		t.Fatalf("MutatedFiles = %#v, want %#v", got.ProbeSummaries[0].MutatedFiles, wantMutatedFiles)
	}
	if got.ProbeSummaries[0].Commands[0].Command != "cat <probe_workdir>/raw-output.txt" {
		t.Fatalf("command Command = %q, want redacted command", got.ProbeSummaries[0].Commands[0].Command)
	}
	wantArgs := []string{"internal/review/runner.go", "<probe_workdir>/raw-output.txt"}
	if !reflect.DeepEqual(got.ProbeSummaries[0].Commands[0].Args, wantArgs) {
		t.Fatalf("command Args = %#v, want %#v", got.ProbeSummaries[0].Commands[0].Args, wantArgs)
	}
	if got.ProbeSummaries[0].Commands[0].WorkDir != "<probe_workdir>" {
		t.Fatalf("command WorkDir = %q, want redacted workdir", got.ProbeSummaries[0].Commands[0].WorkDir)
	}
	if got.ProbeSummaries[0].Commands[0].Error != "failed at <probe_workdir>/raw-output.txt" {
		t.Fatalf("command Error = %q, want redacted error", got.ProbeSummaries[0].Commands[0].Error)
	}
}

func TestFinalizeReportKeepsEmptyTrustedProbeSummariesNil(t *testing.T) {
	got, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report: newCleanReportForModelOutputTest(),
		Plan:   newProbePlanForModelOutputTest("probe-1"),
	})
	if err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
	if got.ProbeSummaries != nil {
		t.Fatalf("ProbeSummaries = %#v, want nil", got.ProbeSummaries)
	}
}

func TestFinalizeReportModelOutputRejectsComputedSummary(t *testing.T) {
	report := newCleanReportForModelOutputTest()
	report.ComputedSummary = &reviewreport.ReviewReportComputedSummary{FindingCount: 99}
	data := mustMarshalJSONForModelOutputTest(t, report)

	_, err := reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content: string(data),
		Plan:    newProbePlanForModelOutputTest("probe-1"),
	})
	if err == nil {
		t.Fatal("FinalizeReportModelOutput() error = nil, want computed_summary rejection")
	}
	for _, want := range []string{"review runner decode report", "computed_summary"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("FinalizeReportModelOutput() error = %q, want %q", err.Error(), want)
		}
	}
}

func TestFinalizeReportComputesSummaryForCleanReport(t *testing.T) {
	got, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report: newCleanReportForModelOutputTest(),
		Plan:   newProbePlanForModelOutputTest("probe-1"),
	})
	if err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
	assertComputedSummaryForModelOutputTest(t, got.ComputedSummary, reviewreport.ReviewReportComputedSummary{
		CheckedSurfaceCount: 1,
		CandidateRiskCount:  1,
		DismissedRiskCount:  1,
	})
}

func TestFinalizeReportComputesSummaryForFindingRisk(t *testing.T) {
	got, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report: newPlanAwareHasFindingsReportForModelOutputTest(),
		Plan:   newProbePlanForModelOutputTest("probe-1"),
	})
	if err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
	assertComputedSummaryForModelOutputTest(t, got.ComputedSummary, reviewreport.ReviewReportComputedSummary{
		RootCauseGroupCount: 1,
		FindingCount:        1,
		CheckedSurfaceCount: 1,
		CandidateRiskCount:  1,
		FindingRiskCount:    1,
	})
}

func TestFinalizeReportComputesBlockedProbeCount(t *testing.T) {
	got, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report: newBlockedReportForModelOutputTest(),
		Plan:   newProbePlanForModelOutputTest("probe-1"),
		TrustedProbeSummaries: []reviewreport.ReviewProbeSummary{
			{
				ProbeID: "probe-1",
				Mode:    reviewreport.ReviewProbeHostReadOnly,
				Status:  reviewreport.ReviewProbeBlocked,
			},
		},
	})
	if err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
	assertComputedSummaryForModelOutputTest(t, got.ComputedSummary, reviewreport.ReviewReportComputedSummary{
		UnverifiedSurfaceCount: 1,
		CandidateRiskCount:     1,
		UnverifiedRiskCount:    1,
		ProbeCount:             1,
		BlockedProbeCount:      1,
	})
}

func TestFinalizeReportOverwritesPreexistingComputedSummary(t *testing.T) {
	report := newCleanReportForModelOutputTest()
	report.ComputedSummary = &reviewreport.ReviewReportComputedSummary{
		FindingCount:              99,
		MutatedWorktreeProbeCount: 99,
	}

	got, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report: report,
		Plan:   newProbePlanForModelOutputTest("probe-1"),
	})
	if err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
	assertComputedSummaryForModelOutputTest(t, got.ComputedSummary, reviewreport.ReviewReportComputedSummary{
		CheckedSurfaceCount: 1,
		CandidateRiskCount:  1,
		DismissedRiskCount:  1,
	})
}

func TestFinalizeReportAllowsFetchedExternalDocSnippetRef(t *testing.T) {
	docs := newExternalDocsForModelOutputTest()
	report := newCleanReportForModelOutputTest()
	ref := newExternalDocEvidenceRefForModelOutputTest(docs)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}

	if _, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report:       report,
		Plan:         newNoProbePlanForModelOutputTest(),
		ExternalDocs: docs,
	}); err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
}

func TestFinalizeReportExternalDocRefIgnoresCredibilityMetadata(t *testing.T) {
	docs := newExternalDocsForModelOutputTest()
	docs[0].SourceCredibility = externaldoc.SourceCredibilityThirdParty
	docs[0].SourceCredibilityReason = "third_party: test metadata"
	report := newCleanReportForModelOutputTest()
	ref := newExternalDocEvidenceRefForModelOutputTest(docs)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}

	if _, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report:       report,
		Plan:         newNoProbePlanForModelOutputTest(),
		ExternalDocs: docs,
	}); err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
}

func TestFinalizeReportRejectsUnknownExternalDocSnippetRef(t *testing.T) {
	docs := newExternalDocsForModelOutputTest()
	report := newCleanReportForModelOutputTest()
	ref := newExternalDocEvidenceRefForModelOutputTest(docs)
	ref.SnippetID = "external-doc-1-snippet-missing"
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}

	_, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report:       report,
		Plan:         newNoProbePlanForModelOutputTest(),
		ExternalDocs: docs,
	})
	if err == nil {
		t.Fatal("FinalizeReport() error = nil, want unknown external_doc snippet error")
	}
	for _, want := range []string{"review runner finalize report", "unknown fetched external_doc snippet"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("FinalizeReport() error = %q, want %q", err.Error(), want)
		}
	}
}

func TestFinalizeReportRejectsMismatchedExternalDocSnippetMetadata(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*reviewreport.ReviewEvidenceRef)
		wantErr string
	}{
		{
			name: "url",
			mutate: func(ref *reviewreport.ReviewEvidenceRef) {
				ref.URL = "https://docs.example.test/other"
			},
			wantErr: "url does not match fetched external_doc URL",
		},
		{
			name: "fetched_at",
			mutate: func(ref *reviewreport.ReviewEvidenceRef) {
				ref.FetchedAt = time.Date(2026, time.May, 31, 13, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
			},
			wantErr: "fetched_at does not match fetched external_doc timestamp",
		},
		{
			name: "content_hash",
			mutate: func(ref *reviewreport.ReviewEvidenceRef) {
				ref.ContentHash = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
			},
			wantErr: "content_hash does not match fetched external_doc snippet hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := newExternalDocsForModelOutputTest()
			report := newCleanReportForModelOutputTest()
			ref := newExternalDocEvidenceRefForModelOutputTest(docs)
			tt.mutate(&ref)
			report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}
			report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}

			_, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
				Report:       report,
				Plan:         newNoProbePlanForModelOutputTest(),
				ExternalDocs: docs,
			})
			if err == nil {
				t.Fatal("FinalizeReport() error = nil, want external_doc metadata mismatch")
			}
			for _, want := range []string{"review runner finalize report", tt.wantErr} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("FinalizeReport() error = %q, want %q", err.Error(), want)
				}
			}
		})
	}
}

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
