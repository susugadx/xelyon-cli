package lsp

import "testing"

// ===== CheckDiagnosticsForFiles Tests =====

func TestCheckDiagnosticsForFiles_NoLSP(t *testing.T) {
	result := CheckDiagnosticsForFiles([]string{"test.go", "main.go"})

	if result.HasErrors {
		t.Error("expected HasErrors=false when LSP is unavailable")
	}
	if result.ErrorCount != 0 {
		t.Errorf("expected ErrorCount=0, got %d", result.ErrorCount)
	}
	if result.WarnCount != 0 {
		t.Errorf("expected WarnCount=0, got %d", result.WarnCount)
	}
	if result.Summary != "" {
		t.Errorf("expected empty Summary, got %q", result.Summary)
	}
}

func TestCheckDiagnosticsForFilesWithClient_NoLSP(t *testing.T) {
	result := CheckDiagnosticsForFilesWithClient(nil, []string{"test.go"})

	if result.HasErrors {
		t.Error("expected HasErrors=false when explicit client is nil")
	}
	if result.ErrorCount != 0 || result.WarnCount != 0 {
		t.Errorf("expected zero counts, got errors=%d warnings=%d", result.ErrorCount, result.WarnCount)
	}
	if result.Summary != "" {
		t.Errorf("expected empty Summary, got %q", result.Summary)
	}
}

func TestCheckDiagnosticsForFiles_EmptyFiles(t *testing.T) {
	result := CheckDiagnosticsForFiles([]string{})

	if result.HasErrors {
		t.Error("expected HasErrors=false for empty file list")
	}
	if result.ErrorCount != 0 || result.WarnCount != 0 {
		t.Errorf("expected zero counts, got errors=%d warnings=%d", result.ErrorCount, result.WarnCount)
	}
}

func TestDiagnosticCheckResult_ZeroValue(t *testing.T) {
	// ゼロ値が「問題なし」を表すことを確認
	var result DiagnosticCheckResult

	if result.HasErrors {
		t.Error("zero value HasErrors should be false")
	}
	if result.ErrorCount != 0 {
		t.Error("zero value ErrorCount should be 0")
	}
	if result.WarnCount != 0 {
		t.Error("zero value WarnCount should be 0")
	}
	if result.Summary != "" {
		t.Error("zero value Summary should be empty")
	}
}

func TestGetDiagnosticsSummaryWithClient_NoLSP(t *testing.T) {
	if got := GetDiagnosticsSummaryWithClient(nil, "main.go"); got != "" {
		t.Fatalf("expected empty summary with nil client, got %q", got)
	}
}

func TestGetDiagnosticsSummary_NoLSP(t *testing.T) {
	if got := GetDiagnosticsSummary("main.go"); got != "" {
		t.Fatalf("expected empty summary without explicit client, got %q", got)
	}
}
