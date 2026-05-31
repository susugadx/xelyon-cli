package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
