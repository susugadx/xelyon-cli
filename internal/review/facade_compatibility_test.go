package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var _ ReviewProbeExecutor = (*ProbeRunner)(nil)
var _ ReviewExternalDocFetcher = (*HTTPReviewExternalDocFetcher)(nil)

func TestReviewFacadeReportAndArtifactCompatibility(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	reportJSON, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal(report) error = %v", err)
	}

	decoded, err := DecodeReviewReportJSON(reportJSON)
	if err != nil {
		t.Fatalf("DecodeReviewReportJSON() error = %v", err)
	}
	if err := ValidateReviewReport(decoded); err != nil {
		t.Fatalf("ValidateReviewReport() error = %v", err)
	}
	if err := ValidateReviewReportAgainstProbePlan(decoded, newNoProbeReviewProbePlanForTest(), nil); err != nil {
		t.Fatalf("ValidateReviewReportAgainstProbePlan() error = %v", err)
	}

	checkJSON := mustMarshalReviewSaturationCheckForTest(t, newSaturatedReviewSaturationCheckForTest())
	if _, err := DecodeReviewSaturationCheckJSON(checkJSON, newNoProbeReviewProbePlanForTest(), decoded); err != nil {
		t.Fatalf("DecodeReviewSaturationCheckJSON() error = %v", err)
	}

	buffer := NewBufferedReviewRunArtifactWriter()
	if err := buffer.WriteReviewRunArtifact("report.json", reportJSON); err != nil {
		t.Fatalf("BufferedReviewRunArtifactWriter.WriteReviewRunArtifact() error = %v", err)
	}
	if buffer.Len() != 1 {
		t.Fatalf("BufferedReviewRunArtifactWriter.Len() = %d, want 1", buffer.Len())
	}

	dir := filepath.Join(t.TempDir(), "artifacts")
	writer, err := NewReviewRunDirectoryArtifactWriter(dir)
	if err != nil {
		t.Fatalf("NewReviewRunDirectoryArtifactWriter() error = %v", err)
	}
	if err := buffer.FlushTo(writer); err != nil {
		t.Fatalf("BufferedReviewRunArtifactWriter.FlushTo() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(writer.Dir(), "report.json")); err != nil {
		t.Fatalf("facade artifact writer did not write report.json: %v", err)
	}
}

func TestReviewFacadeProbeCompatibility(t *testing.T) {
	plan := newValidReviewProbePlanForTest()
	planJSON, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal(plan) error = %v", err)
	}

	decoded, err := DecodeReviewProbePlanJSON(planJSON)
	if err != nil {
		t.Fatalf("DecodeReviewProbePlanJSON() error = %v", err)
	}
	if err := ValidateReviewProbePlan(decoded); err != nil {
		t.Fatalf("ValidateReviewProbePlan() error = %v", err)
	}
	if err := ValidateReviewProbePlanAgainstEvidence(decoded, newReviewProbePlanEvidenceBundleForFacadeTest()); err != nil {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v", err)
	}

	requests, err := BuildReviewProbeRequestsFromPlan(decoded)
	if err != nil {
		t.Fatalf("BuildReviewProbeRequestsFromPlan() error = %v", err)
	}
	if len(requests) != len(plan.Probes) {
		t.Fatalf("request count = %d, want %d", len(requests), len(plan.Probes))
	}
	if requests[0].ID != plan.Probes[0].ID || !reflect.DeepEqual(requests[0].Commands[0].Args, plan.Probes[0].Commands[0].Args) {
		t.Fatalf("converted request = %#v, want plan-derived request", requests[0])
	}

	_ = NewProbeRunner(t.TempDir())
}

func TestReviewFacadeExternalDocCompatibility(t *testing.T) {
	fetchReq := ReviewExternalDocFetchRequest{
		URL:   "https://docs.example.test/spec",
		DocID: "external-doc-1",
		FocusTerms: []ReviewExternalDocFocusTerm{
			{Term: "previous_response_id", Reason: "query focus"},
		},
	}
	doc := ReviewExternalDocEvidence{
		DocID:             fetchReq.DocID,
		URL:               fetchReq.URL,
		SourceCredibility: ReviewExternalDocSourceCredibilityUnknown,
		Snippets: []ReviewExternalDocSnippetEvidence{
			{SnippetID: "external-doc-1-snippet-1", ContentHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		},
	}

	if doc.SourceCredibility != ReviewExternalDocSourceCredibilityUnknown || doc.Snippets[0].SnippetID == "" {
		t.Fatalf("externaldoc aliases produced unexpected doc = %#v", doc)
	}
	if NewHTTPReviewExternalDocFetcher(nil) == nil {
		t.Fatal("NewHTTPReviewExternalDocFetcher() = nil")
	}
}

func newReviewProbePlanEvidenceBundleForFacadeTest() ReviewEvidenceBundle {
	repoRoot := "/tmp/repo"
	changedPath := filepath.Join(repoRoot, "internal", "review", "probe_plan_validate.go")
	return ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repoRoot,
		CWD:        repoRoot,
		ChangedFiles: []ReviewChangedFile{
			{Path: changedPath, Status: "M", Unstaged: true},
		},
		RelatedSearchHits: []ReviewRelatedSearchHit{
			{Path: filepath.Join(repoRoot, "internal", "review", "probe_plan_test.go"), Line: 1, Snippet: "ValidateReviewProbePlan", Reason: "focused validation coverage"},
		},
		Inventory: ReviewChangeInventory{
			Production: []string{changedPath},
		},
	}
}
