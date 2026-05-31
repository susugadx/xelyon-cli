package finalcheck

import (
	"strings"
	"testing"
)

func TestBuildFailureResult_FeedbackAndFingerprint(t *testing.T) {
	result := BuildFailureResult("go test ./...", 2, "compile failed", " main.go | 2 +-")
	if !result.NeedsContinue {
		t.Fatal("NeedsContinue = false, want true")
	}
	if !strings.Contains(result.Feedback, "[SYSTEM] Final check failed") {
		t.Fatalf("Feedback missing system marker: %q", result.Feedback)
	}
	if !strings.Contains(result.Feedback, "exit code 2") {
		t.Fatalf("Feedback missing exit code: %q", result.Feedback)
	}
	if !strings.Contains(result.Feedback, "main.go | 2 +-") {
		t.Fatalf("Feedback missing diff stat: %q", result.Feedback)
	}
	if result.FailureFingerprint == "" {
		t.Fatal("expected non-empty failure fingerprint")
	}
}

func TestTruncateOutput(t *testing.T) {
	input := strings.Repeat("x", 2001)
	got := TruncateOutput(input)
	wantSuffix := "\n... (truncated)"
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("TruncateOutput() = %q, want suffix %q", got, wantSuffix)
	}
	if len(got) != 2000+len(wantSuffix) {
		t.Fatalf("len(TruncateOutput()) = %d, want %d", len(got), 2000+len(wantSuffix))
	}
}

func TestFailureFingerprint_NormalizesOutput(t *testing.T) {
	first := FailureFingerprint("go test", 1, "\x1b[31mError\x1b[0m: fail\n")
	second := FailureFingerprint("go test", 1, "Error:   fail")
	if first != second {
		t.Fatalf("FailureFingerprint normalization mismatch: first=%q second=%q", first, second)
	}
}
