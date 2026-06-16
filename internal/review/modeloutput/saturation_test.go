package modeloutput_test

import (
	"strings"
	"testing"

	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestFinalizeSaturationCheckModelOutputRejectsMalformedJSONWithRunnerPrefix(t *testing.T) {
	_, err := reviewmodeloutput.FinalizeSaturationCheckModelOutput(reviewmodeloutput.SaturationCheckModelOutputInput{
		Content:         "{not-json",
		Plan:            newNoProbePlanForModelOutputTest(),
		FinalizedReport: newCleanReportForModelOutputTest(),
	})
	if err == nil {
		t.Fatal("FinalizeSaturationCheckModelOutput() error = nil, want decode error")
	}
	for _, want := range []string{"review runner decode saturation check", "decode review saturation check"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("FinalizeSaturationCheckModelOutput() error = %q, want %q", err.Error(), want)
		}
	}
}

func TestFinalizeSaturationCheckModelOutputRejectsUnknownExternalDocSnippetRefWithRunnerPrefix(t *testing.T) {
	docs := newExternalDocsForModelOutputTest()
	ref := newExternalDocEvidenceRefForModelOutputTest(docs)
	ref.SnippetID = "external-doc-1-snippet-missing"
	check := reviewreport.ReviewSaturationCheck{
		SchemaVersion:  reviewreport.ReviewSaturationCheckSchemaVersionV1,
		Status:         reviewreport.ReviewSaturationStatusNeedsRevision,
		CheckedSummary: "A file-backed candidate was not represented in the finalized report.",
		AdditionalFindingCandidates: []reviewreport.ReviewSaturationAdditionalFindingCandidate{
			{
				Summary:      "A report-pass finding candidate is grounded in external docs.",
				EvidenceRefs: []reviewreport.ReviewEvidenceRef{ref},
				Reason:       "The candidate uses fetched external documentation evidence.",
			},
		},
		RevisionInstructions: "Revise the report to include or explicitly dismiss the external doc-backed candidate.",
	}

	_, err := reviewmodeloutput.FinalizeSaturationCheckModelOutput(reviewmodeloutput.SaturationCheckModelOutputInput{
		Content:         string(mustMarshalJSONForModelOutputTest(t, check)),
		Plan:            newNoProbePlanForModelOutputTest(),
		FinalizedReport: newCleanReportForModelOutputTest(),
		ExternalDocs:    docs,
	})
	if err == nil {
		t.Fatal("FinalizeSaturationCheckModelOutput() error = nil, want unknown external_doc snippet error")
	}
	for _, want := range []string{"review runner decode saturation check", "unknown fetched external_doc snippet"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("FinalizeSaturationCheckModelOutput() error = %q, want %q", err.Error(), want)
		}
	}
}

func TestFinalizeSaturationCheckModelOutputRejectsMismatchedExternalDocSnippetMetadata(t *testing.T) {
	docs := newExternalDocsForModelOutputTest()
	ref := newExternalDocEvidenceRefForModelOutputTest(docs)
	ref.ContentHash = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
	check := reviewreport.ReviewSaturationCheck{
		SchemaVersion:  reviewreport.ReviewSaturationCheckSchemaVersionV1,
		Status:         reviewreport.ReviewSaturationStatusNeedsRevision,
		CheckedSummary: "A file-backed candidate was not represented in the finalized report.",
		AdditionalFindingCandidates: []reviewreport.ReviewSaturationAdditionalFindingCandidate{
			{
				Summary:      "A report-pass finding candidate is grounded in external docs.",
				EvidenceRefs: []reviewreport.ReviewEvidenceRef{ref},
				Reason:       "The candidate uses fetched external documentation evidence.",
			},
		},
		RevisionInstructions: "Revise the report to include or explicitly dismiss the external doc-backed candidate.",
	}

	_, err := reviewmodeloutput.FinalizeSaturationCheckModelOutput(reviewmodeloutput.SaturationCheckModelOutputInput{
		Content:         string(mustMarshalJSONForModelOutputTest(t, check)),
		Plan:            newNoProbePlanForModelOutputTest(),
		FinalizedReport: newCleanReportForModelOutputTest(),
		ExternalDocs:    docs,
	})
	if err == nil {
		t.Fatal("FinalizeSaturationCheckModelOutput() error = nil, want external_doc metadata mismatch")
	}
	for _, want := range []string{"review runner decode saturation check", "content_hash does not match fetched external_doc snippet hash"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("FinalizeSaturationCheckModelOutput() error = %q, want %q", err.Error(), want)
		}
	}
}
