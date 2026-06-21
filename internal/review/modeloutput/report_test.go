package modeloutput_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestFinalizeReportRejectsCleanReportWithBlockedTrustedProbe(t *testing.T) {
	tests := []struct {
		name            string
		status          domain.ReviewProbeStatus
		mutatedWorktree bool
	}{
		{name: "blocked", status: domain.ReviewProbeBlocked},
		{name: "timed out", status: domain.ReviewProbeTimedOut},
		{name: "mutated worktree", status: domain.ReviewProbeMutatedWorktree},
		{name: "mutated worktree flag", status: domain.ReviewProbeFailed, mutatedWorktree: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
				Report: newCleanReportForModelOutputTest(),
				Plan:   newProbePlanForModelOutputTest("probe-1"),
				TrustedProbeSummaries: []reviewreport.ReviewProbeSummary{
					{
						ProbeID:         "probe-1",
						Mode:            domain.ReviewProbeHostReadOnly,
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
				Mode:    domain.ReviewProbeHostReadOnly,
				Status:  domain.ReviewProbeBlocked,
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
			Mode:            domain.ReviewProbeHostReadOnly,
			Status:          domain.ReviewProbeFailed,
			MutatedFiles:    []string{repoFile, probeFile},
			OutputTruncated: true,
			Error:           "raw paths " + repoFile + " " + probeFile,
			Commands: []reviewreport.ReviewProbeCommandSummary{
				{
					Command:         "cat " + probeFile,
					Args:            []string{repoFile, probeFile},
					WorkDir:         probeRoot,
					Status:          domain.ReviewProbeFailed,
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
				Mode:    domain.ReviewProbeHostReadOnly,
				Status:  domain.ReviewProbeBlocked,
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
